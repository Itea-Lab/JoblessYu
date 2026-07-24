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

