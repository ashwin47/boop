package projects

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{"Uini": "uini", "My  Project!": "my-project", "--": "project", "FPL XI 2026": "fpl-xi-2026"}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
