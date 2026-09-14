// Package osv parses vulnerability records in the Open Source Vulnerability
// format (https://ossf.github.io/osv-schema/) and maps them to the internal
// schwachstellen model.
//
// The package is I/O free: it operates on byte slices and readers. Network
// access is handled by the Client, which keeps the parsing logic trivially
// testable with httptest.
package osv

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// Record is a single OSV vulnerability entry as served by osv.dev.
type Record struct {
	SchemaVersion    string          `json:"schema_version"`
	ID               string          `json:"id"`
	Published        string          `json:"published"`
	Modified         string          `json:"modified"`
	Withdrawn        string          `json:"withdrawn"`
	Aliases          []string        `json:"aliases"`
	Summary          string          `json:"summary"`
	Details          string          `json:"details"`
	Severity         []Severity      `json:"severity"`
	Affected         []Affected      `json:"affected"`
	DatabaseSpecific json.RawMessage `json:"database_specific"`
}

// Severity is a CVSS (or other) severity score entry.
type Severity struct {
	Type   string `json:"type"`
	Score  string `json:"score"`
	Source string `json:"source"`
}

// Affected describes a package and the version ranges impacted by a
// vulnerability.
type Affected struct {
	Package          *Package        `json:"package"`
	Severity         []Severity      `json:"severity"`
	Ranges           []Range         `json:"ranges"`
	Versions         []string        `json:"versions"`
	DatabaseSpecific json.RawMessage `json:"database_specific"`
}

// Package identifies the affected ecosystem package.
type Package struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
	PURL      string `json:"purl"`
}

// Range describes a set of affected versions, either as explicit version
// events (ECOSYSTEM/SEMVER) or git commits (GIT).
type Range struct {
	Type   string  `json:"type"`
	Repo   string  `json:"repo"`
	Events []Event `json:"events"`
}

// Event is a single boundary in a version range (introduced/fixed/...).
type Event struct {
	Introduced   string `json:"introduced"`
	Fixed        string `json:"fixed"`
	LastAffected string `json:"last_affected"`
	Limit        string `json:"limit"`
}

// ParseRecord decodes a single OSV JSON record.
func ParseRecord(data []byte) (Record, error) {
	var r Record
	if err := json.Unmarshal(data, &r); err != nil {
		return Record{}, fmt.Errorf("osv: invalid record: %w", err)
	}
	if r.ID == "" {
		return Record{}, errors.New("osv: record has no id")
	}
	return r, nil
}

// ParseRecordsFromZip reads an OSV all.zip archive (one JSON record per entry)
// and returns the parsed records. Entries that fail to parse are skipped so a
// single malformed record does not abort an initial bulk import.
func ParseRecordsFromZip(r io.Reader) ([]Record, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("osv: read zip: %w", err)
	}
	zr, err := zip.NewReader(strings.NewReader(string(body)), int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("osv: open zip: %w", err)
	}
	var out []Record
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			continue
		}
		rec, err := ParseRecord(data)
		if err != nil {
			continue
		}
		out = append(out, rec)
	}
	return out, nil
}

// ModifiedEntry is one row of the OSV modified_id.csv index.
type ModifiedEntry struct {
	Modified time.Time
	Path     string // e.g. "PyPI/PYSEC-2021-123"
}

// ReadModifiedSince reads the OSV modified_id.csv stream (reverse-chronological)
// and returns entries strictly newer than since. Reading stops at the first
// entry not newer than since, so the potentially large index is not consumed
// in full.
func ReadModifiedSince(r io.Reader, since time.Time) ([]ModifiedEntry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("osv: read modified csv: %w", err)
	}
	var out []ModifiedEntry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		ts, path, ok := strings.Cut(line, ",")
		if !ok {
			continue
		}
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(ts))
		if err != nil {
			continue
		}
		if !t.After(since) {
			break
		}
		out = append(out, ModifiedEntry{Modified: t, Path: strings.TrimSpace(path)})
	}
	return out, nil
}
