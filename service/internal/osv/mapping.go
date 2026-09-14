package osv

import (
	"strconv"
	"strings"
	"time"

	"github.com/mwildt/jansvca/service/internal/eventstore"
	"github.com/mwildt/jansvca/service/internal/schwachstellen/domain"
)

// ImportRecord is the internal representation of an OSV record, normalized for
// the schwachstellen module: an id, mutable metadata and the set of affected
// (component, version-range) pairs.
type ImportRecord struct {
	ID          string
	Identifier  string
	Title       string
	Description string
	CVSS        float64
	Withdrawn   bool
	Source      string
	Modified    time.Time
	Affected    []AffectedRange
}

// AffectedRange is one (component, version-range) pair derived from an OSV
// affected entry.
type AffectedRange struct {
	Component    string
	VersionRange string
}

// Map converts an OSV Record into the internal ImportRecord. OSV records
// commonly expose multiple affected packages and multiple ranges per package;
// each ECOSYSTEM/SEMVER range with an introduced/fixed pair becomes one
// AffectedRange. GIT ranges and entries without a package are skipped.
func Map(r Record) ImportRecord {
	out := ImportRecord{
		ID:          r.ID,
		Identifier:  r.ID,
		Title:       r.Summary,
		Description: r.Details,
		CVSS:        mapCVSS(r.Severity),
		Withdrawn:   r.Withdrawn != "",
		Source:      "osv",
		Modified:    parseTime(r.Modified),
	}
	for _, a := range r.Affected {
		if a.Package == nil {
			continue
		}
		component := a.Package.PURL
		if component == "" {
			component = a.Package.Name
		}
		if component == "" {
			continue
		}
		for _, rg := range a.Ranges {
			if rg.Type != "ECOSYSTEM" && rg.Type != "SEMVER" {
				continue
			}
			if vr := rangeFromEvents(rg.Events); vr != "" {
				out.Affected = append(out.Affected, AffectedRange{
					Component:    component,
					VersionRange: vr,
				})
			}
		}
	}
	return out
}

// ToCreate produces the domain events to create a vulnerability with its
// affected ranges in a single stream. It returns the create event followed by
// one AffectedRangeAdded per range, mirroring the schwachstellen event model.
func (i ImportRecord) ToCreate() ([]eventstore.PayloadEvent, error) {
	events := make([]eventstore.PayloadEvent, 0, 1+len(i.Affected))
	created, err := domain.CreateVulnerability(i.ID, i.Identifier, i.Title, i.Description, i.CVSS, i.Source)
	if err != nil {
		return nil, err
	}
	events = append(events, created...)
	for _, a := range i.Affected {
		events = append(events, domain.AffectedRangeAdded{
			VulnerabilityID: i.ID,
			Component:       a.Component,
			VersionRange:    a.VersionRange,
		})
	}
	return events, nil
}

// rangeFromEvents translates an OSV event sequence into a semver range
// expression understood by the internal semver package. It supports the
// common introduced/fixed and introduced/last_affected patterns and a bare
// introduced (open-ended) range.
func rangeFromEvents(events []Event) string {
	var introduced, fixed, lastAffected string
	for _, e := range events {
		if e.Introduced != "" {
			introduced = e.Introduced
		}
		if e.Fixed != "" {
			fixed = e.Fixed
		}
		if e.LastAffected != "" {
			lastAffected = e.LastAffected
		}
	}
	var parts []string
	if introduced != "" && introduced != "0" {
		parts = append(parts, ">="+normalizeVersion(introduced))
	}
	if fixed != "" {
		parts = append(parts, "<"+normalizeVersion(fixed))
	} else if lastAffected != "" {
		parts = append(parts, "<="+normalizeVersion(lastAffected))
	}
	return strings.Join(parts, " ")
}

// mapCVSS picks the highest CVSS v3.x score from the severity entries. OSV
// encodes CVSS scores as a vector string (e.g. "CVSS:3.1/AV:N/...") or a bare
// numeric string; only the base score is extracted when a CVSS string is
// present, otherwise the numeric value is parsed directly.
func mapCVSS(severity []Severity) float64 {
	var best float64
	for _, s := range severity {
		if !strings.HasPrefix(strings.ToUpper(s.Type), "CVSS") {
			continue
		}
		score := parseCVSSScore(s.Score)
		if score > best {
			best = score
		}
	}
	return best
}

// parseCVSSScore extracts the base numeric score from a CVSS vector string or
// a bare numeric score. CVSS vector strings take the form
// "CVSS:3.1/AV:N/AC:..." with no leading number; in that case 0 is returned
// since the base score is not present. When osv.dev provides a bare number it
// is parsed directly.
func parseCVSSScore(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return 0
}

// normalizeVersion strips a leading "v" prefix so versions like "v1.2.3"
// satisfy the internal semver parser (major.minor.patch).
func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	return v
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
