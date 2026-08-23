package levels

import "testing"

func TestAtLeast(t *testing.T) {
	cases := []struct {
		l, min string
		want   bool
	}{
		{Info, Info, true}, {Info, Warning, false}, {Error, Warning, true}, {Critical, Error, true}, {"bogus", Info, true}, {"bogus", Success, false},
	}
	for _, c := range cases {
		if got := AtLeast(c.l, c.min); got != c.want {
			t.Errorf("AtLeast(%s, %s) = %v, want %v", c.l, c.min, got, c.want)
		}
	}
	if Valid("fatal") || !Valid(Critical) {
		t.Error("Valid is wrong")
	}
}
