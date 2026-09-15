// Package migration provides a one-time import of existing schwachstellen
// WAL events into the new file-based store. After migration the WAL file can
// be archived; the store becomes the single source of truth.
package migration

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/schwachstellen/domain"
	"github.com/mwildt/jansvca/service/internal/schwachstellen/store"
)

// FromWAL replays every vulnerability stream of the given WAL event store and
// upserts the resulting final state into the store. It returns the number of
// records migrated. Streams that end in a deletion produce a deleted record so
// the index stays consistent. The WAL is read-only here.
func FromWAL(es *eventstore.Store, s *store.Store) (int, error) {
	streams := groupByStream(es.All())
	states := map[string]*domain.Vulnerability{}
	for _, envs := range streams {
		if len(envs) == 0 {
			continue
		}
		var v *domain.Vulnerability
		for _, env := range envs {
			v = applyEvent(v, env)
		}
		if v == nil || v.ID == "" {
			continue
		}
		states[v.ID] = v
	}
	n := 0
	for _, id := range sortedIDs(states) {
		v := states[id]
		rec := toRecord(v)
		if err := s.Put(rec); err != nil {
			return n, fmt.Errorf("migration: put %s: %w", v.ID, err)
		}
		n++
	}
	return n, nil
}

// groupByStream buckets envelopes by StreamID, preserving per-stream order.
func groupByStream(envs []eventstore.Envelope) map[eventstore.StreamID][]eventstore.Envelope {
	out := map[eventstore.StreamID][]eventstore.Envelope{}
	for _, env := range envs {
		out[env.StreamID] = append(out[env.StreamID], env)
	}
	return out
}

func sortedIDs(m map[string]*domain.Vulnerability) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func applyEvent(v *domain.Vulnerability, env eventstore.Envelope) *domain.Vulnerability {
	switch env.Type {
	case domain.EventVulnerabilityCreated:
		var e domain.VulnerabilityCreated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return v
		}
		return &domain.Vulnerability{
			ID:          e.VulnerabilityID,
			Identifier:  e.Identifier,
			Title:       e.Title,
			Description: e.Description,
			CVSS:        e.CVSS,
			Source:      e.Source,
			Affected:    map[string]domain.AffectedRange{},
		}
	case domain.EventVulnerabilityUpdated:
		var e domain.VulnerabilityUpdated
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return v
		}
		if v == nil {
			return nil
		}
		if e.Title != "" {
			v.Title = e.Title
		}
		if e.Description != "" {
			v.Description = e.Description
		}
		if e.CVSS != 0 {
			v.CVSS = e.CVSS
		}
		return v
	case domain.EventVulnerabilityDeleted:
		if v == nil {
			return nil
		}
		v.Deleted = true
		return v
	case domain.EventAffectedRangeAdded:
		var e domain.AffectedRangeAdded
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return v
		}
		if v == nil {
			return nil
		}
		if v.Affected == nil {
			v.Affected = map[string]domain.AffectedRange{}
		}
		v.Affected[e.Component] = domain.AffectedRange{Component: e.Component, VersionRange: e.VersionRange}
		return v
	case domain.EventAffectedRangeRemoved:
		var e domain.AffectedRangeRemoved
		if err := json.Unmarshal(env.Payload, &e); err != nil {
			return v
		}
		if v == nil {
			return nil
		}
		delete(v.Affected, e.Component)
		return v
	}
	return v
}

func toRecord(v *domain.Vulnerability) store.Record {
	rec := store.Record{
		ID:          v.ID,
		Identifier:  v.Identifier,
		Title:       v.Title,
		Description: v.Description,
		CVSS:        v.CVSS,
		Source:      v.Source,
		Deleted:     v.Deleted,
	}
	for _, a := range v.Affected {
		rec.Affected = append(rec.Affected, store.AffectedRange{
			Component:    a.Component,
			VersionRange: a.VersionRange,
			Ecosystem:    a.Ecosystem,
		})
	}
	return rec
}
