// Package domain contains the pure domain model of the schwachstellen
// module: vulnerabilities and their affected version ranges. It has no I/O
// dependencies.
//
// The vulnerability is a state-based aggregate: operations return the new
// desired state rather than a stream of domain events. Persistence is handled
// by the store adapter (JSON files + Bleve), which replaces the previous
// event-sourced WAL.
package domain

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

// AffectedRange describes which versions of a component are affected by a
// vulnerability. VersionRange is a semver range expression. Ecosystem is the
// OSV ecosystem of the package, when known.
type AffectedRange struct {
	Component    string
	VersionRange string
	Ecosystem    string
}

// Vulnerability is the aggregate root for a tracked vulnerability.
// Source records where the vulnerability originated from ("manual" for
// REST-API entries, "osv" for osv.dev imports); it is immutable after
// creation.
type Vulnerability struct {
	ID          string
	Identifier  string
	Title       string
	Description string
	CVSS        float64
	Source      string
	Affected    map[string]AffectedRange
	Deleted     bool
}

var (
	ErrVulnDeleted   = errors.New("schwachstellen: vulnerability is deleted")
	ErrRangeExists   = errors.New("schwachstellen: affected range already exists")
	ErrRangeNotFound = errors.New("schwachstellen: affected range not found")
)

// New creates a new vulnerability state after validating the required fields.
// source defaults to "manual" when empty.
func New(id, identifier, title, description string, cvss float64, source string) (*Vulnerability, error) {
	if id == "" {
		return nil, errors.New("schwachstellen: vulnerability id is required")
	}
	if identifier == "" {
		return nil, errors.New("schwachstellen: identifier is required")
	}
	if title == "" {
		return nil, errors.New("schwachstellen: title is required")
	}
	if source == "" {
		source = "manual"
	}
	return &Vulnerability{
		ID:          id,
		Identifier:  identifier,
		Title:       title,
		Description: description,
		CVSS:        cvss,
		Source:      source,
		Affected:    map[string]AffectedRange{},
	}, nil
}

// Update returns a copy of v with mutable fields changed. It errors when the
// vulnerability is deleted.
func (v *Vulnerability) Update(title, description string, cvss float64) (*Vulnerability, error) {
	if v == nil || v.ID == "" {
		return nil, errors.New("schwachstellen: vulnerability not found")
	}
	if v.Deleted {
		return nil, ErrVulnDeleted
	}
	out := v.clone()
	out.Title = title
	out.Description = description
	out.CVSS = cvss
	return out, nil
}

// Delete returns a copy of v marked as deleted.
func (v *Vulnerability) Delete() (*Vulnerability, error) {
	if v == nil || v.ID == "" {
		return nil, errors.New("schwachstellen: vulnerability not found")
	}
	if v.Deleted {
		return nil, ErrVulnDeleted
	}
	out := v.clone()
	out.Deleted = true
	return out, nil
}

// AddAffectedRange returns a copy of v with the given affected range added.
func (v *Vulnerability) AddAffectedRange(component, versionRange string) (*Vulnerability, error) {
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
	out := v.clone()
	out.Affected[component] = AffectedRange{Component: component, VersionRange: versionRange}
	return out, nil
}

// RemoveAffectedRange returns a copy of v with the given component range
// removed.
func (v *Vulnerability) RemoveAffectedRange(component string) (*Vulnerability, error) {
	if v == nil || v.ID == "" {
		return nil, errors.New("schwachstellen: vulnerability not found")
	}
	if v.Deleted {
		return nil, ErrVulnDeleted
	}
	if _, ok := v.Affected[component]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrRangeNotFound, component)
	}
	out := v.clone()
	delete(out.Affected, component)
	return out, nil
}

// AffectedRangeInput is the desired state of an affected range, used by
// Reconcile to diff against the current aggregate state.
type AffectedRangeInput struct {
	Component    string
	VersionRange string
	Ecosystem    string
}

// Reconcile returns a copy of v aligned with the desired metadata and affected
// ranges. Only the resulting state is returned; callers persist it as-is. A
// nil ranges slice preserves the existing ranges. When ranges is non-nil, it
// replaces the affected set entirely (entries with empty components are
// skipped). The aggregate must exist and not be deleted.
func (v *Vulnerability) Reconcile(title, description string, cvss float64, ranges []AffectedRangeInput) (*Vulnerability, error) {
	if v == nil || v.ID == "" {
		return nil, errors.New("schwachstellen: vulnerability not found")
	}
	if v.Deleted {
		return nil, ErrVulnDeleted
	}
	out := v.clone()
	out.Title = title
	out.Description = description
	out.CVSS = cvss
	if ranges != nil {
		desired := make(map[string]string, len(ranges))
		for _, r := range ranges {
			if r.Component == "" {
				continue
			}
			if prev, dup := desired[r.Component]; dup && prev != r.VersionRange {
				return nil, fmt.Errorf("schwachstellen: conflicting ranges for %s: %s vs %s", r.Component, prev, r.VersionRange)
			}
			desired[r.Component] = r.VersionRange
		}
		out.Affected = make(map[string]AffectedRange, len(desired))
		for _, c := range slices.Sorted(maps.Keys(desired)) {
			prev, ok := v.Affected[c]
			eco := ""
			if ok {
				eco = prev.Ecosystem
			}
			out.Affected[c] = AffectedRange{Component: c, VersionRange: desired[c], Ecosystem: eco}
		}
	}
	return out, nil
}

// clone returns a shallow copy of v with a fresh Affected map.
func (v *Vulnerability) clone() *Vulnerability {
	if v == nil {
		return nil
	}
	out := *v
	if v.Affected != nil {
		out.Affected = make(map[string]AffectedRange, len(v.Affected))
		for k, val := range v.Affected {
			out.Affected[k] = val
		}
	}
	return &out
}
