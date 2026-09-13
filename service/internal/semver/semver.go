// Package semver provides Semantic Versioning parsing and version-range
// matching used by the vulnerability matching feature.
package semver

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed semantic version (major.minor.patch) with an optional
// pre-release suffix.
type Version struct {
	Major      uint64
	Minor      uint64
	Patch      uint64
	Prerelease string
}

// Parse parses a semantic version string of the form
// "MAJOR.MINOR.PATCH[-prerelease]".
func Parse(s string) (Version, error) {
	v := Version{}
	core := s
	if i := strings.IndexByte(s, '+'); i >= 0 {
		core = s[:i]
	}
	pre := ""
	if i := strings.IndexByte(core, '-'); i >= 0 {
		pre = core[i+1:]
		core = core[:i]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("semver: expected major.minor.patch, got %q", s)
	}
	for i, p := range parts {
		n, err := strconv.ParseUint(p, 10, 64)
		if err != nil {
			return Version{}, fmt.Errorf("semver: invalid numeric segment %q in %q", p, s)
		}
		switch i {
		case 0:
			v.Major = n
		case 1:
			v.Minor = n
		case 2:
			v.Patch = n
		}
	}
	v.Prerelease = pre
	return v, nil
}

// Compare returns -1, 0, or +1 depending on whether v is less than, equal to,
// or greater than o. Pre-release versions are considered lower than their
// release counterpart (following semver precedence rules in simplified form).
func (v Version) Compare(o Version) int {
	if v.Major != o.Major {
		return cmp(v.Major, o.Major)
	}
	if v.Minor != o.Minor {
		return cmp(v.Minor, o.Minor)
	}
	if v.Patch != o.Patch {
		return cmp(v.Patch, o.Patch)
	}
	if v.Prerelease == o.Prerelease {
		return 0
	}
	if v.Prerelease == "" {
		return 1
	}
	if o.Prerelease == "" {
		return -1
	}
	return strings.Compare(v.Prerelease, o.Prerelease)
}

func cmp[T uint64](a, b T) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// Constraint is a parsed version range constraint.
type Constraint struct {
	op  comparisonOp
	ver Version
}

type comparisonOp int

const (
	opEq comparisonOp = iota
	opNe
	opLt
	opLe
	opGt
	opGe
	opCaret
	opTilde
)

// ErrInvalidRange is returned for an unparseable version range.
var ErrInvalidRange = errors.New("semver: invalid version range")

// ParseRange parses a single constraint expression. Supported forms:
//
//	"1.2.3"           exact match
//	"=1.2.3"          exact match
//	">1.0.0"          greater than
//	">=1.0.0"         greater than or equal
//	"<2.0.0"          less than
//	"<=2.0.0"         less than or equal
//	"^1.2.0"          compatible with 1.2.0 (>=1.2.0 <2.0.0)
//	"~1.2.0"          patch-level compatible (>=1.2.0 <1.3.0)
func ParseRange(expr string) (Constraint, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return Constraint{}, ErrInvalidRange
	}
	c := Constraint{}
	op := opEq
	rest := expr
	switch {
	case strings.HasPrefix(expr, ">="):
		op = opGe
		rest = expr[2:]
	case strings.HasPrefix(expr, "<="):
		op = opLe
		rest = expr[2:]
	case strings.HasPrefix(expr, ">"):
		op = opGt
		rest = expr[1:]
	case strings.HasPrefix(expr, "<"):
		op = opLt
		rest = expr[1:]
	case strings.HasPrefix(expr, "="):
		op = opEq
		rest = expr[1:]
	case strings.HasPrefix(expr, "^"):
		op = opCaret
		rest = expr[1:]
	case strings.HasPrefix(expr, "~"):
		op = opTilde
		rest = expr[1:]
	}
	rest = strings.TrimSpace(rest)
	ver, err := Parse(rest)
	if err != nil {
		return Constraint{}, err
	}
	c.op = op
	c.ver = ver
	return c, nil
}

// Matches reports whether v satisfies the constraint.
func (c Constraint) Matches(v Version) bool {
	switch c.op {
	case opEq:
		return v.Compare(c.ver) == 0
	case opNe:
		return v.Compare(c.ver) != 0
	case opLt:
		return v.Compare(c.ver) < 0
	case opLe:
		return v.Compare(c.ver) <= 0
	case opGt:
		return v.Compare(c.ver) > 0
	case opGe:
		return v.Compare(c.ver) >= 0
	case opCaret:
		if v.Major != c.ver.Major {
			return false
		}
		return v.Compare(c.ver) >= 0
	case opTilde:
		if v.Major != c.ver.Major || v.Minor != c.ver.Minor {
			return false
		}
		return v.Compare(c.ver) >= 0
	}
	return false
}

// Range is a conjunction of constraints (space/comma separated). A version
// satisfies the range if it satisfies all constraints.
type Range struct {
	constraints []Constraint
}

// ParseRangeExpr parses a range expression consisting of one or more
// constraints separated by spaces or commas.
func ParseRangeExpr(expr string) (Range, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return Range{}, ErrInvalidRange
	}
	expr = strings.ReplaceAll(expr, ",", " ")
	parts := strings.Fields(expr)
	r := Range{}
	for _, p := range parts {
		c, err := ParseRange(p)
		if err != nil {
			return Range{}, err
		}
		r.constraints = append(r.constraints, c)
	}
	return r, nil
}

// Matches reports whether v satisfies all constraints of the range.
func (r Range) Matches(v Version) bool {
	for _, c := range r.constraints {
		if !c.Matches(v) {
			return false
		}
	}
	return true
}
