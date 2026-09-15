package osv

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_FetchAllZip(t *testing.T) {
	zb := mustZip(t, map[string]string{
		"PyPI/PYSEC-1.json": `{"id":"PYSEC-1","summary":"a"}`,
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/all.zip" {
			t.Errorf("path: %q", r.URL.Path)
		}
		_, _ = w.Write(zb)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	recs, err := c.FetchAllZip(context.Background())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(recs) != 1 || recs[0].ID != "PYSEC-1" {
		t.Fatalf("records: %+v", recs)
	}
}

func TestClient_FetchModifiedSince(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/modified_id.csv" {
			t.Errorf("path: %q", r.URL.Path)
		}
		_, _ = w.Write([]byte("2024-03-01T00:00:00Z,PyPI/PYSEC-3\n2024-01-01T00:00:00Z,PyPI/PYSEC-1\n"))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	entries, err := c.FetchModifiedSince(context.Background(), time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != "PyPI/PYSEC-3" {
		t.Fatalf("entries: %+v", entries)
	}
}

func TestClient_FetchRecord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/PyPI/PYSEC-1.json" {
			t.Errorf("path: %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"PYSEC-1","summary":"a"}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	rec, err := c.FetchRecord(context.Background(), "PyPI/PYSEC-1")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if rec.ID != "PYSEC-1" {
		t.Errorf("id: %q", rec.ID)
	}
}

func TestClient_FetchRecord_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	if _, err := c.FetchRecord(context.Background(), "PyPI/PYSEC-x"); err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestNewClient_Defaults(t *testing.T) {
	c := NewClient()
	if c.BaseURL != DefaultBaseURL {
		t.Errorf("base url: %q", c.BaseURL)
	}
	if c.HTTPClient == nil {
		t.Error("http client nil")
	}
}

func TestClient_FetchEcosystemZip(t *testing.T) {
	zb := mustZip(t, map[string]string{
		"GO-1.json": `{"id":"GO-1","summary":"a"}`,
	})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Go/all.zip" {
			t.Errorf("path: %q", r.URL.Path)
		}
		_, _ = w.Write(zb)
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	recs, err := c.FetchEcosystemZip(context.Background(), "Go")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(recs) != 1 || recs[0].ID != "GO-1" {
		t.Fatalf("records: %+v", recs)
	}
}

func TestClient_FetchEcosystemModifiedSince(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Go/modified_id.csv" {
			t.Errorf("path: %q", r.URL.Path)
		}
		// Per-ecosystem CSV omits the ecosystem prefix.
		_, _ = w.Write([]byte("2024-03-01T00:00:00Z,GO-2024-3\n2024-01-01T00:00:00Z,GO-2024-1\n"))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	entries, err := c.FetchEcosystemModifiedSince(context.Background(), "Go", time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(entries) != 1 || entries[0].Path != "GO-2024-3" {
		t.Fatalf("entries: %+v", entries)
	}
}

func TestClient_FetchEcosystemRecord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Go/GO-1.json" {
			t.Errorf("path: %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"id":"GO-1","summary":"a"}`))
	}))
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, HTTPClient: srv.Client()}
	rec, err := c.FetchEcosystemRecord(context.Background(), "Go", "GO-1")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if rec.ID != "GO-1" {
		t.Errorf("id: %q", rec.ID)
	}
}
