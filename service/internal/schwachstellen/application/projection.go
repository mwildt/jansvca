package application

import (
	"encoding/json"
	"sort"
	"sync"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/projekte/domain"
	vulndomain "github.com/mwildt/jansvca/service/internal/schwachstellen/domain"
	"github.com/mwildt/jansvca/service/internal/semver"
)

// VulnerabilityView is the read-model representation of a vulnerability.
type VulnerabilityView struct {
	ID          string
	Identifier  string
	Title       string
	Description string
	CVSS        float64
	Affected    []AffectedRangeView
}

// AffectedRangeView is the read-model representation of an affected range.
type AffectedRangeView struct {
	Component    string
	VersionRange string
}

// Match is a single vulnerability hit for a project component.
type Match struct {
	ProjectID               string
	Component               string
	Version                 string
	VulnerabilityID         string
	VulnerabilityIdentifier string
	CVSS                    float64
}

// MatchingProjection builds the read model for vulnerabilities and project
// components and computes matches between them. It subscribes to both the
// schwachstellen and projekte event streams.
type MatchingProjection struct {
	mu         sync.Mutex
	vulns      map[string]*VulnerabilityView
	components map[string]map[string]string // projectID -> component -> version
}

// NewMatchingProjection creates an empty projection.
func NewMatchingProjection() *MatchingProjection {
	return &MatchingProjection{
		vulns:      map[string]*VulnerabilityView{},
		components: map[string]map[string]string{},
	}
}

// Apply applies a single envelope from either module's event stream.
func (m *MatchingProjection) Apply(env eventstore.Envelope) {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch env.Type {
	case domain.EventComponentAdded:
		var e domain.ComponentAdded
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if m.components[e.ProjectID] == nil {
			m.components[e.ProjectID] = map[string]string{}
		}
		m.components[e.ProjectID][e.Component] = e.Version
	case domain.EventComponentRemoved:
		var e domain.ComponentRemoved
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if comps, ok := m.components[e.ProjectID]; ok {
			delete(comps, e.Component)
		}
	case domain.EventComponentVersionUpdated:
		var e domain.ComponentVersionUpdated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if comps, ok := m.components[e.ProjectID]; ok {
			comps[e.Component] = e.Version
		}
	case domain.EventProjectDeleted:
		var e domain.ProjectDeleted
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		delete(m.components, e.ProjectID)

	case vulndomain.EventVulnerabilityCreated:
		var e vulndomain.VulnerabilityCreated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		m.vulns[e.VulnerabilityID] = &VulnerabilityView{
			ID:          e.VulnerabilityID,
			Identifier:  e.Identifier,
			Title:       e.Title,
			Description: e.Description,
			CVSS:        e.CVSS,
		}
	case vulndomain.EventVulnerabilityUpdated:
		var e vulndomain.VulnerabilityUpdated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if v, ok := m.vulns[e.VulnerabilityID]; ok {
			if e.Title != "" {
				v.Title = e.Title
			}
			if e.Description != "" {
				v.Description = e.Description
			}
			if e.CVSS != 0 {
				v.CVSS = e.CVSS
			}
		}
	case vulndomain.EventVulnerabilityDeleted:
		var e vulndomain.VulnerabilityDeleted
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		delete(m.vulns, e.VulnerabilityID)
	case vulndomain.EventAffectedRangeAdded:
		var e vulndomain.AffectedRangeAdded
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if v, ok := m.vulns[e.VulnerabilityID]; ok {
			v.Affected = append(v.Affected, AffectedRangeView{
				Component:    e.Component,
				VersionRange: e.VersionRange,
			})
		}
	case vulndomain.EventAffectedRangeRemoved:
		var e vulndomain.AffectedRangeRemoved
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if v, ok := m.vulns[e.VulnerabilityID]; ok {
			out := v.Affected[:0]
			for _, a := range v.Affected {
				if a.Component != e.Component {
					out = append(out, a)
				}
			}
			v.Affected = out
		}
	}
}

// Matches returns all vulnerability matches for a given project, sorted by
// CVSS descending then component.
func (m *MatchingProjection) Matches(projectID string) []Match {
	m.mu.Lock()
	defer m.mu.Unlock()
	comps, ok := m.components[projectID]
	if !ok {
		return nil
	}
	var out []Match
	for component, version := range comps {
		ver, err := semver.Parse(version)
		if err != nil {
			continue
		}
		for _, v := range m.vulns {
			for _, a := range v.Affected {
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
						VulnerabilityID:         v.ID,
						VulnerabilityIdentifier: v.Identifier,
						CVSS:                    v.CVSS,
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

// AllVulnerabilities returns all non-deleted vulnerabilities.
func (m *MatchingProjection) AllVulnerabilities() []VulnerabilityView {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]VulnerabilityView, 0, len(m.vulns))
	for _, v := range m.vulns {
		copyV := *v
		copyV.Affected = append([]AffectedRangeView(nil), v.Affected...)
		out = append(out, copyV)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
