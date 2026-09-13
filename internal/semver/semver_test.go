package semver

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in  string
		ok  bool
		maj uint64
		min uint64
		pat uint64
	}{
		{"1.2.3", true, 1, 2, 3},
		{"0.0.1", true, 0, 0, 1},
		{"1.2.3-rc1", true, 1, 2, 3},
		{"1.2", false, 0, 0, 0},
		{"v1.2.3", false, 0, 0, 0},
	}
	for _, c := range cases {
		v, err := Parse(c.in)
		if c.ok && err != nil {
			t.Errorf("Parse(%q): unexpected err %v", c.in, err)
			continue
		}
		if !c.ok && err == nil {
			t.Errorf("Parse(%q): expected error, got %+v", c.in, v)
			continue
		}
		if c.ok && (v.Major != c.maj || v.Minor != c.min || v.Patch != c.pat) {
			t.Errorf("Parse(%q): got %+v", c.in, v)
		}
	}
}

func TestRangeMatches(t *testing.T) {
	cases := []struct {
		range_ string
		ver    string
		want   bool
	}{
		{">=1.0.0 <2.0.0", "1.5.0", true},
		{">=1.0.0 <2.0.0", "2.0.0", false},
		{">=1.0.0 <2.0.0", "0.9.0", false},
		{"^1.2.0", "1.2.5", true},
		{"^1.2.0", "1.9.9", true},
		{"^1.2.0", "2.0.0", false},
		{"~1.2.0", "1.2.9", true},
		{"~1.2.0", "1.3.0", false},
		{">=1.0.0", "1.0.0", true},
		{">=1.0.0", "0.9.0", false},
		{"=1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", false},
	}
	for _, c := range cases {
		r, err := ParseRangeExpr(c.range_)
		if err != nil {
			t.Fatalf("ParseRangeExpr(%q): %v", c.range_, err)
		}
		v, err := Parse(c.ver)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.ver, err)
		}
		if got := r.Matches(v); got != c.want {
			t.Errorf("Range(%q).Matches(%q) = %v, want %v", c.range_, c.ver, got, c.want)
		}
	}
}
