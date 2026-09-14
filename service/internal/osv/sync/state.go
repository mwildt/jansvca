package sync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// stateFile is the JSON schema of the persisted sync progress. Only lastSync
// is tracked for now; the record is intentionally minimal and forward
// compatible (extra fields are ignored on load).
type stateFile struct {
	LastSync time.Time `json:"last_sync"`
}

// StateStore persists the sync progress (last seen modified timestamp) so an
// incremental sync can resume after a restart instead of re-downloading
// all.zip.
type StateStore struct {
	Path string
}

// NewStateStore creates a state store backed by path. A path of "" disables
// persistence (load returns the zero time, save is a no-op), which keeps the
// Sync usable without a data directory (e.g. in tests).
func NewStateStore(path string) *StateStore {
	return &StateStore{Path: path}
}

// Load reads the persisted lastSync. Returns the zero time if the store is
// disabled, the file does not exist, or the record is empty/corrupt.
func (s *StateStore) Load() time.Time {
	if s.Path == "" {
		return time.Time{}
	}
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return time.Time{}
	}
	var st stateFile
	if err := json.Unmarshal(data, &st); err != nil {
		return time.Time{}
	}
	return st.LastSync
}

// Save persists lastSync. It is a no-op when the store is disabled. Errors
// are returned so callers can log them without aborting the sync.
func (s *StateStore) Save(lastSync time.Time) error {
	if s.Path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("osv sync state: create dir: %w", err)
	}
	data, err := json.Marshal(stateFile{LastSync: lastSync})
	if err != nil {
		return fmt.Errorf("osv sync state: marshal: %w", err)
	}
	if err := os.WriteFile(s.Path, data, 0o644); err != nil {
		return fmt.Errorf("osv sync state: write: %w", err)
	}
	return nil
}
