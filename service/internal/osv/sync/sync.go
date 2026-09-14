// Package sync orchestrates importing vulnerabilities from osv.dev into the
// schwachstellen module.
//
// A Sync owns an osv.Client and a set of command ports (VulnerabilityStore) +
// read ports (VulnerabilityRead) of the schwachstellen application. It tracks
// the highest "modified" timestamp it has seen so subsequent runs only fetch
// records newer than the last successful sync.
//
// Sync.Once performs the initial bulk import (all.zip) on the first run and an
// incremental import (modified_id.csv + per-record fetches) on later runs.
// Sync.Run blocks, executing Once repeatedly on the configured interval until
// the context is cancelled.
package sync

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mwildt/jansvca/service/internal/osv"
	vulnapp "github.com/mwildt/jansvca/service/internal/schwachstellen/application"
)

// VulnerabilityStore is the write port used to upsert vulnerabilities.
type VulnerabilityStore interface {
	Import(id, identifier, title, description string, cvss float64, ranges []vulnapp.AffectedRangeInput) error
}

// Sync imports osv.dev vulnerabilities into the schwachstellen module.
type Sync struct {
	Client    *osv.Client
	Store     VulnerabilityStore
	Interval  time.Duration
	OnError   func(err error) // optional; defaults to log.Printf
	BatchSize int             // max records per incremental fetch (0 = unlimited)

	mu       sync.Mutex
	lastSync time.Time
}

// New creates a Sync with sensible defaults. interval is the refresh cadence
// for Run; pass 0 to use the default of one hour.
func New(client *osv.Client, store VulnerabilityStore, interval time.Duration) *Sync {
	if interval <= 0 {
		interval = time.Hour
	}
	if client == nil {
		client = osv.NewClient()
	}
	return &Sync{
		Client:   client,
		Store:    store,
		Interval: interval,
	}
}

func (s *Sync) onError(err error) {
	if s.OnError != nil {
		s.OnError(err)
		return
	}
	log.Printf("osv sync: %v", err)
}

// Once performs a single sync pass. On the first run (no prior state) it
// downloads all.zip; on subsequent runs it applies the modified_id.csv delta.
// It returns the number of records upserted.
func (s *Sync) Once(ctx context.Context) (int, error) {
	s.mu.Lock()
	last := s.lastSync
	s.mu.Unlock()
	if last.IsZero() {
		return s.bulk(ctx)
	}
	return s.incremental(ctx, last)
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

func (s *Sync) bulk(ctx context.Context) (int, error) {
	records, err := s.Client.FetchAllZip(ctx)
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
	s.mu.Lock()
	if latest.IsZero() {
		s.lastSync = time.Now().UTC()
	} else {
		s.lastSync = latest
	}
	s.mu.Unlock()
	return n, nil
}

func (s *Sync) incremental(ctx context.Context, since time.Time) (int, error) {
	entries, err := s.Client.FetchModifiedSince(ctx, since)
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
		rec, err := s.Client.FetchRecord(ctx, e.Path)
		if err != nil {
			s.onError(fmt.Errorf("fetch %s: %w", e.Path, err))
			continue
		}
		imp := osv.Map(rec)
		if err := s.upsert(ctx, imp); err != nil {
			s.onError(fmt.Errorf("upsert %s: %w", imp.ID, err))
			continue
		}
		n++
	}
	s.mu.Lock()
	if latest.After(s.lastSync) {
		s.lastSync = latest
	}
	s.mu.Unlock()
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
		})
	}
	return s.Store.Import(imp.ID, imp.Identifier, imp.Title, imp.Description, imp.CVSS, ranges)
}

// LastSync returns the timestamp of the most recent successful sync (the
// highest OSV "modified" time seen).
func (s *Sync) LastSync() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSync
}
