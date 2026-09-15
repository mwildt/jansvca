package sync_test

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mwildt/jansvca/service/internal/osv"
	syncpkg "github.com/mwildt/jansvca/service/internal/osv/sync"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
	vulnstore "github.com/mwildt/jansvca/service/internal/schwachstellen/store"
)

func mustZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}

func newHarness(t *testing.T, ecosystems ...string) (*vulnapp.CommandHandler, *vulnapp.QueryService, *syncpkg.Sync, *httptest.Server) {
	t.Helper()
	s, err := vulnstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	vulns := vulnapp.NewCommandHandler(s)
	read := vulnapp.NewQueryService(s, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	client := &osv.Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	sync := syncpkg.New(client, vulns, nil, time.Hour, ecosystems...)
	return vulns, read, sync, srv
}

func newHarnessState(t *testing.T, state *syncpkg.StateStore, ecosystems ...string) (*vulnapp.CommandHandler, *vulnapp.QueryService, *syncpkg.Sync, *httptest.Server) {
	t.Helper()
	s, err := vulnstore.New(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	vulns := vulnapp.NewCommandHandler(s)
	read := vulnapp.NewQueryService(s, nil)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	client := &osv.Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	sync := syncpkg.New(client, vulns, state, time.Hour, ecosystems...)
	return vulns, read, sync, srv
}

func TestOnce_BulkImport(t *testing.T) {
	vulns, read, sync, srv := newHarness(t, "Go")
	defer srv.Close()
	goZip := mustZip(t, map[string]string{
		"GO-1.json": `{"id":"GO-1","summary":"b","modified":"2024-01-02T00:00:00Z","affected":[{"package":{"purl":"pkg:golang/foo","ecosystem":"Go"},"ranges":[{"type":"SEMVER","events":[{"introduced":"0"},{"fixed":"2.0.0"}]}]}]}`,
	})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Go/all.zip" {
			_, _ = w.Write(goZip)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	n, err := sync.Once(context.Background())
	if err != nil {
		t.Fatalf("once: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 upsert, got %d", n)
	}
	v := read.Get("GO-1")
	if v == nil {
		t.Fatal("GO-1 not imported")
	}
	if len(v.Affected) != 1 || v.Affected[0].Component != "pkg:golang/foo" || v.Affected[0].VersionRange != "<2.0.0" {
		t.Errorf("affected: %+v", v.Affected)
	}
	if got := sync.LastSyncEcosystem("Go"); !got.Equal(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("lastSync Go: %v", got)
	}
	_ = vulns
}

// TestOnce_BulkImportMultipleEcosystems verifies that a sync configured for
// several ecosystems fetches each per-ecosystem all.zip independently.
func TestOnce_BulkImportMultipleEcosystems(t *testing.T) {
	_, read, sync, srv := newHarness(t, "Go", "PyPI")
	defer srv.Close()
	goZip := mustZip(t, map[string]string{
		"GO-1.json": `{"id":"GO-1","summary":"go","modified":"2024-01-01T00:00:00Z"}`,
	})
	pypiZip := mustZip(t, map[string]string{
		"PYSEC-1.json": `{"id":"PYSEC-1","summary":"pypi","modified":"2024-01-02T00:00:00Z"}`,
	})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Go/all.zip":
			_, _ = w.Write(goZip)
		case "/PyPI/all.zip":
			_, _ = w.Write(pypiZip)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	n, err := sync.Once(context.Background())
	if err != nil {
		t.Fatalf("once: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 upserts, got %d", n)
	}
	if read.Get("GO-1") == nil {
		t.Error("GO-1 not imported")
	}
	if read.Get("PYSEC-1") == nil {
		t.Error("PYSEC-1 not imported")
	}
	if got := sync.LastSync(); !got.Equal(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("lastSync (max): %v", got)
	}
}

func TestOnce_IncrementalImport(t *testing.T) {
	_, read, sync, srv := newHarness(t, "PyPI")
	defer srv.Close()
	zipBytes := mustZip(t, map[string]string{
		"PYSEC-1.json": `{"id":"PYSEC-1","summary":"initial","modified":"2024-01-01T00:00:00Z"}`,
	})
	updatedRecord := `{"id":"PYSEC-1","summary":"updated","modified":"2024-03-01T00:00:00Z","affected":[{"package":{"purl":"pkg:npm/foo","ecosystem":"npm"},"ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"0"},{"fixed":"2.0.0"}]}]}]}`
	newRecord := `{"id":"PYSEC-2","summary":"new","modified":"2024-02-15T00:00:00Z"}`
	phase := "bulk"
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/PyPI/all.zip":
			_, _ = w.Write(zipBytes)
		case r.URL.Path == "/PyPI/modified_id.csv":
			if phase != "incremental" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Per-ecosystem CSV omits the ecosystem prefix.
			_, _ = w.Write([]byte("2024-03-01T00:00:00Z,PYSEC-1\n2024-02-15T00:00:00Z,PYSEC-2\n"))
		case r.URL.Path == "/PyPI/PYSEC-1.json":
			_, _ = w.Write([]byte(updatedRecord))
		case r.URL.Path == "/PyPI/PYSEC-2.json":
			_, _ = w.Write([]byte(newRecord))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	if _, err := sync.Once(context.Background()); err != nil {
		t.Fatalf("bulk: %v", err)
	}
	phase = "incremental"
	n, err := sync.Once(context.Background())
	if err != nil {
		t.Fatalf("incremental: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 upserts, got %d", n)
	}
	v := read.Get("PYSEC-1")
	if v == nil || v.Title != "updated" || len(v.Affected) != 1 {
		t.Errorf("PYSEC-1 not updated: %+v", v)
	}
	if read.Get("PYSEC-2") == nil {
		t.Error("PYSEC-2 not imported")
	}
	if got := sync.LastSyncEcosystem("PyPI"); !got.Equal(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("lastSync PyPI: %v", got)
	}
}

func TestOnce_BulkError(t *testing.T) {
	_, _, sync, srv := newHarness(t, "Go")
	defer srv.Close()
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, err := sync.Once(context.Background()); err == nil {
		t.Fatal("expected error from failed bulk fetch")
	}
}

// TestNew_DefaultEcosystems verifies that a Sync without explicit ecosystems
// defaults to importing the Go and Maven (Java) ecosystems.
func TestNew_DefaultEcosystems(t *testing.T) {
	_, read, sync, srv := newHarness(t)
	defer srv.Close()
	if len(sync.Ecosystems) != 2 || sync.Ecosystems[0] != "Go" || sync.Ecosystems[1] != "Maven" {
		t.Fatalf("default ecosystems: %+v", sync.Ecosystems)
	}
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Go/all.zip":
			_, _ = w.Write(mustZip(t, map[string]string{"GO-1.json": `{"id":"GO-1","summary":"go","modified":"2024-01-01T00:00:00Z"}`}))
		case "/Maven/all.zip":
			_, _ = w.Write(mustZip(t, map[string]string{"GHSA-1.json": `{"id":"GHSA-1","summary":"java","modified":"2024-01-02T00:00:00Z"}`}))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})
	if _, err := sync.Once(context.Background()); err != nil {
		t.Fatalf("once: %v", err)
	}
	if read.Get("GO-1") == nil {
		t.Error("GO-1 not imported by default ecosystems")
	}
	if read.Get("GHSA-1") == nil {
		t.Error("GHSA-1 (Maven/Java) not imported by default ecosystems")
	}
}
