// Package domain contains the pure domain model of the projekte module:
// aggregates, events and business rules. It has no I/O dependencies.
package domain

import "github.com/mwildt/jansvca/internal/eventstore"

// Event type discriminators.
const (
	EventProjectCreated            eventstore.EventType = "projekte.project_created"
	EventProjectRenamed            eventstore.EventType = "projekte.project_renamed"
	EventProjectDescriptionChanged eventstore.EventType = "projekte.project_description_changed"
	EventProjectDeleted            eventstore.EventType = "projekte.project_deleted"
	EventComponentAdded            eventstore.EventType = "projekte.component_added"
	EventComponentRemoved          eventstore.EventType = "projekte.component_removed"
	EventComponentVersionUpdated   eventstore.EventType = "projekte.component_version_updated"
)

// ProjectCreated is emitted when a new project is added.
type ProjectCreated struct {
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// EventType implements eventstore.PayloadEvent.
func (ProjectCreated) EventType() eventstore.EventType { return EventProjectCreated }

// ProjectRenamed is emitted when a project name changes.
type ProjectRenamed struct {
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
}

func (ProjectRenamed) EventType() eventstore.EventType { return EventProjectRenamed }

// ProjectDescriptionChanged is emitted when the description changes.
type ProjectDescriptionChanged struct {
	ProjectID   string `json:"project_id"`
	Description string `json:"description"`
}

func (ProjectDescriptionChanged) EventType() eventstore.EventType {
	return EventProjectDescriptionChanged
}

// ProjectDeleted is emitted for a soft-delete.
type ProjectDeleted struct {
	ProjectID string `json:"project_id"`
}

func (ProjectDeleted) EventType() eventstore.EventType { return EventProjectDeleted }

// ComponentAdded is emitted when a component is added to a project.
type ComponentAdded struct {
	ProjectID string `json:"project_id"`
	Component string `json:"component"`
	Version   string `json:"version"`
}

func (ComponentAdded) EventType() eventstore.EventType { return EventComponentAdded }

// ComponentRemoved is emitted when a component is removed (soft-delete).
type ComponentRemoved struct {
	ProjectID string `json:"project_id"`
	Component string `json:"component"`
}

func (ComponentRemoved) EventType() eventstore.EventType { return EventComponentRemoved }

// ComponentVersionUpdated is emitted when a component's version changes.
type ComponentVersionUpdated struct {
	ProjectID string `json:"project_id"`
	Component string `json:"component"`
	Version   string `json:"version"`
}

func (ComponentVersionUpdated) EventType() eventstore.EventType {
	return EventComponentVersionUpdated
}
