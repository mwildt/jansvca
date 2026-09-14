// Package domain contains the pure domain model of the schwachstellen
// module: vulnerabilities, their affected version ranges and the matching
// logic. It has no I/O dependencies.
package domain

import "github.com/mwildt/jansvca/service/internal/eventstore"

const (
	EventVulnerabilityCreated eventstore.EventType = "schwachstellen.vulnerability_created"
	EventVulnerabilityUpdated eventstore.EventType = "schwachstellen.vulnerability_updated"
	EventVulnerabilityDeleted eventstore.EventType = "schwachstellen.vulnerability_deleted"
	EventAffectedRangeAdded   eventstore.EventType = "schwachstellen.affected_range_added"
	EventAffectedRangeRemoved eventstore.EventType = "schwachstellen.affected_range_removed"
)

// VulnerabilityCreated is emitted when a vulnerability is recorded.
// Source records the origin of the vulnerability (e.g. "manual" for
// user-created entries via the REST API, "osv" for records imported from
// osv.dev). It is set at creation and not changed by later updates.
type VulnerabilityCreated struct {
	VulnerabilityID string  `json:"vulnerability_id"`
	Identifier      string  `json:"identifier"`
	Title           string  `json:"title"`
	Description     string  `json:"description,omitempty"`
	CVSS            float64 `json:"cvss"`
	Source          string  `json:"source,omitempty"`
}

func (VulnerabilityCreated) EventType() eventstore.EventType { return EventVulnerabilityCreated }

// VulnerabilityUpdated is emitted when mutable fields change.
type VulnerabilityUpdated struct {
	VulnerabilityID string  `json:"vulnerability_id"`
	Title           string  `json:"title,omitempty"`
	Description     string  `json:"description,omitempty"`
	CVSS            float64 `json:"cvss,omitempty"`
}

func (VulnerabilityUpdated) EventType() eventstore.EventType { return EventVulnerabilityUpdated }

// VulnerabilityDeleted is emitted for a soft-delete.
type VulnerabilityDeleted struct {
	VulnerabilityID string `json:"vulnerability_id"`
}

func (VulnerabilityDeleted) EventType() eventstore.EventType { return EventVulnerabilityDeleted }

// AffectedRangeAdded is emitted when a component version range is marked
// affected by a vulnerability.
type AffectedRangeAdded struct {
	VulnerabilityID string `json:"vulnerability_id"`
	Component       string `json:"component"`
	VersionRange    string `json:"version_range"`
}

func (AffectedRangeAdded) EventType() eventstore.EventType { return EventAffectedRangeAdded }

// AffectedRangeRemoved is emitted when an affected range is removed.
type AffectedRangeRemoved struct {
	VulnerabilityID string `json:"vulnerability_id"`
	Component       string `json:"component"`
	VersionRange    string `json:"version_range"`
}

func (AffectedRangeRemoved) EventType() eventstore.EventType { return EventAffectedRangeRemoved }
