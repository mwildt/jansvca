// Package domain (migration events): the event types below are kept only to
// migrate an existing WAL into the new file-based store. They are not used by
// the state-based aggregate operations.
package domain

// Legacy event type discriminators, formerly used by the WAL event store.
const (
	EventVulnerabilityCreated = "schwachstellen.vulnerability_created"
	EventVulnerabilityUpdated = "schwachstellen.vulnerability_updated"
	EventVulnerabilityDeleted = "schwachstellen.vulnerability_deleted"
	EventAffectedRangeAdded   = "schwachstellen.affected_range_added"
	EventAffectedRangeRemoved = "schwachstellen.affected_range_removed"
)

// VulnerabilityCreated was emitted when a vulnerability was recorded.
type VulnerabilityCreated struct {
	VulnerabilityID string  `json:"vulnerability_id"`
	Identifier      string  `json:"identifier"`
	Title           string  `json:"title"`
	Description     string  `json:"description,omitempty"`
	CVSS            float64 `json:"cvss"`
	Source          string  `json:"source,omitempty"`
}

// VulnerabilityUpdated was emitted when mutable fields changed.
type VulnerabilityUpdated struct {
	VulnerabilityID string  `json:"vulnerability_id"`
	Title           string  `json:"title,omitempty"`
	Description     string  `json:"description,omitempty"`
	CVSS            float64 `json:"cvss,omitempty"`
}

// VulnerabilityDeleted was emitted for a soft-delete.
type VulnerabilityDeleted struct {
	VulnerabilityID string `json:"vulnerability_id"`
}

// AffectedRangeAdded was emitted when a component version range was marked
// affected by a vulnerability.
type AffectedRangeAdded struct {
	VulnerabilityID string `json:"vulnerability_id"`
	Component       string `json:"component"`
	VersionRange    string `json:"version_range"`
}

// AffectedRangeRemoved was emitted when an affected range was removed.
type AffectedRangeRemoved struct {
	VulnerabilityID string `json:"vulnerability_id"`
	Component       string `json:"component"`
	VersionRange    string `json:"version_range"`
}
