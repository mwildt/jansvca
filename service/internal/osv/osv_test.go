package osv

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseRecord(t *testing.T) {
	rec, err := ParseRecord([]byte(`{"id":"GHSA-1","modified":"2024-01-02T03:04:05Z","summary":"x"}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rec.ID != "GHSA-1" {
		t.Errorf("id: %q", rec.ID)
	}
}

func TestParseRecord_MissingID(t *testing.T) {
	if _, err := ParseRecord([]byte(`{"modified":"2024-01-02T03:04:05Z"}`)); err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestParseRecordsFromZip(t *testing.T) {
	zb := mustZip(t, map[string]string{
		"PyPI/PYSEC-1.json": `{"id":"PYSEC-1","summary":"a"}`,
		"Go/GO-1.json":      `{"id":"GO-1","summary":"b"}`,
	})
	recs, err := ParseRecordsFromZip(bytes.NewReader(zb))
	if err != nil {
		t.Fatalf("parse zip: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("expected 2 records, got %d", len(recs))
	}
	ids := map[string]bool{}
	for _, r := range recs {
		ids[r.ID] = true
	}
	if !ids["PYSEC-1"] || !ids["GO-1"] {
		t.Errorf("missing ids: %+v", ids)
	}
}

func TestParseRecordsFromZip_SkipsMalformed(t *testing.T) {
	zb := mustZip(t, map[string]string{
		"ok.json":  `{"id":"OK","summary":"a"}`,
		"bad.json": `{"id":"BAD", invalid}`,
	})
	recs, err := ParseRecordsFromZip(bytes.NewReader(zb))
	if err != nil {
		t.Fatalf("parse zip: %v", err)
	}
	if len(recs) != 1 || recs[0].ID != "OK" {
		t.Fatalf("expected only OK, got %+v", recs)
	}
}

func TestReadModifiedSince(t *testing.T) {
	csv := strings.Join([]string{
		"2024-03-01T00:00:00Z,PyPI/PYSEC-3",
		"2024-02-01T00:00:00Z,Go/GO-2",
		"2024-01-01T00:00:00Z,PyPI/PYSEC-1",
		"2023-12-01T00:00:00Z,PyPI/PYSEC-0",
	}, "\n")
	entries, err := ReadModifiedSince(strings.NewReader(csv), time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	// Entries strictly newer than 2024-01-01: PYSEC-3 and GO-2. The
	// entry at exactly 2024-01-01 (PYSEC-1) is treated as already seen.
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %+v", len(entries), entries)
	}
	if entries[0].Path != "PyPI/PYSEC-3" || entries[1].Path != "Go/GO-2" {
		t.Errorf("unexpected entries: %+v", entries)
	}
}

func mustZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
