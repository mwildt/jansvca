// Package application contains the use cases, ports and read-model
// projections of the schwachstellen module.
package application

import (
	"errors"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/schwachstellen/domain"
)

// EventStore is the port for persisting and loading vulnerability streams.
type EventStore interface {
	CurrentVersion(id eventstore.StreamID) eventstore.Version
	Load(id eventstore.StreamID) ([]eventstore.Envelope, error)
	Append(id eventstore.StreamID, expected eventstore.Version, events []eventstore.PayloadEvent) ([]eventstore.Envelope, error)
}

// EventBus is the port for publishing recorded events.
type EventBus interface {
	Publish(env eventstore.Envelope)
}

type Repository struct {
	store EventStore
}

func NewRepository(store EventStore) *Repository {
	return &Repository{store: store}
}

func streamID(id string) eventstore.StreamID { return eventstore.StreamID("vulnerability:" + id) }

// Load rebuilds the vulnerability aggregate from its event stream.
func (r *Repository) Load(id string) (*domain.Vulnerability, error) {
	envs, err := r.store.Load(streamID(id))
	if err != nil {
		if errors.Is(err, eventstore.ErrStreamNotFound) {
			return &domain.Vulnerability{}, nil
		}
		return nil, err
	}
	v := &domain.Vulnerability{}
	for _, env := range envs {
		if err := v.Apply(env); err != nil {
			return nil, err
		}
	}
	return v, nil
}

// CommandHandler executes vulnerability commands.
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

func (h *CommandHandler) Create(id, identifier, title, description string, cvss float64) error {
	events, err := domain.CreateVulnerability(id, identifier, title, description, cvss)
	if err != nil {
		return err
	}
	return h.commit(id, 0, events)
}

func (h *CommandHandler) Update(id, title, description string, cvss float64) error {
	v, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := v.Update(title, description, cvss)
	if err != nil {
		return err
	}
	return h.commit(id, v.Version(), events)
}

func (h *CommandHandler) Delete(id string) error {
	v, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := v.Delete()
	if err != nil {
		return err
	}
	return h.commit(id, v.Version(), events)
}

func (h *CommandHandler) AddAffectedRange(id, component, versionRange string) error {
	v, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := v.AddAffectedRange(component, versionRange)
	if err != nil {
		return err
	}
	return h.commit(id, v.Version(), events)
}

func (h *CommandHandler) RemoveAffectedRange(id, component string) error {
	v, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	events, err := v.RemoveAffectedRange(component)
	if err != nil {
		return err
	}
	return h.commit(id, v.Version(), events)
}

// Import creates a vulnerability with its affected ranges in a single stream
// append. It is used by external importers (e.g. the osv.dev sync) to record a
// new vulnerability with all of its ranges atomically. If the stream already
// exists (version > 0) it is treated as an update via Reconcile instead.
type AffectedRangeInput = domain.AffectedRangeInput

func (h *CommandHandler) Import(id, identifier, title, description string, cvss float64, ranges []AffectedRangeInput) error {
	v, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	if v.ID == "" {
		events, err := domain.CreateVulnerability(id, identifier, title, description, cvss)
		if err != nil {
			return err
		}
		for _, r := range ranges {
			if r.Component == "" || r.VersionRange == "" {
				continue
			}
			events = append(events, domain.AffectedRangeAdded{
				VulnerabilityID: id,
				Component:       r.Component,
				VersionRange:    r.VersionRange,
			})
		}
		return h.commit(id, 0, events)
	}
	if v.Deleted {
		return domain.ErrVulnDeleted
	}
	events, err := v.Reconcile(title, description, cvss, ranges)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}
	return h.commit(id, v.Version(), events)
}

// Reconcile aligns an existing vulnerability's metadata and affected ranges
// with the desired state, emitting only the changes. It is a no-op when the
// vulnerability does not exist yet.
func (h *CommandHandler) Reconcile(id, title, description string, cvss float64, ranges []AffectedRangeInput) error {
	v, err := h.repo.Load(id)
	if err != nil {
		return err
	}
	if v.ID == "" || v.Deleted {
		return nil
	}
	events, err := v.Reconcile(title, description, cvss, ranges)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}
	return h.commit(id, v.Version(), events)
}
