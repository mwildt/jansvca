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
	"strconv"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

// filesPerRange is the maximum number of record files stored in a single
// range folder. Each range folder buckets 100 records so a single directory
// never holds more than filesPerRange entries, keeping directory listings fast
// at OSV scale (~200k records).
const filesPerRange = 100

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
//	  vulnerabilities/<ecosystem>/<year>/<range>/<id>.json   one file per record
//	  vuln-index/                  Bleve index (rebuildable from the files)
//
// Records are bucketed by ecosystem and year with at most filesPerRange files
// per range folder, keeping every directory small even for the full OSV
// corpus. The Bleve index is rebuilt from existing files when it is empty or
// missing. A separate component index is rebuilt in memory from the files.
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
		paths:    map[string]string{},
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
	// paths maps vulnerability id -> relative path under vulnDir
	// ("<ecosystem>/<year>/<range>/<id>.json") for O(1) file lookup. It is
	// rebuilt at open time from the file tree and kept in sync by Put.
	paths map[string]string
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

// rebuildIndexFromFiles walks the vulnerabilities directory tree and ensures
// every non-deleted record is present in the Bleve index and the component
// index. It also rebuilds the id->relative-path lookup map from the file tree.
// It is idempotent and runs once at open time. Each record's file location is
// recorded so Get can read it without scanning the tree.
func (s *Store) rebuildIndexFromFiles() error {
	err := filepath.WalkDir(s.vulnDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".json") {
			return nil
		}
		rec, err := s.readRecordPath(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(s.vulnDir, path)
		s.paths[rec.ID] = filepath.ToSlash(rel)
		if rec.Deleted {
			return nil
		}
		if err := s.index.Index(rec.ID, toDoc(rec)); err != nil {
			return fmt.Errorf("store: index %s: %w", rec.ID, err)
		}
		s.indexComponent(rec)
		return nil
	})
	if err != nil {
		return fmt.Errorf("store: walk vuln dir: %w", err)
	}
	return nil
}

// relPathFor computes the relative path under vulnDir where rec should be
// stored: <ecosystem>/<year>/<range>/<id>.json. Ecosystem defaults to "_" and
// year to "0" when they cannot be derived, so every record still lands in a
// stable, reproducible location. <range> is the bucket containing 100 records;
// it is derived from the per-ecosystem/year running count the store maintains.
func (s *Store) relPathFor(rec Record) string {
	eco := primaryEcosystem(rec)
	year := yearOf(rec)
	n := s.countIn(filepath.ToSlash(filepath.Join(eco, year)))
	rangeBucket := (n / filesPerRange) * filesPerRange
	return filepath.ToSlash(filepath.Join(eco, year, strconv.Itoa(rangeBucket), rec.ID+".json"))
}

// countIn returns the number of records already stored under the given
// relative subdirectory ("<ecosystem>/<year>"). It scans the in-memory path
// map so it stays O(total records) but only over the affected buckets.
func (s *Store) countIn(dir string) int {
	prefix := dir + "/"
	n := 0
	for _, p := range s.paths {
		if strings.HasPrefix(p, prefix) {
			n++
		}
	}
	return n
}

// primaryEcosystem returns the ecosystem used to file rec. It prefers the
// first entry of rec.Ecosystems, falling back to the first affected range's
// ecosystem, then "_" when none is known.
func primaryEcosystem(rec Record) string {
	if len(rec.Ecosystems) > 0 && rec.Ecosystems[0] != "" {
		return sanitizeSegment(rec.Ecosystems[0])
	}
	for _, a := range rec.Affected {
		if a.Ecosystem != "" {
			return sanitizeSegment(a.Ecosystem)
		}
	}
	return "_"
}

// yearOf extracts the 4-digit year from rec, preferring the OSV id prefix
// (e.g. "GO-2024-0123" -> "2024") and falling back to "0".
func yearOf(rec Record) string {
	for _, part := range strings.Split(rec.ID, "-") {
		if len(part) == 4 {
			if _, err := strconv.Atoi(part); err == nil {
				return part
			}
		}
	}
	return "0"
}

// sanitizeSegment replaces characters that are invalid in a path segment.
func sanitizeSegment(seg string) string {
	seg = strings.TrimSpace(seg)
	seg = strings.ReplaceAll(seg, string(filepath.Separator), "_")
	seg = strings.ReplaceAll(seg, ":", "_")
	if seg == "" {
		return "_"
	}
	return seg
}

// absPath returns the absolute filesystem path of the JSON file for id, using
// the in-memory path map. It returns "" when the id is unknown (the record
// has not been written yet).
func (s *Store) absPath(id string) string {
	rel, ok := s.paths[id]
	if !ok {
		return ""
	}
	return filepath.Join(s.vulnDir, filepath.FromSlash(rel))
}

// readRecordPath reads and decodes a record from an absolute file path.
func (s *Store) readRecordPath(path string) (Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return Record{}, fmt.Errorf("store: decode %s: %w", path, err)
	}
	return rec, nil
}

// Get loads a single non-deleted record by id. It returns nil, nil when the
// record does not exist or is deleted.
func (s *Store) Get(id string) (*Record, error) {
	s.mu.RLock()
	path := s.absPath(id)
	s.mu.RUnlock()
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
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
// The file location is derived from the record (ecosystem/year/range); when a
// record's ecosystem changes it is relocated and the old file removed.
func (s *Store) Put(rec Record) error {
	if rec.ID == "" {
		return errors.New("store: record id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	oldPath := s.absPath(rec.ID)
	if err := s.writeRecord(rec); err != nil {
		return err
	}
	if oldPath != "" && oldPath != s.absPath(rec.ID) {
		_ = os.Remove(oldPath)
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
	rel := s.relPathFor(rec)
	path := filepath.Join(s.vulnDir, filepath.FromSlash(rel))
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return fmt.Errorf("store: marshal %s: %w", rec.ID, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("store: create dir %s: %w", rec.ID, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("store: write %s: %w", rec.ID, err)
	}
	s.paths[rec.ID] = rel
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
	path := s.absPath(id)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
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
