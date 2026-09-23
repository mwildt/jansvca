package store_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	res, err = s.Query(store.Query{Component: "pkg:gem/rails", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("query component: %v", err)
	}
	if res.Total != 1 || res.Items[0].ID != "V1" {
		t.Fatalf("component filter: %+v", res)
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
	if err := s.Put(store.Record{ID: "GO-2024-0001", Identifier: "I", Title: "T", CVSS: 1.0, Ecosystems: []string{"Go"}}); err != nil {
		t.Fatalf("put: %v", err)
	}
	_ = s.Close()

	// The file lives under the hierarchical layout
	// <ecosystem>/<year>/<range>/<id>.json.
	rel := findVulnFile(t, dir, "GO-2024-0001.json")
	if !strings.HasPrefix(rel, "Go/2024/") {
		t.Fatalf("expected hierarchical path Go/2024/<range>/..., got %q", rel)
	}

	s2, err := store.New(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	t.Cleanup(func() { _ = s2.Close() })
	if g := mustGet(t, s2, "GO-2024-0001"); g == nil || g.Title != "T" {
		t.Fatalf("after reopen: %+v", g)
	}
	res, _ := s2.Query(store.Query{Page: 1, PageSize: 10})
	if res.Total != 1 {
		t.Fatalf("total after reopen: %d", res.Total)
	}
}

// findVulnFile walks the vulnerabilities tree and returns the relative path of
// the file whose base name matches name, or fails the test.
func findVulnFile(t *testing.T, dir, name string) string {
	t.Helper()
	root := filepath.Join(dir, "vulnerabilities")
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if d.Name() == name {
			found, _ = filepath.Rel(root, path)
		}
		return nil
	})
	if found == "" {
		t.Fatalf("file %s not found under %s", name, root)
	}
	return filepath.ToSlash(found)
}

// TestStore_HierarchicalBuckets verifies that 100+ records of the same
// ecosystem/year are split across range buckets of filesPerRange (100) files.
func TestStore_HierarchicalBuckets(t *testing.T) {
	dir := t.TempDir()
	s, err := store.New(dir)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	for i := 0; i < 105; i++ {
		id := fmt.Sprintf("GO-2024-%04d", i)
		if err := s.Put(store.Record{
			ID:         id,
			Identifier: id,
			Title:      "T",
			CVSS:       1.0,
			Ecosystems: []string{"Go"},
		}); err != nil {
			t.Fatalf("put %s: %v", id, err)
		}
	}
	root := filepath.Join(dir, "vulnerabilities", "Go", "2024")
	ranges, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read Go/2024: %v", err)
	}
	var names []string
	for _, d := range ranges {
		if d.IsDir() {
			names = append(names, d.Name())
		}
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "0" || names[1] != "100" {
		t.Fatalf("expected range buckets [0 100], got %+v", names)
	}
	first, _ := os.ReadDir(filepath.Join(root, "0"))
	second, _ := os.ReadDir(filepath.Join(root, "100"))
	if len(first) != 100 || len(second) != 5 {
		t.Fatalf("bucket sizes: 0=%d 100=%d (want 100 and 5)", len(first), len(second))
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
