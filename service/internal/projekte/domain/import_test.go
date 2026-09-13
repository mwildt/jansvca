package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mwildt/jansvca/service/internal/eventstore"
)

func TestImportComponents_AddsNewAndUpdatesChanged(t *testing.T) {
	p := &Project{ID: "p1", Components: map[string]Component{
		"pkg:npm/lit": {Component: "pkg:npm/lit", Version: "3.0.0"},
	}}

	events, err := p.ImportComponents([]ImportInput{
		{Component: "pkg:npm/lit", Version: "3.0.0"},
		{Component: "pkg:npm/lit", Version: "3.2.1"},
		{Component: "pkg:npm/vite", Version: "6.0.0"},
		{Component: "", Version: "1.0.0"},
		{Component: "pkg:npm/empty", Version: ""},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Apply the events through the aggregate's Apply pipeline.
	for i, e := range events {
		env := envelopeForTest(t, e, eventstore.Version(i+1))
		if err := p.Apply(env); err != nil {
			t.Fatalf("apply %T: %v", e, err)
		}
	}
	if got := p.Components["pkg:npm/lit"].Version; got != "3.2.1" {
		t.Errorf("expected lit version 3.2.1, got %q", got)
	}
	if _, ok := p.Components["pkg:npm/vite"]; !ok {
		t.Errorf("expected vite to be added")
	}
}

func TestImportComponents_ConflictingVersions(t *testing.T) {
	p := &Project{ID: "p1", Components: map[string]Component{}}
	_, err := p.ImportComponents([]ImportInput{
		{Component: "pkg:npm/lit", Version: "3.0.0"},
		{Component: "pkg:npm/lit", Version: "3.1.0"},
	})
	if err == nil {
		t.Fatalf("expected conflicting-version error, got nil")
	}
	if want := "conflicting versions"; !contains(err.Error(), want) {
		t.Fatalf("expected error mentioning %q, got %q", want, err.Error())
	}
}

func TestImportComponents_DeletedProject(t *testing.T) {
	p := &Project{ID: "p1", Deleted: true, Components: map[string]Component{}}
	_, err := p.ImportComponents([]ImportInput{{Component: "x", Version: "1.0.0"}})
	if !errors.Is(err, ErrProjectDeleted) {
		t.Fatalf("expected ErrProjectDeleted, got %v", err)
	}
}

func TestImportComponents_NotFound(t *testing.T) {
	p := &Project{}
	_, err := p.ImportComponents([]ImportInput{{Component: "x", Version: "1.0.0"}})
	if err == nil {
		t.Fatalf("expected not-found error, got nil")
	}
}

func envelopeForTest(t *testing.T, e eventstore.PayloadEvent, v eventstore.Version) eventstore.Envelope {
	t.Helper()
	payload, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	return eventstore.Envelope{Type: e.EventType(), Version: v, Payload: payload}
}

func contains(s, substr string) bool { return strings.Contains(s, substr) }
