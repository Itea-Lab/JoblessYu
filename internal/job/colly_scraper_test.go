package job

import (
	"context"
	"errors"
	"testing"
)

func TestScrapeITViecHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewCollyScraper().ScrapeITViec(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ScrapeITViec() error = %v, want context.Canceled", err)
	}
}

func TestParsePostedAge(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"Posted\n5 hours ago", 5},
		{"Posted\n23 hours ago", 23},
		{"Posted\n30 minutes ago", 0},
		{"Posted\n1 day ago", 24},
		{"Posted\n2 days ago", 48},
		{"Posted\n3 days ago", 72},
		{"", -1},
		{"Vietnam best IT Companies", -1},
		{"Posted just now", -1},
	}
	for _, c := range cases {
		got := parsePostedAge(c.input)
		if got != c.want {
			t.Errorf("parsePostedAge(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}
