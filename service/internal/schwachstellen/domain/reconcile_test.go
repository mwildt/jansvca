package domain_test

import (
	"testing"

	"github.com/mwildt/jansvca/service/internal/schwachstellen/domain"
)

func TestReconcile_NoChanges(t *testing.T) {
	v := mustNew(t, "V1", "t", "d", 7.5)
	v.Affected = map[string]domain.AffectedRange{
		"pkg:npm/foo": {Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
	}
	out, err := v.Reconcile("t", "d", 7.5, []domain.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(out.Affected) != 1 || out.Affected["pkg:npm/foo"].VersionRange != "<2.0.0" {
		t.Fatalf("expected unchanged ranges, got %+v", out.Affected)
	}
	if out.Title != "t" || out.CVSS != 7.5 {
		t.Errorf("metadata changed: %+v", out)
	}
}

func TestReconcile_UpdatesMetadata(t *testing.T) {
	v := mustNew(t, "V1", "t", "d", 7.5)
	out, err := v.Reconcile("new-title", "new-desc", 9.1, nil)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if out.Title != "new-title" || out.Description != "new-desc" || out.CVSS != 9.1 {
		t.Errorf("metadata not updated: %+v", out)
	}
}

func TestReconcile_AddsAndRemovesRanges(t *testing.T) {
	v := mustNew(t, "V1", "t", "d", 7.5)
	v.Affected = map[string]domain.AffectedRange{
		"pkg:npm/foo": {Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
	}
	out, err := v.Reconcile("t", "d", 7.5, []domain.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<3.0.0"},
		{Component: "pkg:npm/bar", VersionRange: ">=1.0.0"},
	})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(out.Affected) != 2 {
		t.Fatalf("expected 2 ranges, got %d: %+v", len(out.Affected), out.Affected)
	}
	if out.Affected["pkg:npm/foo"].VersionRange != "<3.0.0" {
		t.Errorf("foo range: %+v", out.Affected["pkg:npm/foo"])
	}
	if _, ok := out.Affected["pkg:npm/bar"]; !ok {
		t.Errorf("bar not added: %+v", out.Affected)
	}
}

func TestReconcile_DeletedRejected(t *testing.T) {
	v := mustNew(t, "V1", "t", "d", 0)
	v.Deleted = true
	if _, err := v.Reconcile("t", "", 0, nil); err != domain.ErrVulnDeleted {
		t.Fatalf("expected ErrVulnDeleted, got %v", err)
	}
}

func TestReconcile_ConflictingRanges(t *testing.T) {
	v := mustNew(t, "V1", "t", "d", 0)
	_, err := v.Reconcile("t", "", 0, []domain.AffectedRangeInput{
		{Component: "c", VersionRange: "<1.0.0"},
		{Component: "c", VersionRange: "<2.0.0"},
	})
	if err == nil {
		t.Fatal("expected error for conflicting ranges")
	}
}

func mustNew(t *testing.T, id, title, desc string, cvss float64) *domain.Vulnerability {
	t.Helper()
	v, err := domain.New(id, id, title, desc, cvss, "manual")
	if err != nil {
		t.Fatalf("new %s: %v", id, err)
	}
	return v
}
