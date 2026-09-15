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

	if got := st.Load(); len(got) != 0 {
		t.Fatalf("expected empty map before save, got %v", got)
	}
	ts := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	if err := st.Save(map[string]time.Time{"Go": ts}); err != nil {
		t.Fatalf("save: %v", err)
	}
	got := st.Load()
	if len(got) != 1 || !got["Go"].Equal(ts) {
		t.Fatalf("load: got %v want {Go:%v}", got, ts)
	}
}

func TestStateStore_Disabled(t *testing.T) {
	st := syncpkg.NewStateStore("")
	if got := st.Load(); len(got) != 0 {
		t.Fatal("disabled store should load empty map")
	}
	if err := st.Save(map[string]time.Time{"Go": time.Now()}); err != nil {
		t.Fatalf("disabled save should be no-op, got %v", err)
	}
}

func TestStateStore_MissingFileLoadsEmpty(t *testing.T) {
	st := syncpkg.NewStateStore(filepath.Join(t.TempDir(), "does-not-exist.json"))
	if got := st.Load(); len(got) != 0 {
		t.Fatal("missing file should load empty map")
	}
}

func TestStateStore_CorruptFileLoadsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "osv-sync.json")
	if err := writeFile(path, []byte("{not json")); err != nil {
		t.Fatalf("write: %v", err)
	}
	st := syncpkg.NewStateStore(path)
	if got := st.Load(); len(got) != 0 {
		t.Fatal("corrupt file should load empty map")
	}
}

// TestStateStore_LegacySingleTimestamp verifies that a legacy osv-sync.json
// with only a top-level last_sync (no per-ecosystem map) loads as an empty map
// so the sync re-runs a fresh per-ecosystem bulk import instead of skipping.
func TestStateStore_LegacySingleTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "osv-sync.json")
	legacy := []byte(`{"last_sync":"2024-01-01T00:00:00Z"}`)
	if err := writeFile(path, legacy); err != nil {
		t.Fatalf("write: %v", err)
	}
	st := syncpkg.NewStateStore(path)
	if got := st.Load(); len(got) != 0 {
		t.Fatalf("legacy single-timestamp state should load empty, got %v", got)
	}
}

func TestStateStore_SaveEcosystemMerges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "osv-sync.json")
	st := syncpkg.NewStateStore(path)
	goTS := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := st.SaveEcosystem("Go", goTS); err != nil {
		t.Fatalf("save Go: %v", err)
	}
	npmTS := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	if err := st.SaveEcosystem("npm", npmTS); err != nil {
		t.Fatalf("save npm: %v", err)
	}
	got := st.Load()
	if len(got) != 2 || !got["Go"].Equal(goTS) || !got["npm"].Equal(npmTS) {
		t.Fatalf("merged: got %v", got)
	}
}

func TestSync_ResumesFromState(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "osv-sync.json")
	// Pre-seed the state with a per-ecosystem lastSync for "PyPI" so the Sync
	// skips the bulk import for that ecosystem and runs incrementally.
	ts := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	st := syncpkg.NewStateStore(statePath)
	if err := st.Save(map[string]time.Time{"PyPI": ts}); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	vulns, _, sync, srv := newHarnessState(t, st, "PyPI")
	defer srv.Close()

	// The server returns a per-ecosystem modified_id.csv newer than the seeded
	// timestamp; no PyPI all.zip request should arrive since the Sync starts
	// incrementally for PyPI.
	srv.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/PyPI/all.zip" {
			t.Errorf("bulk fetch should not occur; got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.URL.Path {
		case "/PyPI/modified_id.csv":
			_, _ = w.Write([]byte("2024-02-01T00:00:00Z,PYSEC-1\n"))
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
	// State advanced and was persisted for PyPI.
	if got := st.Load()["PyPI"]; !got.Equal(time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("persisted PyPI lastSync: %v", got)
	}
	_ = vulns
}
