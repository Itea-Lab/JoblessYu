package job

import (
	"context"
	"errors"
	"testing"

)

// fakeExtractor is a test Extractor that returns a preset result or error.
type fakeExtractor struct {
	meta JobMeta
	err  error
}

func (f *fakeExtractor) Extract(_ context.Context, _, _ string) (JobMeta, error) {
	return f.meta, f.err
}

func TestChainExtractor_PrimarySucceeds(t *testing.T) {
	primary := &fakeExtractor{meta: JobMeta{Level: "Senior", Model: "groq"}}
	fallback := &fakeExtractor{meta: JobMeta{Level: "Junior", Model: "regex"}}

	chain := NewChainExtractor(primary, fallback)
	meta, err := chain.Extract(context.Background(), "title", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Level != "Senior" {
		t.Errorf("Level = %q, want Senior (primary should win)", meta.Level)
	}
	if meta.Model != "groq" {
		t.Errorf("Model = %q, want groq", meta.Model)
	}
}

func TestChainExtractor_PrimaryFails_FallsBack(t *testing.T) {
	primary := &fakeExtractor{err: errors.New("groq rate limited")}
	fallback := &fakeExtractor{meta: JobMeta{Level: "Junior", Model: "regex"}}

	chain := NewChainExtractor(primary, fallback)
	meta, err := chain.Extract(context.Background(), "title", "desc")
	if err != nil {
		t.Fatalf("unexpected error: fallback should succeed, got: %v", err)
	}
	if meta.Level != "Junior" {
		t.Errorf("Level = %q, want Junior (fallback should be used)", meta.Level)
	}
	if meta.Model != "regex" {
		t.Errorf("Model = %q, want regex", meta.Model)
	}
}

func TestChainExtractor_BothFail_ReturnsError(t *testing.T) {
	primary := &fakeExtractor{err: errors.New("groq down")}
	fallback := &fakeExtractor{err: errors.New("regex broken")}

	chain := NewChainExtractor(primary, fallback)
	_, err := chain.Extract(context.Background(), "title", "desc")
	if err == nil {
		t.Fatal("expected error when both extractors fail")
	}
}

func TestChainExtractor_PrimaryModelName(t *testing.T) {
	// Verify primaryModelName() returns correct string for logging.
	chain := &ChainExtractor{
		primary:  &GroqExtractor{},
		fallback: &RegexExtractor{},
	}
	if got := chain.primaryModelName(); got != "groq" {
		t.Errorf("primaryModelName() = %q, want groq", got)
	}

	chain2 := &ChainExtractor{
		primary:  &RegexExtractor{},
		fallback: &GroqExtractor{},
	}
	if got := chain2.primaryModelName(); got != "regex" {
		t.Errorf("primaryModelName() = %q, want regex", got)
	}
}
