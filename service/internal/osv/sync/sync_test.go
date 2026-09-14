package sync_test

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/osv"
	syncpkg "github.com/mwildt/jansvca/service/internal/osv/sync"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
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

func newHarness(t *testing.T) (*vulnapp.CommandHandler, *vulnapp.MatchingProjection, *syncpkg.Sync, *httptest.Server) {
	t.Helper()
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	client := &osv.Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	sync := syncpkg.New(client, vulns, time.Hour)
	return vulns, read, sync, srv
}

func TestOnce_BulkImport(t *testing.T) {
	vulns, read, sync, srv := newHarness(t)
	defer srv.Close()

	zipBytes := mustZip(t, map[string]string{
		"PyPI/PYSEC-1.json": `{"id":"PYSEC-1","summary":"a","modified":"2024-01-01T00:00:00Z","affected":[{"package":{"purl":"pkg:npm/foo"},"ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"0"},{"fixed":"2.0.0"}]}]}]}`,
		"Go/GO-1.json":      `{"id":"GO-1","summary":"b","modified":"2024-01-02T00:00:00Z"}`,
	})
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/all.zip" {
			_, _ = w.Write(zipBytes)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	n, err := sync.Once(context.Background())
	if err != nil {
		t.Fatalf("once: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 upserts, got %d", n)
	}
	v := read.Get("PYSEC-1")
	if v == nil {
		t.Fatal("PYSEC-1 not imported")
	}
	if len(v.Affected) != 1 || v.Affected[0].Component != "pkg:npm/foo" || v.Affected[0].VersionRange != "<2.0.0" {
		t.Errorf("affected: %+v", v.Affected)
	}
	// lastSync should be the latest modified timestamp seen.
	if got := sync.LastSync(); !got.Equal(time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("lastSync: %v", got)
	}
	_ = vulns
}

func TestOnce_IncrementalImport(t *testing.T) {
	_, read, sync, srv := newHarness(t)
	defer srv.Close()

	zipBytes := mustZip(t, map[string]string{
		"PyPI/PYSEC-1.json": `{"id":"PYSEC-1","summary":"initial","modified":"2024-01-01T00:00:00Z"}`,
	})
	updatedRecord := `{"id":"PYSEC-1","summary":"updated","modified":"2024-03-01T00:00:00Z","affected":[{"package":{"purl":"pkg:npm/foo"},"ranges":[{"type":"ECOSYSTEM","events":[{"introduced":"0"},{"fixed":"2.0.0"}]}]}]}`
	newRecord := `{"id":"PYSEC-2","summary":"new","modified":"2024-02-15T00:00:00Z"}`

	phase := "bulk"
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/all.zip":
			_, _ = w.Write(zipBytes)
		case r.URL.Path == "/modified_id.csv":
			if phase != "incremental" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_, _ = w.Write([]byte("2024-03-01T00:00:00Z,PyPI/PYSEC-1\n2024-02-15T00:00:00Z,PyPI/PYSEC-2\n"))
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
	// lastSync advanced to the latest modified timestamp.
	if got := sync.LastSync(); !got.Equal(time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("lastSync: %v", got)
	}
}

func TestOnce_BulkError(t *testing.T) {
	_, _, sync, srv := newHarness(t)
	defer srv.Close()
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	if _, err := sync.Once(context.Background()); err == nil {
		t.Fatal("expected error from failed bulk fetch")
	}
}
