package domain

import (
	"errors"
	"fmt"

	"github.com/mwildt/jansvca/service/internal/eventstore"
)

// Component is a single dependency within a project, identified by a
// coordinate (component identifier) and a concrete version.
type Component struct {
	Component string
	Version   string
}

// Project is the aggregate root for a tracked project.
type Project struct {
	ID          string
	Name        string
	Description string
	Components  map[string]Component
	Deleted     bool

	// version is the aggregate's current event version (for optimistic
	// concurrency on append).
	version eventstore.Version
}

// ErrProjectDeleted is returned when acting on a soft-deleted project.
var ErrProjectDeleted = errors.New("projekte: project is deleted")

// ErrComponentExists is returned when adding an already-present component.
var ErrComponentExists = errors.New("projekte: component already exists")

// ErrComponentNotFound is returned when a component is not present.
var ErrComponentNotFound = errors.New("projekte: component not found")

// Apply applies an envelope's payload to the aggregate, updating state and
// advancing the aggregate version. Unknown event types are ignored.
func (p *Project) Apply(env eventstore.Envelope) error {
	switch env.Type {
	case EventProjectCreated:
		var e ProjectCreated
		if err := decode(env, &e); err != nil {
			return err
		}
		p.ID = e.ProjectID
		p.Name = e.Name
		p.Description = e.Description
		p.Components = map[string]Component{}
	case EventProjectRenamed:
		var e ProjectRenamed
		if err := decode(env, &e); err != nil {
			return err
		}
		p.Name = e.Name
	case EventProjectDescriptionChanged:
		var e ProjectDescriptionChanged
		if err := decode(env, &e); err != nil {
			return err
		}
		p.Description = e.Description
	case EventProjectDeleted:
		p.Deleted = true
	case EventComponentAdded:
		var e ComponentAdded
		if err := decode(env, &e); err != nil {
			return err
		}
		if p.Components == nil {
			p.Components = map[string]Component{}
		}
		p.Components[e.Component] = Component{Component: e.Component, Version: e.Version}
	case EventComponentRemoved:
		var e ComponentRemoved
		if err := decode(env, &e); err != nil {
			return err
		}
		delete(p.Components, e.Component)
	case EventComponentVersionUpdated:
		var e ComponentVersionUpdated
		if err := decode(env, &e); err != nil {
			return err
		}
		if _, ok := p.Components[e.Component]; ok {
			p.Components[e.Component] = Component{Component: e.Component, Version: e.Version}
		}
	}
	p.version = env.Version
	return nil
}

// Version returns the aggregate's current event version.
func (p *Project) Version() eventstore.Version { return p.version }

// CreateProject produces the events for creating a new project.
func CreateProject(id, name, description string) ([]eventstore.PayloadEvent, error) {
	if id == "" {
		return nil, errors.New("projekte: project id is required")
	}
	if name == "" {
		return nil, errors.New("projekte: project name is required")
	}
	return []eventstore.PayloadEvent{ProjectCreated{
		ProjectID:   id,
		Name:        name,
		Description: description,
	}}, nil
}

// Rename produces a rename event after validation against current state.
func (p *Project) Rename(name string) ([]eventstore.PayloadEvent, error) {
	if p == nil || p.ID == "" {
		return nil, errors.New("projekte: project not found")
	}
	if p.Deleted {
		return nil, ErrProjectDeleted
	}
	if name == "" {
		return nil, errors.New("projekte: project name is required")
	}
	return []eventstore.PayloadEvent{ProjectRenamed{ProjectID: p.ID, Name: name}}, nil
}

// ChangeDescription produces a description-changed event.
func (p *Project) ChangeDescription(description string) ([]eventstore.PayloadEvent, error) {
	if p == nil || p.ID == "" {
		return nil, errors.New("projekte: project not found")
	}
	if p.Deleted {
		return nil, ErrProjectDeleted
	}
	return []eventstore.PayloadEvent{ProjectDescriptionChanged{
		ProjectID:   p.ID,
		Description: description,
	}}, nil
}

// Delete produces a soft-delete event.
func (p *Project) Delete() ([]eventstore.PayloadEvent, error) {
	if p == nil || p.ID == "" {
		return nil, errors.New("projekte: project not found")
	}
	if p.Deleted {
		return nil, ErrProjectDeleted
	}
	return []eventstore.PayloadEvent{ProjectDeleted{ProjectID: p.ID}}, nil
}

// AddComponent adds a new component to the project.
func (p *Project) AddComponent(component, version string) ([]eventstore.PayloadEvent, error) {
	if p == nil || p.ID == "" {
		return nil, errors.New("projekte: project not found")
	}
	if p.Deleted {
		return nil, ErrProjectDeleted
	}
	if component == "" {
		return nil, errors.New("projekte: component is required")
	}
	if version == "" {
		return nil, errors.New("projekte: component version is required")
	}
	if _, ok := p.Components[component]; ok {
		return nil, fmt.Errorf("%w: %s", ErrComponentExists, component)
	}
	return []eventstore.PayloadEvent{ComponentAdded{
		ProjectID: p.ID,
		Component: component,
		Version:   version,
	}}, nil
}

// RemoveComponent removes a component (soft-delete of the component).
func (p *Project) RemoveComponent(component string) ([]eventstore.PayloadEvent, error) {
	if p == nil || p.ID == "" {
		return nil, errors.New("projekte: project not found")
	}
	if p.Deleted {
		return nil, ErrProjectDeleted
	}
	if _, ok := p.Components[component]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrComponentNotFound, component)
	}
	return []eventstore.PayloadEvent{ComponentRemoved{
		ProjectID: p.ID,
		Component: component,
	}}, nil
}

// UpdateComponentVersion changes the version of an existing component.
func (p *Project) UpdateComponentVersion(component, version string) ([]eventstore.PayloadEvent, error) {
	if p == nil || p.ID == "" {
		return nil, errors.New("projekte: project not found")
	}
	if p.Deleted {
		return nil, ErrProjectDeleted
	}
	if version == "" {
		return nil, errors.New("projekte: component version is required")
	}
	if _, ok := p.Components[component]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrComponentNotFound, component)
	}
	return []eventstore.PayloadEvent{ComponentVersionUpdated{
		ProjectID: p.ID,
		Component: component,
		Version:   version,
	}}, nil
}

// ImportInput is a single component declaration to import (e.g. from an SBOM).
type ImportInput struct {
	Component string
	Version   string
}

// ImportComponents reconciles a set of components against the project state,
// emitting ComponentAdded for new components and ComponentVersionUpdated for
// components whose version changed. Components already present with the same
// version are skipped. The returned count is the number of emitted events.
func (p *Project) ImportComponents(inputs []ImportInput) ([]eventstore.PayloadEvent, error) {
	if p == nil || p.ID == "" {
		return nil, errors.New("projekte: project not found")
	}
	if p.Deleted {
		return nil, ErrProjectDeleted
	}
	seen := map[string]string{}
	var events []eventstore.PayloadEvent
	for _, in := range inputs {
		if in.Component == "" || in.Version == "" {
			continue
		}
		if prev, dup := seen[in.Component]; dup {
			if prev != in.Version {
				return nil, fmt.Errorf("projekte: conflicting versions for %s: %s vs %s", in.Component, prev, in.Version)
			}
			continue
		}
		seen[in.Component] = in.Version
		existing, ok := p.Components[in.Component]
		if !ok {
			events = append(events, ComponentAdded{
				ProjectID: p.ID,
				Component: in.Component,
				Version:   in.Version,
			})
			continue
		}
		if existing.Version != in.Version {
			events = append(events, ComponentVersionUpdated{
				ProjectID: p.ID,
				Component: in.Component,
				Version:   in.Version,
			})
		}
	}
	return events, nil
}

func decode(env eventstore.Envelope, e any) error {
	if len(env.Payload) == 0 {
		return nil
	}
	return jsonUnmarshal(env.Payload, e)
}
