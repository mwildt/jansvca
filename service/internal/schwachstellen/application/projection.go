package application

import (
	"sort"

	"github.com/mwildt/jansvca/service/internal/schwachstellen/store"
	"github.com/mwildt/jansvca/service/internal/semver"
)

// VulnerabilityView is the read-model representation of a vulnerability.
// Source records the origin ("manual" for REST-API entries, "osv" for
// osv.dev imports).
type VulnerabilityView struct {
	ID          string              `json:"id"`
	Identifier  string              `json:"identifier"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	CVSS        float64             `json:"cvss"`
	Source      string              `json:"source"`
	Ecosystems  []string            `json:"ecosystems"`
	Affected    []AffectedRangeView `json:"affected"`
}

// AffectedRangeView is the read-model representation of an affected range.
type AffectedRangeView struct {
	Component    string `json:"component"`
	VersionRange string `json:"version_range"`
	Ecosystem    string `json:"ecosystem,omitempty"`
}

// Match is a single vulnerability hit for a project component.
type Match struct {
	ProjectID               string  `json:"project_id"`
	Component               string  `json:"component"`
	Version                 string  `json:"version"`
	VulnerabilityID         string  `json:"vulnerability_id"`
	VulnerabilityIdentifier string  `json:"vulnerability_identifier"`
	CVSS                    float64 `json:"cvss"`
}

// VulnerabilityPage is a paginated slice of vulnerabilities plus the total
// number of matching records across all pages.
type VulnerabilityPage struct {
	Items []VulnerabilityView `json:"items"`
	Total uint64              `json:"total"`
}

// QueryService is the read side of the schwachstellen module. It serves
// vulnerability lookups, filtered/paginated search and project matching from
// the file-based store.
type QueryService struct {
	store *store.Store
	// projects is an optional read port for the project component model,
	// used by Matches. When nil, Matches returns an empty slice.
	projects ProjectComponents
}

// ProjectComponents is the read port for the project component model owned by
// the projekte module.
type ProjectComponents interface {
	Components(projectID string) map[string]string
}

// NewQueryService creates a query service over the given store. projects is
// optional; pass nil when matching is not required.
func NewQueryService(s *store.Store, projects ProjectComponents) *QueryService {
	return &QueryService{store: s, projects: projects}
}

// Get returns the read-model view for a single vulnerability by id, or nil if
// it does not exist (or was deleted).
func (q *QueryService) Get(id string) *VulnerabilityView {
	rec, err := q.store.Get(id)
	if err != nil || rec == nil {
		return nil
	}
	return recordToView(rec)
}

// Search runs a filtered, paginated vulnerability search via the Bleve index.
func (q *QueryService) Search(qry store.Query) (VulnerabilityPage, error) {
	res, err := q.store.Query(qry)
	if err != nil {
		return VulnerabilityPage{}, err
	}
	page := VulnerabilityPage{Total: res.Total, Items: make([]VulnerabilityView, 0, len(res.Items))}
	for i := range res.Items {
		page.Items = append(page.Items, *recordToView(&res.Items[i]))
	}
	return page, nil
}

// Matches returns all vulnerability matches for a given project, sorted by
// CVSS descending then component. It uses the component index to avoid
// scanning the whole corpus.
func (q *QueryService) Matches(projectID string) []Match {
	if q.projects == nil {
		return []Match{}
	}
	comps := q.projects.Components(projectID)
	if len(comps) == 0 {
		return []Match{}
	}
	out := make([]Match, 0)
	for component, version := range comps {
		ver, err := semver.Parse(version)
		if err != nil {
			continue
		}
		ids := q.store.ComponentsFor(component)
		for _, id := range ids {
			rec, err := q.store.Get(id)
			if err != nil || rec == nil {
				continue
			}
			for _, a := range rec.Affected {
				if a.Component != component {
					continue
				}
				r, err := semver.ParseRangeExpr(a.VersionRange)
				if err != nil {
					continue
				}
				if r.Matches(ver) {
					out = append(out, Match{
						ProjectID:               projectID,
						Component:               component,
						Version:                 version,
						VulnerabilityID:         rec.ID,
						VulnerabilityIdentifier: rec.Identifier,
						CVSS:                    rec.CVSS,
					})
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CVSS != out[j].CVSS {
			return out[i].CVSS > out[j].CVSS
		}
		return out[i].Component < out[j].Component
	})
	return out
}

// recordToView converts a store record to a read-model view.
func recordToView(rec *store.Record) *VulnerabilityView {
	v := &VulnerabilityView{
		ID:          rec.ID,
		Identifier:  rec.Identifier,
		Title:       rec.Title,
		Description: rec.Description,
		CVSS:        rec.CVSS,
		Source:      rec.Source,
		Ecosystems:  rec.Ecosystems,
		Affected:    make([]AffectedRangeView, 0, len(rec.Affected)),
	}
	for _, a := range rec.Affected {
		v.Affected = append(v.Affected, AffectedRangeView{
			Component:    a.Component,
			VersionRange: a.VersionRange,
			Ecosystem:    a.Ecosystem,
		})
	}
	return v
}
