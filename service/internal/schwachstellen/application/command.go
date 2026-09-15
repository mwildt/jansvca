// Package application contains the use cases (command handler) and query
// service of the schwachstellen module. It orchestrates domain logic against
// the file-based store (JSON files + Bleve index), replacing the former
// event-sourced WAL.
package application

import (
	"errors"
	"sort"

	"github.com/mwildt/jansvca/service/internal/schwachstellen/domain"
	"github.com/mwildt/jansvca/service/internal/schwachstellen/store"
)

// AffectedRangeInput is the desired state of an affected range.
type AffectedRangeInput = domain.AffectedRangeInput

// CommandHandler executes vulnerability commands against the store.
type CommandHandler struct {
	store *store.Store
}

// NewCommandHandler creates a command handler backed by the given store.
func NewCommandHandler(s *store.Store) *CommandHandler {
	return &CommandHandler{store: s}
}

// Create records a new manual vulnerability.
func (h *CommandHandler) Create(id, identifier, title, description string, cvss float64) error {
	if _, err := domain.New(id, identifier, title, description, cvss, "manual"); err != nil {
		return err
	}
	existing, err := h.store.Get(id)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("schwachstellen: vulnerability already exists")
	}
	v, _ := domain.New(id, identifier, title, description, cvss, "manual")
	return h.store.Put(toRecord(v))
}

// Update mutates the metadata of an existing vulnerability.
func (h *CommandHandler) Update(id, title, description string, cvss float64) error {
	v, err := h.load(id)
	if err != nil {
		return err
	}
	if v == nil {
		return errors.New("schwachstellen: vulnerability not found")
	}
	updated, err := v.Update(title, description, cvss)
	if err != nil {
		return err
	}
	return h.store.Put(toRecord(updated))
}

// Delete soft-deletes a vulnerability.
func (h *CommandHandler) Delete(id string) error {
	v, err := h.load(id)
	if err != nil {
		return err
	}
	if v == nil {
		return errors.New("schwachstellen: vulnerability not found")
	}
	deleted, err := v.Delete()
	if err != nil {
		return err
	}
	return h.store.Put(toRecord(deleted))
}

// AddAffectedRange adds a component version range to a vulnerability.
func (h *CommandHandler) AddAffectedRange(id, component, versionRange string) error {
	v, err := h.load(id)
	if err != nil {
		return err
	}
	if v == nil {
		return errors.New("schwachstellen: vulnerability not found")
	}
	updated, err := v.AddAffectedRange(component, versionRange)
	if err != nil {
		return err
	}
	return h.store.Put(toRecord(updated))
}

// RemoveAffectedRange removes an affected range from a vulnerability.
func (h *CommandHandler) RemoveAffectedRange(id, component string) error {
	v, err := h.load(id)
	if err != nil {
		return err
	}
	if v == nil {
		return errors.New("schwachstellen: vulnerability not found")
	}
	updated, err := v.RemoveAffectedRange(component)
	if err != nil {
		return err
	}
	return h.store.Put(toRecord(updated))
}

// Import creates or updates a vulnerability with its affected ranges. It is
// used by external importers (e.g. the osv.dev sync) to record a vulnerability
// with all of its ranges in one operation. When the vulnerability exists, it
// is reconciled to the desired state (metadata + ranges replaced).
func (h *CommandHandler) Import(id, identifier, title, description string, cvss float64, source string, ranges []AffectedRangeInput) error {
	if id == "" {
		return errors.New("schwachstellen: vulnerability id is required")
	}
	if identifier == "" {
		return errors.New("schwachstellen: identifier is required")
	}
	if source == "" {
		source = "manual"
	}
	v, err := h.load(id)
	if err != nil {
		return err
	}
	if v == nil {
		nv, err := domain.New(id, identifier, title, description, cvss, source)
		if err != nil {
			return err
		}
		reconciled, err := nv.Reconcile(title, description, cvss, ranges)
		if err != nil {
			return err
		}
		return h.store.Put(toRecord(reconciled))
	}
	if v.Deleted {
		return domain.ErrVulnDeleted
	}
	updated, err := v.Reconcile(title, description, cvss, ranges)
	if err != nil {
		return err
	}
	return h.store.Put(toRecord(updated))
}

// Reconcile aligns an existing vulnerability's metadata and affected ranges
// with the desired state. It is a no-op when the vulnerability does not exist
// yet.
func (h *CommandHandler) Reconcile(id, title, description string, cvss float64, ranges []AffectedRangeInput) error {
	v, err := h.load(id)
	if err != nil {
		return err
	}
	if v == nil || v.Deleted {
		return nil
	}
	updated, err := v.Reconcile(title, description, cvss, ranges)
	if err != nil {
		return err
	}
	return h.store.Put(toRecord(updated))
}

func (h *CommandHandler) load(id string) (*domain.Vulnerability, error) {
	rec, err := h.store.Get(id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return fromRecord(rec), nil
}

// toRecord converts a domain vulnerability to a store record.
func toRecord(v *domain.Vulnerability) store.Record {
	rec := store.Record{
		ID:          v.ID,
		Identifier:  v.Identifier,
		Title:       v.Title,
		Description: v.Description,
		CVSS:        v.CVSS,
		Source:      v.Source,
		Deleted:     v.Deleted,
	}
	ecos := map[string]struct{}{}
	for _, a := range v.Affected {
		rec.Affected = append(rec.Affected, store.AffectedRange{
			Component:    a.Component,
			VersionRange: a.VersionRange,
			Ecosystem:    a.Ecosystem,
		})
		if a.Ecosystem != "" {
			ecos[a.Ecosystem] = struct{}{}
		}
	}
	rec.Ecosystems = sortedKeys(ecos)
	return rec
}

// fromRecord converts a store record to a domain vulnerability.
func fromRecord(rec *store.Record) *domain.Vulnerability {
	v := &domain.Vulnerability{
		ID:          rec.ID,
		Identifier:  rec.Identifier,
		Title:       rec.Title,
		Description: rec.Description,
		CVSS:        rec.CVSS,
		Source:      rec.Source,
		Affected:    map[string]domain.AffectedRange{},
		Deleted:     rec.Deleted,
	}
	for _, a := range rec.Affected {
		v.Affected[a.Component] = domain.AffectedRange{
			Component:    a.Component,
			VersionRange: a.VersionRange,
			Ecosystem:    a.Ecosystem,
		}
	}
	return v
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
