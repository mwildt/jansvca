// Package store is the persistence adapter for the schwachstellen module.
// Vulnerabilities are stored as one JSON file per record under a data
// directory; a Bleve index provides full-text search, filtering, faceting and
// pagination; a secondary in-memory index maps each affected component to the
// vulnerability ids that reference it, enabling fast semver matching without
// scanning the whole corpus.
//
// The store is the single source of truth for the schwachstellen module: the
// former WAL event store is replaced by this file+index adapter. The store is
// safe for concurrent use.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

// Record is the persisted representation of a vulnerability. It is the
// canonical state written to <dataDir>/vulnerabilities/<id>.json.
type Record struct {
	ID          string          `json:"id"`
	Identifier  string          `json:"identifier"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	CVSS        float64         `json:"cvss"`
	Source      string          `json:"source,omitempty"`
	Ecosystems  []string        `json:"ecosystems,omitempty"`
	Affected    []AffectedRange `json:"affected,omitempty"`
	Deleted     bool            `json:"deleted,omitempty"`
}

// AffectedRange is one (component, version-range) pair impacted by a
// vulnerability. Ecosystem is the OSV ecosystem of the package, when known
// (e.g. "npm", "PyPI"); it is derived from the component (purl) or the OSV
// package and indexed for ecosystem filtering.
type AffectedRange struct {
	Component    string `json:"component"`
	VersionRange string `json:"version_range"`
	Ecosystem    string `json:"ecosystem,omitempty"`
}

// New creates and opens a store rooted at dataDir. The directory layout is:
//
//	dataDir/
//	  vulnerabilities/<id>.json   one file per record (canonical state)
//	  vuln-index/                 Bleve index (rebuildable from the files)
//
// On first open the Bleve index is rebuilt from existing files if it is empty
// or missing. A separate component index is rebuilt in memory from the files.
func New(dataDir string) (*Store, error) {
	vulnDir := filepath.Join(dataDir, "vulnerabilities")
	indexDir := filepath.Join(dataDir, "vuln-index")
	if err := os.MkdirAll(vulnDir, 0o755); err != nil {
		return nil, fmt.Errorf("store: create vuln dir: %w", err)
	}
	s := &Store{
		dataDir:  dataDir,
		vulnDir:  vulnDir,
		indexDir: indexDir,
		compIdx:  map[string]map[string]struct{}{},
	}
	if err := s.openIndex(); err != nil {
		return nil, err
	}
	if err := s.rebuildIndexFromFiles(); err != nil {
		return nil, err
	}
	return s, nil
}

// Store persists vulnerabilities as JSON files plus a Bleve search index and a
// component-to-vulnerability index.
type Store struct {
	mu       sync.RWMutex
	dataDir  string
	vulnDir  string
	indexDir string
	index    bleve.Index
	// compIdx maps component -> set of vulnerability IDs that reference it.
	compIdx map[string]map[string]struct{}
}

// Close releases the Bleve index.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index == nil {
		return nil
	}
	err := s.index.Close()
	s.index = nil
	return err
}

// openIndex opens an existing Bleve index at indexDir or creates a new one
// with the schwachstellen document mapping.
func (s *Store) openIndex() error {
	m := bleve.NewIndexMapping()
	dm := bleve.NewDocumentMapping()
	dm.AddFieldMappingsAt("title", textFieldMapping())
	dm.AddFieldMappingsAt("identifier", textFieldMapping())
	dm.AddFieldMappingsAt("description", textFieldMapping())
	dm.AddFieldMappingsAt("id", bleve.NewTextFieldMapping())
	dm.AddFieldMappingsAt("source", bleve.NewKeywordFieldMapping())
	dm.AddFieldMappingsAt("ecosystems", bleve.NewKeywordFieldMapping())
	dm.AddFieldMappingsAt("cvss", bleve.NewNumericFieldMapping())
	dm.AddFieldMappingsAt("deleted", bleve.NewBooleanFieldMapping())
	m.AddDocumentMapping("_default", dm)

	idx, err := bleve.Open(s.indexDir)
	if err != nil {
		if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) || strings.Contains(err.Error(), "index path does not exist") {
			idx, err = bleve.New(s.indexDir, m)
			if err != nil {
				return fmt.Errorf("store: create bleve index: %w", err)
			}
		} else {
			return fmt.Errorf("store: open bleve index: %w", err)
		}
	}
	s.index = idx
	return nil
}

func textFieldMapping() *mapping.FieldMapping {
	fm := bleve.NewTextFieldMapping()
	fm.Store = false
	return fm
}

// rebuildIndexFromFiles scans the vulnerabilities directory and ensures every
// non-deleted record is present in the Bleve index and the component index. It
// is idempotent and runs once at open time.
func (s *Store) rebuildIndexFromFiles() error {
	entries, err := os.ReadDir(s.vulnDir)
	if err != nil {
		return fmt.Errorf("store: read vuln dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		rec, err := s.readRecordFile(e.Name())
		if err != nil {
			continue
		}
		if rec.Deleted {
			continue
		}
		if err := s.index.Index(rec.ID, toDoc(rec)); err != nil {
			return fmt.Errorf("store: index %s: %w", rec.ID, err)
		}
		s.indexComponent(rec)
	}
	return nil
}

// filePath returns the absolute path of the JSON file for id.
func (s *Store) filePath(id string) string {
	return filepath.Join(s.vulnDir, id+".json")
}

// readRecordFile reads and decodes a record by its file name (e.g. "V1.json").
func (s *Store) readRecordFile(name string) (Record, error) {
	data, err := os.ReadFile(filepath.Join(s.vulnDir, name))
	if err != nil {
		return Record{}, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, fmt.Errorf("store: decode %s: %w", name, err)
	}
	return rec, nil
}

// Get loads a single non-deleted record by id. It returns nil, nil when the
// record does not exist or is deleted.
func (s *Store) Get(id string) (*Record, error) {
	data, err := os.ReadFile(s.filePath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("store: read %s: %w", id, err)
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("store: decode %s: %w", id, err)
	}
	if rec.Deleted {
		return nil, nil
	}
	return &rec, nil
}

// Put writes the record as a JSON file and updates the Bleve index and the
// component index. It is the single write path used by the command handler.
func (s *Store) Put(rec Record) error {
	if rec.ID == "" {
		return errors.New("store: record id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.writeRecord(rec); err != nil {
		return err
	}
	if rec.Deleted {
		if err := s.index.Delete(rec.ID); err != nil {
			return fmt.Errorf("store: bleve delete %s: %w", rec.ID, err)
		}
		s.dropComponent(rec.ID)
		return nil
	}
	if err := s.index.Index(rec.ID, toDoc(rec)); err != nil {
		return fmt.Errorf("store: bleve index %s: %w", rec.ID, err)
	}
	s.reindexComponent(rec)
	return nil
}

func (s *Store) writeRecord(rec Record) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("store: marshal %s: %w", rec.ID, err)
	}
	if err := os.WriteFile(s.filePath(rec.ID), data, 0o644); err != nil {
		return fmt.Errorf("store: write %s: %w", rec.ID, err)
	}
	return nil
}

// indexComponent adds all components of rec to the component index.
func (s *Store) indexComponent(rec Record) {
	for _, a := range rec.Affected {
		if a.Component == "" {
			continue
		}
		if s.compIdx[a.Component] == nil {
			s.compIdx[a.Component] = map[string]struct{}{}
		}
		s.compIdx[a.Component][rec.ID] = struct{}{}
	}
}

// reindexComponent replaces the component-index entries for rec with the
// components currently in rec.Affected.
func (s *Store) reindexComponent(rec Record) {
	s.dropComponent(rec.ID)
	s.indexComponent(rec)
}

// dropComponent removes rec.ID from every component bucket of the component
// index.
func (s *Store) dropComponent(id string) {
	for comp, ids := range s.compIdx {
		delete(ids, id)
		if len(ids) == 0 {
			delete(s.compIdx, comp)
		}
	}
}

// ComponentsFor returns the sorted list of vulnerability ids that reference the
// given component, or nil if none. Used by the matching query service.
func (s *Store) ComponentsFor(component string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids, ok := s.compIdx[component]
	if !ok || len(ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// QueryResult is one page of vulnerability records plus the total number of
// matching records across all pages.
type QueryResult struct {
	Items []Record `json:"items"`
	Total uint64   `json:"total"`
}

// Query runs a filtered, paginated search against the Bleve index. The
// returned items are full Record values loaded from the JSON files (Bleve only
// returns ids + facets). When query text is empty and no filter is set, all
// records are returned in id order via a match-all query.
func (s *Store) Query(q Query) (QueryResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	bq := s.buildQuery(q)
	size := q.PageSize
	if size <= 0 {
		size = 50
	}
	req := bleve.NewSearchRequestOptions(bq, size, q.from(), false)
	req.Fields = []string{"id"}
	res, err := s.index.Search(req)
	if err != nil {
		return QueryResult{}, fmt.Errorf("store: bleve search: %w", err)
	}
	out := QueryResult{Total: res.Total}
	if res.Hits != nil {
		out.Items = make([]Record, 0, len(res.Hits))
	}
	for _, hit := range res.Hits {
		id, _ := hit.Fields["id"].(string)
		if id == "" {
			id = hit.ID
		}
		rec, err := s.getLocked(id)
		if err != nil {
			return out, err
		}
		if rec == nil {
			continue
		}
		out.Items = append(out.Items, *rec)
	}
	return out, nil
}

// getLocked loads a record by id assuming the read lock is already held.
func (s *Store) getLocked(id string) (*Record, error) {
	data, err := os.ReadFile(s.filePath(id))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	if rec.Deleted {
		return nil, nil
	}
	return &rec, nil
}

// Query holds the optional filters and pagination for a vulnerability search.
type Query struct {
	Text       string
	Source     string
	Ecosystem  string
	MinCVSS    float64
	HasMinCVSS bool
	Page       int
	PageSize   int
}

// from is the zero-based Bleve offset derived from Page/PageSize.
func (q Query) from() int {
	if q.PageSize <= 0 {
		return 0
	}
	if q.Page <= 0 {
		return 0
	}
	return (q.Page - 1) * q.PageSize
}

// buildQuery constructs the Bleve query for a Query.
func (s *Store) buildQuery(q Query) query.Query {
	var qs []query.Query
	notDeleted := bleve.NewBoolFieldQuery(false)
	notDeleted.SetField("deleted")
	qs = append(qs, notDeleted)

	if q.Text != "" {
		tq := bleve.NewQueryStringQuery(q.Text)
		qs = append(qs, tq)
	}
	if q.Source != "" {
		sq := bleve.NewTermQuery(q.Source)
		sq.SetField("source")
		qs = append(qs, sq)
	}
	if q.Ecosystem != "" {
		sq := bleve.NewTermQuery(q.Ecosystem)
		sq.SetField("ecosystems")
		qs = append(qs, sq)
	}
	if q.HasMinCVSS {
		minIncl := true
		maxIncl := false
		nq := bleve.NewNumericRangeInclusiveQuery(&q.MinCVSS, nil, &minIncl, &maxIncl)
		nq.SetField("cvss")
		qs = append(qs, nq)
	}
	if len(qs) == 0 {
		return bleve.NewMatchAllQuery()
	}
	return bleve.NewConjunctionQuery(qs...)
}

// doc is the indexed representation of a record.
type doc struct {
	ID         string   `json:"id"`
	Identifier string   `json:"identifier"`
	Title      string   `json:"title"`
	Source     string   `json:"source"`
	Ecosystems []string `json:"ecosystems"`
	CVSS       float64  `json:"cvss"`
	Deleted    bool     `json:"deleted"`
}

func toDoc(rec Record) doc {
	return doc{
		ID:         rec.ID,
		Identifier: rec.Identifier,
		Title:      rec.Title,
		Source:     rec.Source,
		Ecosystems: rec.Ecosystems,
		CVSS:       rec.CVSS,
		Deleted:    rec.Deleted,
	}
}
