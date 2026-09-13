// Package application contains the use cases (command handlers) and ports of
// the projekte module. It orchestrates domain logic against the event store
// and event bus ports, keeping the domain pure.
package application

import (
	"errors"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/projekte/domain"
)

// EventStore is the port for persisting and loading project event streams.
type EventStore interface {
	CurrentVersion(id eventstore.StreamID) eventstore.Version
	Load(id eventstore.StreamID) ([]eventstore.Envelope, error)
	Append(id eventstore.StreamID, expected eventstore.Version, events []eventstore.PayloadEvent) ([]eventstore.Envelope, error)
}

// EventBus is the port for publishing recorded events to other modules.
type EventBus interface {
	Publish(env eventstore.Envelope)
}

// Repository reconstructs a Project aggregate from its event stream.
type Repository struct {
	store EventStore
}

func NewRepository(store EventStore) *Repository {
	return &Repository{store: store}
}

// Load rebuilds the project aggregate from the store. Returns a zero-value
// Project (ID == "") and a nil error if the stream does not exist.
func (r *Repository) Load(id string) (*domain.Project, error) {
	envs, err := r.store.Load(streamID(id))
	if err != nil {
		if errors.Is(err, eventstore.ErrStreamNotFound) {
			return &domain.Project{}, nil
		}
		return nil, err
	}
	p := &domain.Project{}
	for _, env := range envs {
		if err := p.Apply(env); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func streamID(id string) eventstore.StreamID { return eventstore.StreamID("project:" + id) }

// CommandHandler executes commands, persists resulting events and publishes
// them on the bus.
type CommandHandler struct {
	repo  *Repository
	store EventStore
	bus   EventBus
}

func NewCommandHandler(store EventStore, bus EventBus) *CommandHandler {
	return &CommandHandler{repo: NewRepository(store), store: store, bus: bus}
}

func (h *CommandHandler) commit(id string, expected eventstore.Version, events []eventstore.PayloadEvent) error {
	recorded, err := h.store.Append(streamID(id), expected, events)
	if err != nil {
		return err
	}
	for _, env := range recorded {
		h.bus.Publish(env)
	}
	return nil
}

// CreateProject creates a new project.
func (h *CommandHandler) CreateProject(id, name, description string) error {
	events, err := domain.CreateProject(id, name, description)
	if err != nil {
		return err
	}
	return h.commit(id, 0, events)
}

// Rename renames an existing project.
func (h *CommandHandler) Rename(id, name string) error {
	p, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := p.Rename(name)
	if err != nil {
		return err
	}
	return h.commit(id, p.Version(), events)
}

// ChangeDescription updates the description of a project.
func (h *CommandHandler) ChangeDescription(id, description string) error {
	p, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := p.ChangeDescription(description)
	if err != nil {
		return err
	}
	return h.commit(id, p.Version(), events)
}

// Delete soft-deletes a project.
func (h *CommandHandler) Delete(id string) error {
	p, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := p.Delete()
	if err != nil {
		return err
	}
	return h.commit(id, p.Version(), events)
}

// AddComponent adds a component with a concrete version to a project.
func (h *CommandHandler) AddComponent(id, component, version string) error {
	p, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := p.AddComponent(component, version)
	if err != nil {
		return err
	}
	return h.commit(id, p.Version(), events)
}

// RemoveComponent removes a component from a project.
func (h *CommandHandler) RemoveComponent(id, component string) error {
	p, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := p.RemoveComponent(component)
	if err != nil {
		return err
	}
	return h.commit(id, p.Version(), events)
}

// UpdateComponentVersion changes the version of a component.
func (h *CommandHandler) UpdateComponentVersion(id, component, version string) error {
	p, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := p.UpdateComponentVersion(component, version)
	if err != nil {
		return err
	}
	return h.commit(id, p.Version(), events)
}

// ImportComponents reconciles a set of components against the project state,
// adding new components and updating changed versions. Inputs with empty
// component or version are skipped.
func (h *CommandHandler) ImportComponents(id string, inputs []domain.ImportInput) (int, error) {
	p, err := h.repo.Load(id)
	if err != nil {
		return 0, err
	}
	events, err := p.ImportComponents(inputs)
	if err != nil {
		return 0, err
	}
	if err := h.commit(id, p.Version(), events); err != nil {
		return 0, err
	}
	return len(events), nil
}
