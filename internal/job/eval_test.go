package job

import (
	"context"
	"testing"
)

func TestRunAIEvalWithRegex(t *testing.T) {
	regex := NewRegexExtractor()
	ctx := context.Background()

	report := RunAIEval(ctx, regex)

	if report.TotalCases == 0 {
		t.Fatalf("Expected benchmark cases, got 0")
	}

	if report.LevelMatches == 0 {
		t.Errorf("Expected level matches > 0, got 0")
	}

	if report.LocationMatches == 0 {
		t.Errorf("Expected location matches > 0, got 0")
	}

	if report.HallucinationCount > 0 {
		t.Errorf("Expected 0 hallucinations for regex extractor, got %d", report.HallucinationCount)
	}
}
