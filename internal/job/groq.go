package job

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

//go:embed skills.md
var skillsPrompt string

// Groq free tier limits (as of 2026-07):
//   30 RPM      — requests per minute
//   8,000 TPM   — tokens per minute
//   1,000 RPD   — requests per day
//   200,000 TPD — tokens per day
//
// Actual tokens per call (measured, not estimated):
//   skills.md system prompt: ~706 tokens
//   schema.go description:  ~249 tokens
//   typical JD user message: ~300 tokens
//   JSON output:             ~200 tokens
//   TOTAL:                   ~1,455 tokens/call
//
// Without prompt caching: 8,000 / 1,455 ≈ 5 calls/min max.
// 12s throttle = 5 calls/min × 1,455 = 7,275 tokens/min (under 8K TPM).
// This is safe WITHOUT relying on Groq's prompt caching (which is automatic
// but not guaranteed — cache may be evicted on long gaps between /jobs queries).
//
// First /jobs on 20 un-enriched jobs: ~4 min (12s × 20). Discord interaction
// timeout is 15 min — safe margin.
const (
	groqBaseURL    = "https://api.groq.com/openai/v1"
	groqThrottle   = 12 * time.Second
	groqMaxRetries = 1 // retry once on JSON parse failure, then fall back
	groqMaxTokens  = 800
)

// jsonParseError wraps a JSON unmarshal failure. Extract retries only on
// these errors (the model may produce different output on retry). API and
// network errors fall back immediately — retrying won't help.
type jsonParseError struct{ err error }

func (e *jsonParseError) Error() string { return e.err.Error() }
func (e *jsonParseError) Unwrap() error { return e.err }

// GroqExtractor calls Groq's OpenAI-compatible API to classify job
// descriptions. It embeds skills.md as the system prompt and requests
// JSON object mode (qwen/qwen3.6-27b does not support strict JSON schema).
//
// On JSON parse failure, it retries once. On API/network failure, it
// returns immediately so ChainExtractor can fall back to RegexExtractor.
type GroqExtractor struct {
	client *openai.Client
	model  string
	ticker *time.Ticker // rate-limiter; stopped via Close()
}

// NewGroqExtractor creates a Groq-backed Extractor. apiKey is the Groq
// API key; model is the Groq model ID (e.g. "qwen/qwen3.6-27b").
func NewGroqExtractor(apiKey, model string) *GroqExtractor {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = groqBaseURL
	return &GroqExtractor{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
		ticker: time.NewTicker(groqThrottle),
	}
}

// Close stops the rate-limiter ticker. Safe to call multiple times.
// Extractor interface doesn't include Close() — callers check via type
// assertion: `if closer, ok := ext.(interface{ Close() }); ok { closer.Close() }`
func (g *GroqExtractor) Close() {
	if g.ticker != nil {
		g.ticker.Stop()
	}
}

func (g *GroqExtractor) Extract(ctx context.Context, title, description string) (JobMeta, error) {
	// Rate-limit: block until the next throttle tick. This naturally
	// spaces calls at 12s intervals, keeping us under 5 calls/min.
	select {
	case <-g.ticker.C:
	case <-ctx.Done():
		return JobMeta{}, ctx.Err()
	}

	systemMsg := skillsPrompt + "\n\nReturn JSON with this shape:\n" + jobMetaSchemaDescription
	userMsg := fmt.Sprintf("Title: %s\n\nDescription:\n%s", title, description)

	var lastErr error
	for attempt := 0; attempt <= groqMaxRetries; attempt++ {
		meta, err := g.callGroq(ctx, systemMsg, userMsg)
		if err == nil {
			return meta, nil
		}

		// Only retry on JSON parse errors — the model may produce
		// different output on retry. API/network errors fall back
		// immediately since retrying won't help.
		var parseErr *jsonParseError
		if !errors.As(err, &parseErr) {
			return JobMeta{}, err
		}
		lastErr = err
		slog.Warn("Groq JSON parse error, retrying", "attempt", attempt+1, "err", err)
	}

	return JobMeta{}, fmt.Errorf("groq extraction failed after %d attempts: %w", groqMaxRetries+1, lastErr)
}

func (g *GroqExtractor) callGroq(ctx context.Context, systemMsg, userMsg string) (JobMeta, error) {
	resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: g.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemMsg},
			{Role: openai.ChatMessageRoleUser, Content: userMsg},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
		Temperature: 0.0,
		MaxTokens:   groqMaxTokens,
	})
	if err != nil {
		// API/network/rate-limit error — return as plain error so
		// Extract() doesn't retry, ChainExtractor falls back immediately.
		return JobMeta{}, fmt.Errorf("groq API call: %w", err)
	}

	if len(resp.Choices) == 0 {
		return JobMeta{}, fmt.Errorf("groq returned no choices")
	}

	content := resp.Choices[0].Message.Content
	content = strings.TrimSpace(content)
	if content == "" {
		return JobMeta{}, fmt.Errorf("groq returned empty content")
	}

	var meta JobMeta
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		// JSON parse error — wrap as jsonParseError so Extract() retries.
		return JobMeta{}, &jsonParseError{err: fmt.Errorf("parse groq JSON: %w (content: %s)", err, truncate(content, 200))}
	}

	// Validate required fields. Groq JSON object mode guarantees valid
	// JSON but not schema adherence — a retry may fix a malformed response.
	if meta.Level == "" {
		meta.Level = LevelUnknown
	}
	if meta.Tags == nil {
		meta.Tags = map[string][]string{}
	}
	meta.Model = g.model

	return meta, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
