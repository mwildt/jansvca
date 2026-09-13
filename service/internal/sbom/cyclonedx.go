// Package sbom parses Software Bill of Materials documents into a flat list
// of component declarations (component identifier + version) that the projekte
// module can import.
//
// CycloneDX (https://cyclonedx.org) SBOMs are supported in JSON form. The
// parser is intentionally permissive: it reads the spec-version field only for
// reporting and tolerates missing fields by skipping the affected component.
package sbom

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mwildt/jansvca/service/internal/projekte/domain"
)

// ErrUnsupportedFormat is returned when the document is neither a CycloneDX JSON
// document nor otherwise recognisable.
var ErrUnsupportedFormat = errors.New("sbom: unsupported format (only CycloneDX JSON is supported)")

type cycloneDXComponent struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	Version    string `json:"version"`
	PURL       string `json:"purl"`
	BOMRef     string `json:"bom-ref"`
	PackageURL string `json:"packageURL"`
}

type cycloneDXDocument struct {
	BomFormat   string              `json:"bomFormat"`
	SpecVersion string              `json:"specVersion"`
	Components  []cycloneDXComponent `json:"components"`
}

// Parse parses a CycloneDX JSON document and returns the declared components.
// The purl takes precedence as the identifier, falling back to name@version and
// finally to the bom-ref. Components without an identifier or version are
// skipped.
func Parse(data []byte) ([]domain.ImportInput, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, errors.New("sbom: empty document")
	}
	if trimmed[0] != '{' {
		return nil, ErrUnsupportedFormat
	}

	var doc cycloneDXDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("sbom: invalid JSON: %w", err)
	}
	if doc.BomFormat != "" && !strings.EqualFold(doc.BomFormat, "CycloneDX") {
		return nil, fmt.Errorf("sbom: unexpected bomFormat %q (expected CycloneDX)", doc.BomFormat)
	}

	var out []domain.ImportInput
	for _, c := range doc.Components {
		identifier := c.PURL
		if identifier == "" {
			identifier = c.PackageURL
		}
		if identifier == "" && c.Name != "" {
			identifier = c.Name
			if c.Version != "" {
				identifier = c.Name + "@" + c.Version
			}
		}
		if identifier == "" {
			identifier = c.BOMRef
		}
		if identifier == "" || c.Version == "" {
			continue
		}
		out = append(out, domain.ImportInput{Component: identifier, Version: c.Version})
	}
	return out, nil
}
