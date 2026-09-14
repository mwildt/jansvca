package application_test

import (
	"path/filepath"
	"testing"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
)

func TestImport_NewVulnerabilityWithRanges(t *testing.T) {
	dir := t.TempDir()
	bus := eventstore.NewBus()
	store, err := eventstore.New(filepath.Join(dir, "schwachstellen.wal"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	read := vulnapp.NewMatchingProjection()
	bus.Subscribe(func(env eventstore.Envelope) bool {
		p := eventstore.EventType("schwachstellen.")
		return len(env.Type) >= len(p) && env.Type[:len(p)] == p
	}, func(env eventstore.Envelope) {
		read.Apply(env)
	})
	vulns := vulnapp.NewCommandHandler(store, bus)

	err = vulns.Import("GHSA-1", "GHSA-1", "RCE in foo", "details", 9.8, "osv", []vulnapp.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
		{Component: "pkg:npm/bar", VersionRange: ">=1.0.0"},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	v := read.Get("GHSA-1")
	if v == nil {
		t.Fatal("vuln not in read model")
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
	dir := t.TempDir()
	bus := eventstore.NewBus()
	store, err := eventstore.New(filepath.Join(dir, "schwachstellen.wal"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	read := vulnapp.NewMatchingProjection()
	bus.Subscribe(func(env eventstore.Envelope) bool {
		p := eventstore.EventType("schwachstellen.")
		return len(env.Type) >= len(p) && env.Type[:len(p)] == p
	}, func(env eventstore.Envelope) {
		read.Apply(env)
	})
	vulns := vulnapp.NewCommandHandler(store, bus)

	if err := vulns.Import("V1", "V1", "t", "d", 5.0, "osv", []vulnapp.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
	}); err != nil {
		t.Fatalf("import 1: %v", err)
	}

	// Update: change range, add a new one, change metadata.
	if err := vulns.Import("V1", "V1", "new-title", "new-desc", 8.0, "osv", []vulnapp.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<3.0.0"},
		{Component: "pkg:npm/bar", VersionRange: ">=1.0.0"},
	}); err != nil {
		t.Fatalf("import 2: %v", err)
	}
	v := read.Get("V1")
	if v == nil {
		t.Fatal("vuln not in read model")
	}
	if v.Title != "new-title" || v.CVSS != 8.0 {
		t.Errorf("metadata not updated: %+v", v)
	}
	if len(v.Affected) != 2 {
		t.Fatalf("affected ranges: %+v", v.Affected)
	}
}

func TestImport_NoOpWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	bus := eventstore.NewBus()
	store, err := eventstore.New(filepath.Join(dir, "schwachstellen.wal"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	read := vulnapp.NewMatchingProjection()
	bus.Subscribe(func(env eventstore.Envelope) bool {
		p := eventstore.EventType("schwachstellen.")
		return len(env.Type) >= len(p) && env.Type[:len(p)] == p
	}, func(env eventstore.Envelope) {
		read.Apply(env)
	})
	vulns := vulnapp.NewCommandHandler(store, bus)

	ranges := []vulnapp.AffectedRangeInput{
		{Component: "pkg:npm/foo", VersionRange: "<2.0.0"},
	}
	if err := vulns.Import("V1", "V1", "t", "d", 5.0, "osv", ranges); err != nil {
		t.Fatalf("import 1: %v", err)
	}
	before := store.CurrentVersion("vulnerability:V1")
	// Re-importing identical data should be a no-op.
	if err := vulns.Import("V1", "V1", "t", "d", 5.0, "osv", ranges); err != nil {
		t.Fatalf("import 2: %v", err)
	}
	after := store.CurrentVersion("vulnerability:V1")
	if before != after {
		t.Fatalf("expected no new events (before=%d after=%d)", before, after)
	}
}
