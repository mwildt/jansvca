package domain_test

import (
	"testing"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/schwachstellen/domain"
)

func TestReconcile_NoChangesNoEvents(t *testing.T) {
	v := &domain.Vulnerability{ID: "V1", Title: "t", Description: "d", CVSS: 7.5}
	v.Affected = map[string]domain.AffectedRange{
		"pkg:npm/foo": {Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
	}
	events, err := v.Reconcile("t", "d", 7.5, []domain.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %d: %+v", len(events), events)
	}
}

func TestReconcile_UpdatesMetadata(t *testing.T) {
	v := &domain.Vulnerability{ID: "V1", Title: "t", Description: "d", CVSS: 7.5}
	v.Affected = map[string]domain.AffectedRange{}
	events, err := v.Reconcile("new-title", "new-desc", 9.1, nil)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType() != domain.EventVulnerabilityUpdated {
		t.Errorf("type: %v", events[0].EventType())
	}
}

func TestReconcile_AddsAndRemovesRanges(t *testing.T) {
	v := &domain.Vulnerability{ID: "V1", Title: "t", Description: "d", CVSS: 7.5}
	v.Affected = map[string]domain.AffectedRange{
		"pkg:npm/foo": {Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
	}
	events, err := v.Reconcile("t", "d", 7.5, []domain.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<3.0.0"},
		{Component: "pkg:npm/bar", VersionRange: ">=1.0.0"},
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	types := map[eventstore.EventType]int{}
	for _, e := range events {
		types[e.EventType()]++
	}
	// foo's range changed (<2.0.0 -> <3.0.0): one remove + one add;
	// bar is new: one add. Total: 1 removed, 2 added.
	if types[domain.EventAffectedRangeRemoved] != 1 {
		t.Errorf("removed: %+v", types)
	}
	if types[domain.EventAffectedRangeAdded] != 2 {
		t.Errorf("added: %+v", types)
	}
}

func TestReconcile_DeletedRejected(t *testing.T) {
	v := &domain.Vulnerability{ID: "V1", Title: "t", Deleted: true}
	v.Affected = map[string]domain.AffectedRange{}
	if _, err := v.Reconcile("t", "", 0, nil); err != domain.ErrVulnDeleted {
		t.Fatalf("expected ErrVulnDeleted, got %v", err)
	}
}

func TestReconcile_ConflictingRanges(t *testing.T) {
	v := &domain.Vulnerability{ID: "V1", Title: "t"}
	v.Affected = map[string]domain.AffectedRange{}
	_, err := v.Reconcile("t", "", 0, []domain.AffectedRangeInput{
		{Component: "c", VersionRange: "<1.0.0"},
		{Component: "c", VersionRange: "<2.0.0"},
	})
	if err == nil {
		t.Fatal("expected error for conflicting ranges")
	}
}
