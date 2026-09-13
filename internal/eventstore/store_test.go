package eventstore

import (
	"path/filepath"
	"testing"
)

type sampleEvent struct {
	Value string
}

func (sampleEvent) EventType() EventType { return "test.sample" }

func TestAppendAndLoad(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "test.wal"))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer s.Close()

	id := StreamID("agg-1")
	if v := s.CurrentVersion(id); v != 0 {
		t.Fatalf("expected version 0 for new stream, got %d", v)
	}
	recorded, err := s.Append(id, 0, []PayloadEvent{
		sampleEvent{Value: "a"},
		sampleEvent{Value: "b"},
	})
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	if len(recorded) != 2 {
		t.Fatalf("expected 2 recorded, got %d", len(recorded))
	}
	if recorded[0].Version != 1 || recorded[1].Version != 2 {
		t.Fatalf("unexpected versions: %+v", recorded)
	}
	if s.CurrentVersion(id) != 2 {
		t.Fatalf("expected current version 2, got %d", s.CurrentVersion(id))
	}

	envs, err := s.Load(id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(envs) != 2 {
		t.Fatalf("expected 2 loaded, got %d", len(envs))
	}
	if envs[1].Version != 2 {
		t.Fatalf("expected last version 2, got %d", envs[1].Version)
	}
}

func TestReplayRebuilds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.wal")

	s1, err := New(path)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	id := StreamID("agg-1")
	if _, err := s1.Append(id, 0, []PayloadEvent{sampleEvent{Value: "x"}}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := s1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	s2, err := New(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	if s2.CurrentVersion(id) != 1 {
		t.Fatalf("expected version 1 after replay, got %d", s2.CurrentVersion(id))
	}
	envs, err := s2.Load(id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(envs) != 1 {
		t.Fatalf("expected 1 env after replay, got %d", len(envs))
	}
}

func TestConcurrencyConflict(t *testing.T) {
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "test.wal"))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	defer s.Close()
	id := StreamID("agg-1")
	if _, err := s.Append(id, 0, []PayloadEvent{sampleEvent{Value: "a"}}); err != nil {
		t.Fatalf("append: %v", err)
	}
	_, err = s.Append(id, 0, []PayloadEvent{sampleEvent{Value: "b"}})
	if err == nil {
		t.Fatal("expected concurrency error, got nil")
	}
}

func TestBusDispatch(t *testing.T) {
	bus := NewBus()
	var got []EventType
	bus.Subscribe(nil, func(env Envelope) {
		got = append(got, env.Type)
	})
	bus.Publish(Envelope{Type: "a"})
	bus.Publish(Envelope{Type: "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected dispatch: %+v", got)
	}
}
