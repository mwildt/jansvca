package application_test

import (
	"path/filepath"
	"testing"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	projapp "github.com/mwildt/jansvca/service/internal/projekte/application"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
	vulnstore "github.com/mwildt/jansvca/service/internal/schwachstellen/store"
)

func TestMatchingEndToEnd(t *testing.T) {
	dir := t.TempDir()
	bus := eventstore.NewBus()
	projStore, err := eventstore.New(filepath.Join(dir, "projekte.wal"))
	if err != nil {
		t.Fatalf("proj store: %v", err)
	}
	projRead := projapp.NewProjectProjection()
	bus.Subscribe(func(env eventstore.Envelope) bool {
		p := eventstore.EventType("projekte.")
		return len(env.Type) >= len(p) && env.Type[:len(p)] == p
	}, func(env eventstore.Envelope) {
		projRead.Apply(env)
	})
	projects := projapp.NewCommandHandler(projStore, bus)

	s, err := vulnstore.New(dir)
	if err != nil {
		t.Fatalf("vuln store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	vulns := vulnapp.NewCommandHandler(s)
	matchRead := vulnapp.NewQueryService(s, projRead)

	if err := projects.CreateProject("p1", "Projekt Eins", "desc"); err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := projects.AddComponent("p1", "pkg:gem/rails", "1.5.0"); err != nil {
		t.Fatalf("add component: %v", err)
	}
	if err := vulns.Create("v1", "CVE-2024-0001", "RCE in rails", "bad", 9.8); err != nil {
		t.Fatalf("create vuln: %v", err)
	}
	if err := vulns.AddAffectedRange("v1", "pkg:gem/rails", ">=1.0.0 <2.0.0"); err != nil {
		t.Fatalf("add range: %v", err)
	}

	if v := matchRead.Get("v1"); v == nil || v.Source != "manual" {
		t.Errorf("expected source=manual, got %+v", v)
	}
	matches := matchRead.Matches("p1")
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d: %+v", len(matches), matches)
	}
	m := matches[0]
	if m.Component != "pkg:gem/rails" || m.Version != "1.5.0" {
		t.Errorf("unexpected match component/version: %+v", m)
	}
	if m.VulnerabilityID != "v1" || m.CVSS != 9.8 {
		t.Errorf("unexpected vuln: %+v", m)
	}
	if got := len(projRead.All()); got != 1 {
		t.Errorf("expected 1 project in read model, got %d", got)
	}

	if err := projects.Delete("p1"); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	if got := len(projRead.All()); got != 0 {
		t.Errorf("expected 0 projects after delete, got %d", got)
	}
	if got := len(matchRead.Matches("p1")); got != 0 {
		t.Errorf("expected 0 matches after project delete, got %d", got)
	}

	if err := projects.CreateProject("p2", "Projekt Zwei", ""); err != nil {
		t.Fatalf("create project p2: %v", err)
	}
	if err := projects.AddComponent("p2", "pkg:gem/rails", "2.5.0"); err != nil {
		t.Fatalf("add component p2: %v", err)
	}
	if got := len(matchRead.Matches("p2")); got != 0 {
		t.Errorf("expected 0 matches for out-of-range version, got %d", got)
	}
}
