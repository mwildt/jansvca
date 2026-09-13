package sbom

import (
	"errors"
	"testing"
)

const sampleBOM = `{
  "bomFormat": "CycloneDX",
  "specVersion": "1.5",
  "components": [
    {"type": "library", "name": "lit", "version": "3.2.1", "purl": "pkg:npm/lit@3.2.1"},
    {"type": "library", "name": "vite", "version": "6.0.0", "purl": "pkg:npm/vite@6.0.0"},
    {"type": "library", "name": "rack", "version": "2.2.0"},
    {"type": "library", "name": "noversion"},
    {"type": "library", "version": "1.0.0"},
    {"type": "library", "name": "bybomRef", "version": "1.2.3", "bom-ref": "ref-123"}
  ]
}`

func TestParse_CycloneDX(t *testing.T) {
	out, err := Parse([]byte(sampleBOM))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := 4
	if len(out) != want {
		t.Fatalf("expected %d components, got %d: %+v", want, len(out), out)
	}
	got := map[string]string{}
	for _, c := range out {
		got[c.Component] = c.Version
	}
	if got["pkg:npm/lit@3.2.1"] != "3.2.1" {
		t.Errorf("expected purl-based identifier for lit, got %+v", got)
	}
	if got["rack@2.2.0"] != "2.2.0" {
		t.Errorf("expected name@version fallback for rack, got %+v", got)
	}
	if got["bybomRef"] != "1.2.3" {
		t.Errorf("expected bom-ref fallback, got %+v", got)
	}
}

func TestParse_Empty(t *testing.T) {
	if _, err := Parse([]byte("   ")); err == nil {
		t.Fatalf("expected error for empty document")
	}
}

func TestParse_NonJSON(t *testing.T) {
	_, err := Parse([]byte("<xml>not json</xml>"))
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestParse_WrongBomFormat(t *testing.T) {
	_, err := Parse([]byte(`{"bomFormat":"SPDX","components":[]}`))
	if err == nil {
		t.Fatalf("expected error for non-CycloneDX bomFormat")
	}
}

func TestParse_NoComponents(t *testing.T) {
	out, err := Parse([]byte(`{"bomFormat":"CycloneDX","specVersion":"1.4","components":[]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected 0 components, got %d", len(out))
	}
}
