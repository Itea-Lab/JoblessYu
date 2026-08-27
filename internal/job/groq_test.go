package job

import (
	"testing"
	"time"
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

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain JSON",
			input: `{"level": "Junior", "type": "Fulltime"}`,
			want:  `{"level": "Junior", "type": "Fulltime"}`,
		},
		{
			name:  "markdown fenced",
			input: "```json\n{\"level\": \"Junior\"}\n```",
			want:  `{"level": "Junior"}`,
		},
		{
			name:  "prose before JSON",
			input: `Here is the result: {"level": "Senior", "remote": true}`,
			want:  `{"level": "Senior", "remote": true}`,
		},
		{
			name:  "prose around JSON",
			input: `Sure! {"level": "Fresher"} Hope that helps!`,
			want:  `{"level": "Fresher"}`,
		},
		{
			name:  "nested objects",
			input: `{"tags": {"Cloud": ["AWS", "IAM"]}, "level": "Junior"}`,
			want:  `{"tags": {"Cloud": ["AWS", "IAM"]}, "level": "Junior"}`,
		},
		{
			name:  "no JSON",
			input: `I cannot classify this job.`,
			want:  "",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractJSON(c.input)
			if got != c.want {
				t.Errorf("extractJSON(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		min  time.Duration
		max  time.Duration
	}{
		{
			name: "seconds only",
			msg:  "Rate limit reached. Please try again in 3.84s.",
			min:  4 * time.Second,
			max:  5 * time.Second,
		},
		{
			name: "minutes and seconds",
			msg:  "Rate limit reached for model qwen/qwen3.6-27b on tokens per day (TPD): Limit 200000. Please try again in 26m44.88s.",
			min:  26*time.Minute + 44*time.Second,
			max:  26*time.Minute + 47*time.Second,
		},
		{
			name: "hours minutes seconds",
			msg:  "Please try again in 1h2m3s.",
			min:  1*time.Hour + 2*time.Minute + 3*time.Second,
			max:  1*time.Hour + 2*time.Minute + 6*time.Second,
		},
		{
			name: "fallback unparseable",
			msg:  "Rate limit error without time format",
			min:  5 * time.Second,
			max:  5 * time.Second,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := parseRetryAfter(c.msg)
			if got < c.min || got > c.max {
				t.Errorf("parseRetryAfter(%q) = %v, want between %v and %v", c.msg, got, c.min, c.max)
			}
		})
	}
}

