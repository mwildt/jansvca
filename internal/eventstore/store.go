// Package eventstore provides an append-only, Write-Ahead-Log (WAL) backed
// event store for event-sourced aggregates.
//
// Each store instance owns a single append-only log. Streams are identified
// by a StreamID; the store preserves event order and exposes optimistic
// concurrency via per-stream versions.
package eventstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventID uniquely identifies an event within the store.
type EventID string

// StreamID identifies an aggregate event stream.
type StreamID string

// Version is the monotonic, per-stream sequence number of an event.
// The first event in a stream has Version 1.
type Version uint64

// EventType is the discriminator of a domain event.
type EventType string

// Envelope is the persisted representation of a single domain event.
type Envelope struct {
	EventID   EventID   `json:"event_id"`
	StreamID  StreamID  `json:"stream_id"`
	Type      EventType `json:"type"`
	Version   Version   `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Payload   []byte    `json:"payload"`
}

// PayloadEvent is a domain event that can marshal/unmarshal itself to JSON.
type PayloadEvent interface {
	EventType() EventType
}

// RecordedEvent couples an event with the recorded envelope metadata for
// handlers that need both the typed payload and the metadata.
type RecordedEvent struct {
	Envelope Envelope
	Payload  PayloadEvent
}

// ErrConcurrency is returned by Append when the expected version does not
// match the current stream version.
var ErrConcurrency = errors.New("eventstore: concurrency conflict")

// ErrStreamNotFound is returned by Load when no events exist for the stream.
var ErrStreamNotFound = errors.New("eventstore: stream not found")

// Store is an append-only, WAL-backed event store.
//
// It is safe for concurrent use. Persistence is append-only to a single file;
// entries are durable on fsync before Append returns.
type Store struct {
	mu   sync.Mutex
	path string
	file *os.File
	enc  *json.Encoder

	// streams maps StreamID -> ordered envelopes (in-memory index).
	streams map[StreamID][]Envelope
}

// New opens or creates a WAL event store at the given file path.
func New(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("eventstore: create dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("eventstore: open wal: %w", err)
	}
	s := &Store{
		path:    path,
		file:    f,
		enc:     json.NewEncoder(f),
		streams: make(map[StreamID][]Envelope),
	}
	if err := s.replay(); err != nil {
		_ = f.Close()
		return nil, err
	}
	return s, nil
}

// replay rebuilds the in-memory index from the existing WAL contents.
func (s *Store) replay() error {
	dec := json.NewDecoder(s.file)
	for {
		var env Envelope
		if err := dec.Decode(&env); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("eventstore: replay decode: %w", err)
		}
		s.streams[env.StreamID] = append(s.streams[env.StreamID], env)
	}
	return nil
}

// CurrentVersion returns the last version for a stream, or 0 if empty.
func (s *Store) CurrentVersion(id StreamID) Version {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := s.streams[id]
	if len(events) == 0 {
		return 0
	}
	return events[len(events)-1].Version
}

// Load returns all events for a stream in order.
func (s *Store) Load(id StreamID) ([]Envelope, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	events, ok := s.streams[id]
	if !ok || len(events) == 0 {
		return nil, ErrStreamNotFound
	}
	out := make([]Envelope, len(events))
	copy(out, events)
	return out, nil
}

// Append persists events to the WAL and updates the in-memory index.
//
// expectedVersion is used for optimistic concurrency: pass 0 for a new stream
// or the current version returned by CurrentVersion/Load. Mismatch yields
// ErrConcurrency.
//
// Payloads are encoded as JSON; each event is assigned a monotonically
// increasing per-stream version.
func (s *Store) Append(id StreamID, expectedVersion Version, events []PayloadEvent) ([]Envelope, error) {
	if len(events) == 0 {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	current := s.streams[id]
	last := Version(0)
	if len(current) > 0 {
		last = current[len(current)-1].Version
	}
	if expectedVersion != last {
		return nil, fmt.Errorf("%w: stream %s expected %d got %d", ErrConcurrency, id, expectedVersion, last)
	}

	now := time.Now().UTC()
	recorded := make([]Envelope, 0, len(events))
	next := last
	for _, e := range events {
		payload, err := json.Marshal(e)
		if err != nil {
			return nil, fmt.Errorf("eventstore: marshal payload: %w", err)
		}
		next++
		env := Envelope{
			EventID:   EventID(fmt.Sprintf("%s-%d", id, next)),
			StreamID:  id,
			Type:      e.EventType(),
			Version:   next,
			Timestamp: now,
			Payload:   payload,
		}
		recorded = append(recorded, env)
	}

	for _, env := range recorded {
		if err := s.enc.Encode(&env); err != nil {
			return nil, fmt.Errorf("eventstore: write wal: %w", err)
		}
	}
	if err := s.file.Sync(); err != nil {
		return nil, fmt.Errorf("eventstore: fsync wal: %w", err)
	}
	s.streams[id] = append(current, recorded...)
	return recorded, nil
}

// All returns every envelope in the store, grouped by stream, in append order.
// Useful for projections/read-model rebuilds.
func (s *Store) All() []Envelope {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Envelope
	for _, events := range s.streams {
		out = append(out, events...)
	}
	return out
}

// Close flushes and closes the underlying WAL file.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	return err
}
