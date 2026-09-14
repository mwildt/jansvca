package osv

import (
	"testing"
)

func TestMap_BasicRecord(t *testing.T) {
	rec := Record{
		ID:       "GHSA-1",
		Summary:  "RCE in foo",
		Details:  "details",
		Severity: []Severity{{Type: "CVSS_V3", Score: "9.8"}},
		Modified: "2024-05-01T00:00:00Z",
		Affected: []Affected{{
			Package: &Package{PURL: "pkg:npm/foo", Name: "foo", Ecosystem: "npm"},
			Ranges: []Range{{
				Type: "ECOSYSTEM",
				Events: []Event{
					{Introduced: "0"},
					{Fixed: "2.0.0"},
				},
			}},
		}},
	}
	imp := Map(rec)
	if imp.ID != "GHSA-1" || imp.Identifier != "GHSA-1" {
		t.Errorf("id/identifier: %+v", imp)
	}
	if imp.Title != "RCE in foo" || imp.Description != "details" {
		t.Errorf("title/desc: %+v", imp)
	}
	if imp.CVSS != 9.8 {
		t.Errorf("cvss: %v", imp.CVSS)
	}
	if len(imp.Affected) != 1 {
		t.Fatalf("affected: %+v", imp.Affected)
	}
	a := imp.Affected[0]
	if a.Component != "pkg:npm/foo" {
		t.Errorf("component: %q", a.Component)
	}
	if a.VersionRange != "<2.0.0" {
		t.Errorf("range: %q", a.VersionRange)
	}
}

func TestMap_IntroducedAndLastAffected(t *testing.T) {
	rec := Record{
		ID:      "GHSA-2",
		Summary: "x",
		Affected: []Affected{{
			Package: &Package{PURL: "pkg:npm/bar"},
			Ranges: []Range{{
				Type: "ECOSYSTEM",
				Events: []Event{
					{Introduced: "1.0.0"},
					{LastAffected: "1.5.0"},
				},
			}},
		}},
	}
	imp := Map(rec)
	if len(imp.Affected) != 1 {
		t.Fatalf("affected: %+v", imp.Affected)
	}
	if got, want := imp.Affected[0].VersionRange, ">=1.0.0 <=1.5.0"; got != want {
		t.Errorf("range: got %q want %q", got, want)
	}
}

func TestMap_SkipsGitRangesAndMissingPackage(t *testing.T) {
	rec := Record{
		ID: "GHSA-3",
		Affected: []Affected{
			{
				Ranges: []Range{{Type: "GIT", Events: []Event{{Introduced: "0"}, {Fixed: "deadbee"}}}},
			},
			{
				Package: &Package{PURL: "pkg:npm/ok"},
				Ranges: []Range{{
					Type:   "GIT",
					Events: []Event{{Introduced: "0"}, {Fixed: "v1.2.3"}},
				}},
			},
		},
	}
	imp := Map(rec)
	if len(imp.Affected) != 0 {
		t.Fatalf("expected no affected ranges, got %+v", imp.Affected)
	}
}

func TestMap_StripsVPrefix(t *testing.T) {
	rec := Record{
		ID: "GHSA-4",
		Affected: []Affected{{
			Package: &Package{PURL: "pkg:npm/baz"},
			Ranges: []Range{{
				Type: "ECOSYSTEM",
				Events: []Event{
					{Introduced: "v1.0.0"},
					{Fixed: "v2.0.0"},
				},
			}},
		}},
	}
	imp := Map(rec)
	if got, want := imp.Affected[0].VersionRange, ">=1.0.0 <2.0.0"; got != want {
		t.Errorf("range: got %q want %q", got, want)
	}
}

func TestMap_CVSSVectorIgnoredWhenNumericOnly(t *testing.T) {
	rec := Record{
		ID:       "GHSA-5",
		Summary:  "x",
		Severity: []Severity{{Type: "CVSS_V3", Score: "CVSS:3.1/AV:N/AC:L"}},
	}
	imp := Map(rec)
	if imp.CVSS != 0 {
		t.Errorf("vector string should yield 0, got %v", imp.CVSS)
	}
}

func TestMap_UsesNameWhenNoPURL(t *testing.T) {
	rec := Record{
		ID: "GHSA-6",
		Affected: []Affected{{
			Package: &Package{Name: "lodash", Ecosystem: "npm"},
			Ranges: []Range{{
				Type:   "SEMVER",
				Events: []Event{{Introduced: "0"}, {Fixed: "4.17.21"}},
			}},
		}},
	}
	imp := Map(rec)
	if len(imp.Affected) != 1 || imp.Affected[0].Component != "lodash" {
		t.Fatalf("expected lodash component, got %+v", imp.Affected)
	}
	if imp.Affected[0].VersionRange != "<4.17.21" {
		t.Errorf("range: %q", imp.Affected[0].VersionRange)
	}
}
