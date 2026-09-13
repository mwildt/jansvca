package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	projapp "github.com/mwildt/jansvca/service/internal/projekte/application"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
	"github.com/mwildt/jansvca/service/internal/server"
)

const bomDoc = `{"bomFormat":"CycloneDX","specVersion":"1.5","components":[
  {"type":"library","name":"lit","version":"3.2.1","purl":"pkg:npm/lit@3.2.1"},
  {"type":"library","name":"vite","version":"6.0.0","purl":"pkg:npm/vite@6.0.0"}
]}`

func setupHandler(t *testing.T) http.Handler {
	t.Helper()
	dir := t.TempDir()
	bus := eventstore.NewBus()
	projStore, err := eventstore.New(filepath.Join(dir, "projekte.wal"))
	if err != nil {
		t.Fatal(err)
	}
	vulnStore, err := eventstore.New(filepath.Join(dir, "vuln.wal"))
	if err != nil {
		t.Fatal(err)
	}
	projRead := projapp.NewProjectProjection()
	matchRead := vulnapp.NewMatchingProjection()

	bus.Subscribe(func(env eventstore.Envelope) bool {
		p := eventstore.EventType("projekte.")
		return len(env.Type) >= len(p) && env.Type[:len(p)] == p
	}, func(env eventstore.Envelope) {
		projRead.Apply(env)
		matchRead.Apply(env)
	})
	bus.Subscribe(func(env eventstore.Envelope) bool {
		p := eventstore.EventType("schwachstellen.")
		return len(env.Type) >= len(p) && env.Type[:len(p)] == p
	}, func(env eventstore.Envelope) {
		matchRead.Apply(env)
	})

	return server.New(server.Deps{
		Projects:        projapp.NewCommandHandler(projStore, bus),
		Vulnerabilities: vulnapp.NewCommandHandler(vulnStore, bus),
		ProjectRead:     projRead,
		MatchingRead:    matchRead,
	})
}

func TestSBOMUpload_ResultsInComponents(t *testing.T) {
	h := setupHandler(t)

	req := httptest.NewRequest("POST", "/api/projects", strings.NewReader(`{"id":"p1","name":"Projekt"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	t.Logf("create: %d %s", rec.Code, rec.Body.String())

	req = httptest.NewRequest("POST", "/api/projects/p1/sbom", strings.NewReader(bomDoc))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	t.Logf("sbom: %d %s", rec.Code, rec.Body.String())
	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest("GET", "/api/projects/p1", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	t.Logf("get: %d %s", rec.Code, rec.Body.String())
	var proj struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Components []struct {
			Component string `json:"component"`
			Version   string `json:"version"`
		} `json:"components"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &proj); err != nil {
		t.Fatal(err)
	}
	if proj.ID != "p1" {
		t.Errorf("expected id p1, got %q", proj.ID)
	}
	if len(proj.Components) != 2 {
		t.Fatalf("expected 2 components in GET, got %d: %+v", len(proj.Components), proj.Components)
	}
	for _, c := range proj.Components {
		if c.Component == "" || c.Version == "" {
			t.Errorf("expected non-empty component/version, got %+v", c)
		}
	}
}
