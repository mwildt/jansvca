package application

import (
	"encoding/json"
	"sync"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/projekte/domain"
)

// ProjectView is the read-model representation of a project (not deleted).
type ProjectView struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Components  []ComponentView `json:"components"`
}

// ComponentView is the read-model representation of a component.
type ComponentView struct {
	Component string `json:"component"`
	Version   string `json:"version"`
}

// ProjectProjection builds a read model of all non-deleted projects and their
// components from the projekte event stream.
type ProjectProjection struct {
	mu       sync.Mutex
	projects map[string]*ProjectView
}

// NewProjectProjection creates an empty projection.
func NewProjectProjection() *ProjectProjection {
	return &ProjectProjection{projects: map[string]*ProjectView{}}
}

// Apply updates the projection for a single envelope.
func (p *ProjectProjection) Apply(env eventstore.Envelope) {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch env.Type {
	case domain.EventProjectCreated:
		var e domain.ProjectCreated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		p.projects[e.ProjectID] = &ProjectView{
			ID:          e.ProjectID,
			Name:        e.Name,
			Description: e.Description,
			Components:  []ComponentView{},
		}
	case domain.EventProjectRenamed:
		var e domain.ProjectRenamed
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if v, ok := p.projects[e.ProjectID]; ok {
			v.Name = e.Name
		}
	case domain.EventProjectDescriptionChanged:
		var e domain.ProjectDescriptionChanged
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if v, ok := p.projects[e.ProjectID]; ok {
			v.Description = e.Description
		}
	case domain.EventProjectDeleted:
		var e domain.ProjectDeleted
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		delete(p.projects, e.ProjectID)
	case domain.EventComponentAdded:
		var e domain.ComponentAdded
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if v, ok := p.projects[e.ProjectID]; ok {
			v.Components = append(v.Components, ComponentView{Component: e.Component, Version: e.Version})
		}
	case domain.EventComponentRemoved:
		var e domain.ComponentRemoved
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if v, ok := p.projects[e.ProjectID]; ok {
			out := v.Components[:0]
			for _, c := range v.Components {
				if c.Component != e.Component {
					out = append(out, c)
				}
			}
			v.Components = out
		}
	case domain.EventComponentVersionUpdated:
		var e domain.ComponentVersionUpdated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return
		}
		if v, ok := p.projects[e.ProjectID]; ok {
			for i := range v.Components {
				if v.Components[i].Component == e.Component {
					v.Components[i].Version = e.Version
				}
			}
		}
	}
}

// All returns all non-deleted projects.
func (p *ProjectProjection) All() []ProjectView {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]ProjectView, 0, len(p.projects))
	for _, v := range p.projects {
		copyV := *v
		copyV.Components = append([]ComponentView(nil), v.Components...)
		out = append(out, copyV)
	}
	return out
}

// Get returns a single project by id, or nil if not found / deleted.
func (p *ProjectProjection) Get(id string) *ProjectView {
	p.mu.Lock()
	defer p.mu.Unlock()
	if v, ok := p.projects[id]; ok {
		copyV := *v
		copyV.Components = append([]ComponentView(nil), v.Components...)
		return &copyV
	}
	return nil
}

// Components returns the component->version map for a project, or nil if the
// project does not exist. It is the read port consumed by the schwachstellen
// matching query service.
func (p *ProjectProjection) Components(projectID string) map[string]string {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.projects[projectID]
	if !ok {
		return nil
	}
	out := make(map[string]string, len(v.Components))
	for _, c := range v.Components {
		out[c.Component] = c.Version
	}
	return out
}
