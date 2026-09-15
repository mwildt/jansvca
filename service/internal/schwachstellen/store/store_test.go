package store_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mwildt/jansvca/service/internal/schwachstellen/store"
)

func TestStore_PutGetQuery(t *testing.T) {
	dir := t.TempDir()
	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.Put(store.Record{
		ID:          "V1",
		Identifier:  "CVE-2024-0001",
		Title:       "RCE in rails",
		Description: "bad",
		CVSS:        9.8,
		Source:      "manual",
		Ecosystems:  []string{"RubyGems"},
		Affected: []store.AffectedRange{
			{Component: "pkg:gem/rails", VersionRange: ">=1.0.0 <2.0.0", Ecosystem: "RubyGems"},
		},
	}); err != nil {
		t.Fatalf("put: %v", err)
	}
	if err := s.Put(store.Record{
		ID: "V2", Identifier: "CVE-2024-0002", Title: "XSS in lit", CVSS: 4.3, Source: "osv",
		Ecosystems: []string{"npm"},
		Affected: []store.AffectedRange{
			{Component: "pkg:npm/lit", VersionRange: "<3.2.2", Ecosystem: "npm"},
		},
	}); err != nil {
		t.Fatalf("put v2: %v", err)
	}

	got := mustGet(t, s, "V1")
	if got.Title != "RCE in rails" || got.CVSS != 9.8 {
		t.Errorf("get: %+v", got)
	}

	res, err := s.Query(store.Query{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if res.Total != 2 {
		t.Fatalf("total: %d", res.Total)
	}

	res, err = s.Query(store.Query{MinCVSS: 7, HasMinCVSS: true, Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("query cvss: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != "V1" {
		t.Fatalf("cvss filter: %+v", res)
	}

	res, err = s.Query(store.Query{Ecosystem: "npm", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("query eco: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != "V2" {
		t.Fatalf("ecosystem filter: %+v", res)
	}

	ids := s.ComponentsFor("pkg:gem/rails")
	if len(ids) != 1 || ids[0] != "V1" {
		t.Fatalf("components for rails: %+v", ids)
	}

	if err := s.Put(store.Record{ID: "V1", Deleted: true}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if g, _ := s.Get("V1"); g != nil {
		t.Fatal("V1 should be gone after delete")
	}
	res, _ = s.Query(store.Query{Page: 1, PageSize: 10})
	if res.Total != 1 {
		t.Fatalf("total after delete: %d", res.Total)
	}
}

func TestStore_PersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if err := s.Put(store.Record{ID: "V1", Identifier: "I", Title: "T", CVSS: 1.0}); err != nil {
		t.Fatalf("put: %v", err)
	}
	_ = s.Close()

	if _, err := os.ReadFile(filepath.Join(dir, "vulnerabilities", "V1.json")); err != nil {
		t.Fatalf("file: %v", err)
	}

	s2, err := store.New(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })
	if g := mustGet(t, s2, "V1"); g == nil || g.Title != "T" {
		t.Fatalf("after reopen: %+v", g)
	}
	res, _ := s2.Query(store.Query{Page: 1, PageSize: 10})
	if res.Total != 1 {
		t.Fatalf("total after reopen: %d", res.Total)
	}
}

func mustGet(t *testing.T, s *store.Store, id string) *store.Record {
	t.Helper()
	g, err := s.Get(id)
	if err != nil {
		t.Fatalf("get %s: %v", id, err)
	}
	if g == nil {
		t.Fatalf("get %s: nil", id)
	}
	return g
}
