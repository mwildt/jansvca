package application_test

import (
	"testing"

	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
	vulnstore "github.com/mwildt/jansvca/service/internal/schwachstellen/store"
)

func newTestStore(t *testing.T) *vulnstore.Store {
	t.Helper()
	s, err := vulnstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestImport_NewVulnerabilityWithRanges(t *testing.T) {
	s := newTestStore(t)
	vulns := vulnapp.NewCommandHandler(s)
	err := vulns.Import("GHSA-1", "GHSA-1", "RCE in foo", "details", 9.8, "osv", []vulnapp.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<2.0.0", Ecosystem: "npm"},
		{Component: "pkg:npm/bar", VersionRange: ">=1.0.0", Ecosystem: "npm"},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	v, err := s.Get("GHSA-1")
	if err != nil || v == nil {
		t.Fatal("vuln not in store")
	}
	if v.Title != "RCE in foo" || v.CVSS != 9.8 {
		t.Errorf("metadata: %+v", v)
	}
	if v.Source != "osv" {
		t.Errorf("source: %q (want osv)", v.Source)
	}
	if len(v.Affected) != 2 {
		t.Fatalf("affected ranges: %+v", v.Affected)
	}
}

func TestImport_UpdateExistingVulnerability(t *testing.T) {
	s := newTestStore(t)
	vulns := vulnapp.NewCommandHandler(s)
	if err := vulns.Import("V1", "V1", "t", "d", 5.0, "osv", []vulnapp.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<2.0.0", Ecosystem: "npm"},
	}); err != nil {
		t.Fatalf("import 1: %v", err)
	}
	if err := vulns.Import("V1", "V1", "new-title", "new-desc", 8.0, "osv", []vulnapp.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<3.0.0", Ecosystem: "npm"},
		{Component: "pkg:npm/bar", VersionRange: ">=1.0.0", Ecosystem: "npm"},
	}); err != nil {
		t.Fatalf("import 2: %v", err)
	}
	v, _ := s.Get("V1")
	if v == nil {
		t.Fatal("vuln not in store")
	}
	if v.Title != "new-title" || v.CVSS != 8.0 {
		t.Errorf("metadata not updated: %+v", v)
	}
	if len(v.Affected) != 2 {
		t.Fatalf("affected ranges: %+v", v.Affected)
	}
}

func TestImport_IdempotentWhenUnchanged(t *testing.T) {
	s := newTestStore(t)
	vulns := vulnapp.NewCommandHandler(s)
	ranges := []vulnapp.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<2.0.0", Ecosystem: "npm"},
	}
	if err := vulns.Import("V1", "V1", "t", "d", 5.0, "osv", ranges); err != nil {
		t.Fatalf("import 1: %v", err)
	}
	v1, _ := s.Get("V1")
	before := v1.Affected
	if err := vulns.Import("V1", "V1", "t", "d", 5.0, "osv", ranges); err != nil {
		t.Fatalf("import 2: %v", err)
	}
	v2, _ := s.Get("V1")
	after := v2.Affected
	if len(before) != len(after) {
		t.Fatalf("affected changed on idempotent re-import: before=%v after=%v", before, after)
	}
}

func TestImport_PersistsAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	s, err := vulnstore.New(dir)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	vulns := vulnapp.NewCommandHandler(s)
	if err := vulns.Create("v1", "CVE-1", "T", "D", 1.0); err != nil {
		t.Fatalf("create: %v", err)
	}
	_ = s.Close()

	s2, err := vulnstore.New(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })
	v, _ := s2.Get("v1")
	if v == nil || v.Title != "T" {
		t.Fatalf("after reopen: %+v", v)
	}
}
