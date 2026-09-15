// Package sync orchestrates importing vulnerabilities from osv.dev into the
// schwachstellen module.
//
// A Sync owns an osv.Client and a set of command ports (VulnerabilityStore) +
// read ports (VulnerabilityRead) of the schwachstellen application. It imports
// only the configured ecosystems (default Go), fetching the per-ecosystem
// all.zip for the initial bulk import and the per-ecosystem modified_id.csv
// for incremental updates. Per-ecosystem progress is persisted so each
// ecosystem resumes independently after a restart.
//
// Sync.Once performs the initial bulk import (per-ecosystem all.zip) for any
// ecosystem without prior state, then an incremental import
// (per-ecosystem modified_id.csv + per-record fetches) for the rest.
// Sync.Run blocks, executing Once repeatedly on the configured interval until
// the context is cancelled.
package sync

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mwildt/jansvca/service/internal/osv"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
)

// DefaultEcosystems is the OSV ecosystem imported when none is configured.
const DefaultEcosystem = "Go"

// VulnerabilityStore is the write port used to upsert vulnerabilities.
type VulnerabilityStore interface {
	Import(id, identifier, title, description string, cvss float64, source string, ranges []vulnapp.AffectedRangeInput) error
}

// Sync imports osv.dev vulnerabilities into the schwachstellen module.
type Sync struct {
	Client     *osv.Client
	Store      VulnerabilityStore
	State      *StateStore // persists per-ecosystem lastSync across restarts (optional)
	Interval   time.Duration
	Ecosystems []string        // ecosystems to import; defaults to [DefaultEcosystem]
	OnError    func(err error) // optional; defaults to log.Printf
	BatchSize  int             // max records per incremental fetch (0 = unlimited)

	mu        sync.Mutex
	lastSyncs map[string]time.Time // ecosystem -> highest modified timestamp seen
}

// New creates a Sync with sensible defaults. interval is the refresh cadence
// for Run; pass 0 to use the default of one hour. state is an optional
// StateStore to resume incremental syncs after a restart; pass nil to keep the
// progress in memory only (first run always downloads per-ecosystem all.zip).
// ecosystems selects which OSV ecosystems to import; an empty list defaults to
// [DefaultEcosystem].
func New(client *osv.Client, store VulnerabilityStore, state *StateStore, interval time.Duration, ecosystems ...string) *Sync {
	if interval <= 0 {
		interval = time.Hour
	}
	if client == nil {
		client = osv.NewClient()
	}
	if len(ecosystems) == 0 {
		ecosystems = []string{DefaultEcosystem}
	}
	ecosystems = normalizeEcosystems(ecosystems)
	s := &Sync{
		Client:     client,
		Store:      store,
		State:      state,
		Interval:   interval,
		Ecosystems: ecosystems,
		lastSyncs:  map[string]time.Time{},
	}
	if state != nil {
		s.lastSyncs = state.Load()
	}
	return s
}

// normalizeEcosystems trims whitespace, drops empty entries and de-duplicates
// while preserving the first-seen order.
func normalizeEcosystems(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, e := range in {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

func (s *Sync) onError(err error) {
	if s.OnError != nil {
		s.OnError(err)
		return
	}
	log.Printf("osv sync: %v", err)
}

// Once performs a single sync pass across every configured ecosystem. For each
// ecosystem without prior state it downloads the per-ecosystem all.zip; for
// ecosystems with prior state it applies the per-ecosystem modified_id.csv
// delta. It returns the total number of records upserted across all
// ecosystems.
func (s *Sync) Once(ctx context.Context) (int, error) {
	s.mu.Lock()
	lastSyncs := make(map[string]time.Time, len(s.lastSyncs))
	for k, v := range s.lastSyncs {
		lastSyncs[k] = v
	}
	s.mu.Unlock()

	total := 0
	for _, eco := range s.Ecosystems {
		last := lastSyncs[eco]
		var n int
		var err error
		if last.IsZero() {
			n, err = s.bulkEcosystem(ctx, eco)
		} else {
			n, err = s.incrementalEcosystem(ctx, eco, last)
		}
		if err != nil {
			return total, fmt.Errorf("ecosystem %s: %w", eco, err)
		}
		total += n
	}
	return total, nil
}

// Run executes Once on the configured interval until ctx is cancelled. The
// first pass runs immediately; errors are reported via OnError but do not stop
// the loop.
func (s *Sync) Run(ctx context.Context) {
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for {
		if _, err := s.Once(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.onError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// bulkEcosystem downloads the per-ecosystem all.zip and upserts every record.
// It records the highest modified timestamp so the next pass runs
// incrementally.
func (s *Sync) bulkEcosystem(ctx context.Context, ecosystem string) (int, error) {
	records, err := s.Client.FetchEcosystemZip(ctx, ecosystem)
	if err != nil {
		return 0, fmt.Errorf("initial bulk fetch: %w", err)
	}
	n := 0
	latest := time.Time{}
	for _, rec := range records {
		imp := osv.Map(rec)
		if imp.Modified.After(latest) {
			latest = imp.Modified
		}
		if err := s.upsert(ctx, imp); err != nil {
			s.onError(fmt.Errorf("upsert %s: %w", imp.ID, err))
			continue
		}
		n++
	}
	s.setEcosystemSync(ecosystem, latest)
	return n, nil
}

// incrementalEcosystem fetches the per-ecosystem modified_id.csv and upserts
// every record newer than since. The CSV paths omit the ecosystem prefix, so
// individual records are fetched via FetchEcosystemRecord which prepends it.
func (s *Sync) incrementalEcosystem(ctx context.Context, ecosystem string, since time.Time) (int, error) {
	entries, err := s.Client.FetchEcosystemModifiedSince(ctx, ecosystem, since)
	if err != nil {
		return 0, fmt.Errorf("fetch modified index: %w", err)
	}
	if s.BatchSize > 0 && len(entries) > s.BatchSize {
		entries = entries[:s.BatchSize]
	}
	n := 0
	latest := since
	for _, e := range entries {
		if e.Modified.After(latest) {
			latest = e.Modified
		}
		rec, err := s.Client.FetchEcosystemRecord(ctx, ecosystem, e.Path)
		if err != nil {
			s.onError(fmt.Errorf("fetch %s/%s: %w", ecosystem, e.Path, err))
			continue
		}
		imp := osv.Map(rec)
		if err := s.upsert(ctx, imp); err != nil {
			s.onError(fmt.Errorf("upsert %s: %w", imp.ID, err))
			continue
		}
		n++
	}
	s.setEcosystemSync(ecosystem, latest)
	return n, nil
}

func (s *Sync) upsert(ctx context.Context, imp osv.ImportRecord) error {
	ranges := make([]vulnapp.AffectedRangeInput, 0, len(imp.Affected))
	for _, a := range imp.Affected {
		if a.Component == "" || a.VersionRange == "" {
			continue
		}
		ranges = append(ranges, vulnapp.AffectedRangeInput{
			Component:    a.Component,
			VersionRange: a.VersionRange,
			Ecosystem:    a.Ecosystem,
		})
	}
	return s.Store.Import(imp.ID, imp.Identifier, imp.Title, imp.Description, imp.CVSS, imp.Source, ranges)
}

// LastSync returns the highest modified timestamp seen across all configured
// ecosystems (the zero time when nothing has been synced yet).
func (s *Sync) LastSync() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	var latest time.Time
	for _, ts := range s.lastSyncs {
		if ts.After(latest) {
			latest = ts
		}
	}
	return latest
}

// LastSyncEcosystem returns the highest modified timestamp seen for a single
// ecosystem (the zero time when it has not been synced yet).
func (s *Sync) LastSyncEcosystem(ecosystem string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSyncs[ecosystem]
}

// setEcosystemSync stores the latest modified timestamp seen for an ecosystem
// during a sync and persists it. A zero latest falls back to the current time
// so an empty all.zip still advances the cursor.
func (s *Sync) setEcosystemSync(ecosystem string, latest time.Time) {
	s.mu.Lock()
	if latest.IsZero() {
		latest = time.Now().UTC()
	}
	s.lastSyncs[ecosystem] = latest
	s.mu.Unlock()
	s.persistEcosystem(ecosystem, latest)
}

// persistEcosystem writes a single ecosystem's progress to the StateStore, if
// configured. Errors are reported via OnError but never abort the sync.
func (s *Sync) persistEcosystem(ecosystem string, last time.Time) {
	if s.State == nil {
		return
	}
	if err := s.State.SaveEcosystem(ecosystem, last); err != nil {
		s.onError(err)
	}
}
