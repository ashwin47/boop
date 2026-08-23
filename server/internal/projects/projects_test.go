package projects

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{"Uini": "uini", "My  Project!": "my-project", "--": "project", "FPL XI 2026": "fpl-xi-2026"}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIcons(t *testing.T) {
	for _, ok := range []string{"", "circle:mint", "blob:slate", "🚀", "ab"} {
		if !ValidIcon(ok) {
			t.Errorf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"star:mint", "circle:red", "circle:", ":mint", "toolongtext"} {
		if ValidIcon(bad) {
			t.Errorf("%q should be invalid", bad)
		}
	}
	a, b := DefaultIcon("uini"), DefaultIcon("uini")
	if a != b || !ValidIcon(a) || !strings.Contains(a, ":") {
		t.Errorf("DefaultIcon not deterministic/valid: %q %q", a, b)
	}
	if DefaultIcon("uini") == DefaultIcon("infrastructure") && DefaultIcon("uini") == DefaultIcon("tonight") {
		t.Errorf("DefaultIcon should vary across seeds")
	}
}
