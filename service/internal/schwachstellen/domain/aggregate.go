package domain

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mwildt/jansvca/service/internal/eventstore"
)

// AffectedRange describes which versions of a component are affected by a
// vulnerability. VersionRange is a semver range expression.
type AffectedRange struct {
	Component    string
	VersionRange string
}

// Vulnerability is the aggregate root for a tracked vulnerability.
type Vulnerability struct {
	ID          string
	Identifier  string
	Title       string
	Description string
	CVSS        float64
	Affected    map[string]AffectedRange
	Deleted     bool
	version     eventstore.Version
}

var (
	ErrVulnDeleted   = errors.New("schwachstellen: vulnerability is deleted")
	ErrRangeExists   = errors.New("schwachstellen: affected range already exists")
	ErrRangeNotFound = errors.New("schwachstellen: affected range not found")
)

// Apply applies an envelope to the aggregate.
func (v *Vulnerability) Apply(env eventstore.Envelope) error {
	switch env.Type {
	case EventVulnerabilityCreated:
		var e VulnerabilityCreated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return err
		}
		v.ID = e.VulnerabilityID
		v.Identifier = e.Identifier
		v.Title = e.Title
		v.Description = e.Description
		v.CVSS = e.CVSS
		v.Affected = map[string]AffectedRange{}
	case EventVulnerabilityUpdated:
		var e VulnerabilityUpdated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return err
		}
		if e.Title != "" {
			v.Title = e.Title
		}
		if e.Description != "" {
			v.Description = e.Description
		}
		if e.CVSS != 0 {
			v.CVSS = e.CVSS
		}
	case EventVulnerabilityDeleted:
		v.Deleted = true
	case EventAffectedRangeAdded:
		var e AffectedRangeAdded
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return err
		}
		if v.Affected == nil {
			v.Affected = map[string]AffectedRange{}
		}
		v.Affected[e.Component] = AffectedRange{Component: e.Component, VersionRange: e.VersionRange}
	case EventAffectedRangeRemoved:
		var e AffectedRangeRemoved
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return err
		}
		delete(v.Affected, e.Component)
	}
	v.version = env.Version
	return nil
}

// Version returns the aggregate's current event version.
func (v *Vulnerability) Version() eventstore.Version { return v.version }

// CreateVulnerability produces the events for creating a vulnerability.
func CreateVulnerability(id, identifier, title, description string, cvss float64) ([]eventstore.PayloadEvent, error) {
	if id == "" {
		return nil, errors.New("schwachstellen: vulnerability id is required")
	}
	if identifier == "" {
		return nil, errors.New("schwachstellen: identifier is required")
	}
	if title == "" {
		return nil, errors.New("schwachstellen: title is required")
	}
	return []eventstore.PayloadEvent{VulnerabilityCreated{
		VulnerabilityID: id,
		Identifier:      identifier,
		Title:           title,
		Description:     description,
		CVSS:            cvss,
	}}, nil
}

// Update produces an update event for mutable fields.
func (v *Vulnerability) Update(title, description string, cvss float64) ([]eventstore.PayloadEvent, error) {
	if v == nil || v.ID == "" {
		return nil, errors.New("schwachstellen: vulnerability not found")
	}
	if v.Deleted {
		return nil, ErrVulnDeleted
	}
	return []eventstore.PayloadEvent{VulnerabilityUpdated{
		VulnerabilityID: v.ID,
		Title:           title,
		Description:     description,
		CVSS:            cvss,
	}}, nil
}

// Delete produces a soft-delete event.
func (v *Vulnerability) Delete() ([]eventstore.PayloadEvent, error) {
	if v == nil || v.ID == "" {
		return nil, errors.New("schwachstellen: vulnerability not found")
	}
	if v.Deleted {
		return nil, ErrVulnDeleted
	}
	return []eventstore.PayloadEvent{VulnerabilityDeleted{VulnerabilityID: v.ID}}, nil
}

// AddAffectedRange marks a component version range as affected.
func (v *Vulnerability) AddAffectedRange(component, versionRange string) ([]eventstore.PayloadEvent, error) {
	if v == nil || v.ID == "" {
		return nil, errors.New("schwachstellen: vulnerability not found")
	}
	if v.Deleted {
		return nil, ErrVulnDeleted
	}
	if component == "" {
		return nil, errors.New("schwachstellen: component is required")
	}
	if versionRange == "" {
		return nil, errors.New("schwachstellen: version range is required")
	}
	if _, ok := v.Affected[component]; ok {
		return nil, fmt.Errorf("%w: %s", ErrRangeExists, component)
	}
	return []eventstore.PayloadEvent{AffectedRangeAdded{
		VulnerabilityID: v.ID,
		Component:       component,
		VersionRange:    versionRange,
	}}, nil
}

// RemoveAffectedRange removes an affected range.
func (v *Vulnerability) RemoveAffectedRange(component string) ([]eventstore.PayloadEvent, error) {
	if v == nil || v.ID == "" {
		return nil, errors.New("schwachstellen: vulnerability not found")
	}
	if v.Deleted {
		return nil, ErrVulnDeleted
	}
	if _, ok := v.Affected[component]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrRangeNotFound, component)
	}
	return []eventstore.PayloadEvent{AffectedRangeRemoved{
		VulnerabilityID: v.ID,
		Component:       component,
		VersionRange:    v.Affected[component].VersionRange,
	}}, nil
}
