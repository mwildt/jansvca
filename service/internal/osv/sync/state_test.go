package sync_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	syncpkg "github.com/mwildt/jansvca/service/internal/osv/sync"
)

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func TestStateStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "osv-sync.json")
	st := syncpkg.NewStateStore(path)

	if got := st.Load(); !got.IsZero() {
		t.Fatalf("expected zero time before save, got %v", got)
	}
	ts := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	if err := st.Save(ts); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := st.Load(); !got.Equal(ts) {
		t.Fatalf("load: got %v want %v", got, ts)
	}
}

func TestStateStore_Disabled(t *testing.T) {
	st := syncpkg.NewStateStore("")
	if !st.Load().IsZero() {
		t.Fatal("disabled store should load zero time")
	}
	if err := st.Save(time.Now()); err != nil {
		t.Fatalf("disabled save should be no-op, got %v", err)
	}
}

func TestStateStore_MissingFileLoadsZero(t *testing.T) {
	st := syncpkg.NewStateStore(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if !st.Load().IsZero() {
		t.Fatal("missing file should load zero time")
	}
}

func TestStateStore_CorruptFileLoadsZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "osv-sync.json")
	if err := writeFile(path, []byte("{not json")); err != nil {
		t.Fatalf("write: %v", err)
	}
	st := syncpkg.NewStateStore(path)
	if !st.Load().IsZero() {
		t.Fatal("corrupt file should load zero time")
	}
}

func TestSync_ResumesFromState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "osv-sync.json")
	// Pre-seed the state with a lastSync so the Sync skips the bulk import.
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	st := syncpkg.NewStateStore(statePath)
	if err := st.Save(ts); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	vulns, _, sync, srv := newHarnessState(t, st)
	defer srv.Close()

	// The server returns a modified_id.csv newer than the seeded timestamp;
	// no all.zip request should arrive since the Sync starts incrementally.
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/all.zip" {
			t.Errorf("bulk fetch should not occur; got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.URL.Path {
		case "/modified_id.csv":
			_, _ = w.Write([]byte("2024-02-01T00:00:00Z,PyPI/PYSEC-1\n"))
		case "/PyPI/PYSEC-1.json":
			_, _ = w.Write([]byte(`{"id":"PYSEC-1","summary":"a","modified":"2024-02-01T00:00:00Z"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	n, err := sync.Once(context.Background())
	if err != nil {
		t.Fatalf("once: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 upsert, got %d", n)
	}
	// State advanced and was persisted.
	if got := st.Load(); !got.Equal(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("persisted lastSync: %v", got)
	}
	_ = vulns
}
