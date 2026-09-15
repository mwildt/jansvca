package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// stateFile is the JSON schema of the persisted sync progress. lastSync is the
// legacy single-timestamp field (kept for backward compatibility with existing
// osv-sync.json files); ecosystems tracks the highest modified timestamp seen
// per ecosystem so per-ecosystem incremental syncs can resume independently.
type stateFile struct {
	LastSync   time.Time            `json:"last_sync,omitempty"`
	Ecosystems map[string]time.Time `json:"ecosystems,omitempty"`
}

// StateStore persists the sync progress so an incremental sync can resume after
// a restart instead of re-downloading all.zip. Progress is tracked per
// ecosystem (map[ecosystem]lastSeenModified); a legacy single lastSync value
// is migrated into every ecosystem on first load for backward compatibility.
type StateStore struct {
	Path string
}

// NewStateStore creates a state store backed by path. A path of "" disables
// persistence (load returns empty maps, save is a no-op), which keeps the Sync
// usable without a data directory (e.g. in tests).
func NewStateStore(path string) *StateStore {
	return &StateStore{Path: path}
}

// Load reads the persisted per-ecosystem progress. Returns an empty (non-nil)
// map if the store is disabled, the file does not exist, or the record is
// empty/corrupt. A legacy single lastSync value cannot be attributed to a
// specific ecosystem, so it is dropped to force a fresh per-ecosystem bulk
// import rather than skipping records that were synced under the old
// all-ecosystem model.
func (s *StateStore) Load() map[string]time.Time {
	out := map[string]time.Time{}
	if s.Path == "" {
		return out
	}
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return out
	}
	var st stateFile
	if err := json.Unmarshal(data, &st); err != nil {
		return out
	}
	if len(st.Ecosystems) > 0 {
		for eco, ts := range st.Ecosystems {
			if eco != "" && !ts.IsZero() {
				out[eco] = ts
			}
		}
	}
	return out
}

// Save persists the per-ecosystem progress map. It is a no-op when the store
// is disabled. Errors are returned so callers can log them without aborting
// the sync.
func (s *StateStore) Save(ecosystems map[string]time.Time) error {
	if s.Path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("osv sync state: create dir: %w", err)
	}
	st := stateFile{Ecosystems: ecosystems}
	data, err := json.Marshal(st)
	if err != nil {
		return fmt.Errorf("osv sync state: marshal: %w", err)
	}
	if err := os.WriteFile(s.Path, data, 0o644); err != nil {
		return fmt.Errorf("osv sync state: write: %w", err)
	}
	return nil
}

// SaveEcosystem persists a single ecosystem's progress merged into the
// currently stored state. It is a convenience wrapper around Save for callers
// that update one ecosystem at a time.
func (s *StateStore) SaveEcosystem(ecosystem string, last time.Time) error {
	cur := s.Load()
	cur[ecosystem] = last
	return s.Save(cur)
}
