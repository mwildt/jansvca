package application_test

import (
	"path/filepath"
	"testing"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	projapp "github.com/mwildt/jansvca/service/internal/projekte/application"
	"github.com/mwildt/jansvca/service/internal/sbom"
)

const sbomDoc = `{
  "bomFormat": "CycloneDX",
  "specVersion": "1.5",
  "components": [
    {"type": "library", "name": "lit", "version": "3.2.1", "purl": "pkg:npm/lit@3.2.1"},
    {"type": "library", "name": "vite", "version": "6.0.0", "purl": "pkg:npm/vite@6.0.0"}
  ]
}`

func TestImportComponents_FromSBOM(t *testing.T) {
	dir := t.TempDir()
	bus := eventstore.NewBus()
	store, err := eventstore.New(filepath.Join(dir, "projekte.wal"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	read := projapp.NewProjectProjection()
	bus.Subscribe(func(env eventstore.Envelope) bool {
		p := eventstore.EventType("projekte.")
		return len(env.Type) >= len(p) && env.Type[:len(p)] == p
	}, func(env eventstore.Envelope) {
		read.Apply(env)
	})
	h := projapp.NewCommandHandler(store, bus)

	if err := h.CreateProject("p1", "Projekt", ""); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// First import adds both components.
	inputs, err := sbom.Parse([]byte(sbomDoc))
	if err != nil {
		t.Fatalf("parse sbom: %v", err)
	}
	n, err := h.ImportComponents("p1", inputs)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 imported, got %d", n)
	}
	v := read.Get("p1")
	if len(v.Components) != 2 {
		t.Fatalf("expected 2 components in read model, got %d", len(v.Components))
	}

	// Re-importing the same SBOM is a no-op.
	n, err = h.ImportComponents("p1", inputs)
	if err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 changes on re-import, got %d", n)
	}

	// An updated SBOM version yields a single change.
	updated := `{
		"bomFormat":"CycloneDX","specVersion":"1.5",
		"components":[
			{"type":"library","name":"lit","version":"3.3.0","purl":"pkg:npm/lit@3.2.1"},
			{"type":"library","name":"vite","version":"6.0.0","purl":"pkg:npm/vite@6.0.0"}
		]
	}`
	upd, err := sbom.Parse([]byte(updated))
	if err != nil {
		t.Fatalf("parse updated: %v", err)
	}
	n, err = h.ImportComponents("p1", upd)
	if err != nil {
		t.Fatalf("import updated: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 change, got %d", n)
	}
	v = read.Get("p1")
	if got := findVersion(v.Components, "pkg:npm/lit@3.2.1"); got != "3.3.0" {
		t.Errorf("expected lit updated to 3.3.0, got %q", got)
	}
}

func findVersion(cs []projapp.ComponentView, component string) string {
	for _, c := range cs {
		if c.Component == component {
			return c.Version
		}
	}
	return ""
}
