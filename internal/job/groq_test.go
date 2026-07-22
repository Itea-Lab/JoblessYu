package job

import (
	"testing"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		input string
		n     int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly20chars!!!", 20, "exactly20chars!!!"},
		{"this is a very long string that exceeds the limit", 10, "this is a ..."},
		{"", 10, ""},
	}
	for _, c := range cases {
		t.Run(c.input[:min(len(c.input), 10)], func(t *testing.T) {
			got := truncate(c.input, c.n)
			if got != c.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", c.input, c.n, got, c.want)
			}
		})
	}
}

