package job

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sashabaranov/go-openai"
)

//go:embed skills.md
var skillsPrompt string

// Groq free tier limits (as of 2026-07):
//
//	30 RPM      — requests per minute
//	8,000 TPM   — tokens per minute
//	1,000 RPD   — requests per day
//	200,000 TPD — tokens per day
//
// Actual tokens per call (measured against real JDs):
//
//	skills.md system prompt: ~706 tokens
//	schema.go description:  ~249 tokens
//	JD user message (truncated to 1,500 chars): ~375 tokens
//	JSON output:             ~200 tokens
//	TOTAL:                   ~1,530 tokens/call
//
// 18s throttle = 3.3 calls/min × 1,530 = ~5,050 tokens/min (63% of 8K TPM).
// This leaves comfortable headroom for occasional long JDs and avoids
// relying on Groq's prompt caching (which is automatic but not guaranteed).
//
// JDs shorter than 50 chars are skipped entirely (regex handles them) —
// ~20% of scraped jobs have empty descriptions, each wasting ~1,100 tokens.
//
// First /jobs on 20 un-enriched jobs: ~6 min (18s × 20). Discord interaction
// timeout is 15 min — safe margin.
const (
	groqBaseURL     = "https://api.groq.com/openai/v1"
	groqThrottle    = 18 * time.Second
	groqMaxRetries  = 1 // retry once on JSON parse failure, then fall back
	groqMaxTokens   = 800
	groqMaxJDChars  = 1500 // truncate JDs to this many chars before sending
	groqMinJDLength = 50   // skip Groq for JDs shorter than this
)

// jsonParseError wraps a JSON unmarshal failure. Extract retries only on
// these errors (the model may produce different output on retry). API and
// network errors fall back immediately — retrying won't help.
type jsonParseError struct{ err error }

func (e *jsonParseError) Error() string { return e.err.Error() }
func (e *jsonParseError) Unwrap() error { return e.err }

// GroqExtractor calls Groq's OpenAI-compatible API to classify job
// descriptions. It embeds skills.md as the system prompt and parses
// JSON from the model's free-text response (no response_format constraint,
// which avoids 400 "Failed to generate JSON" errors on certain models).
//
// JDs shorter than 50 chars are rejected immediately so regex handles them.
// JDs are truncated to 1,500 chars to stay within Groq's free-tier TPM limit.
//
// On JSON parse failure, it retries once. On API/network failure, it
// returns immediately so the BatchEnricher can fall back to RegexExtractor.
type GroqExtractor struct {
	client    *openai.Client
	model     string
	ticker    *time.Ticker // rate-limiter; stopped via Close()
	firstCall atomic.Bool  // skip throttle on the first call
	apiKey    string       // empty = not configured, short-circuits Extract
}

// NewGroqExtractor creates a Groq-backed Extractor. apiKey is the Groq
// API key; model is the Groq model ID (e.g. "llama-3.1-8b-instant").
func NewGroqExtractor(apiKey, model string) *GroqExtractor {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = groqBaseURL
	g := &GroqExtractor{
		client: openai.NewClientWithConfig(cfg),
		model:  model,
		ticker: time.NewTicker(groqThrottle),
		apiKey: apiKey,
	}
	g.firstCall.Store(true)
	return g
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
	// Short-circuit when API key is not configured — avoids wasting 18s
	// throttle + 401 API call per job. The enricher falls back to regex.
	if g.apiKey == "" {
		return JobMeta{}, fmt.Errorf("groq: API key not configured")
	}

	// Skip short/empty JDs — regex handles them. ~20% of scraped jobs have
	// empty descriptions; sending them to Groq wastes ~1,100 tokens each.
	if len(strings.TrimSpace(description)) < groqMinJDLength {
		return JobMeta{}, fmt.Errorf("groq: JD too short (%d chars), skipping", len(description))
	}

	// Truncate JD to limit token usage. Most job-relevant info (title,
	// required skills, experience level) appears in the first paragraph.
	if len(description) > groqMaxJDChars {
		description = description[:groqMaxJDChars]
	}

	// Rate-limit: skip the wait on the first call, then enforce 18s spacing
	// for subsequent calls.
	if g.firstCall.Load() {
		g.firstCall.Store(false)
	} else {
		select {
		case <-g.ticker.C:
		case <-ctx.Done():
			return JobMeta{}, ctx.Err()
		}
	}

	// Keep the original description for DetectExpertise (keyword fallback
	// has no token budget and can safely scan the full JD).
	originalDescription := description
	systemMsg := skillsPrompt + "\n\nReturn JSON with this shape:\n" + jobMetaSchemaDescription
	userMsg := fmt.Sprintf("Title: %s\n\nDescription:\n%s\n\nReturn ONLY a JSON object. No prose, no markdown fences.", title, description)

	var lastErr error
	for attempt := 0; attempt <= groqMaxRetries; attempt++ {
		meta, err := g.callGroq(ctx, systemMsg, userMsg, title, originalDescription)
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

func (g *GroqExtractor) callGroq(ctx context.Context, systemMsg, userMsg, title, description string) (JobMeta, error) {
	resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: g.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemMsg},
			{Role: openai.ChatMessageRoleUser, Content: userMsg},
		},
		Temperature: 0.0,
		MaxTokens:   groqMaxTokens,
	})
	if err != nil {
		// API/network/rate-limit error — return as plain error so
		// Extract() doesn't retry. The BatchEnricher handles fallback.
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

	// Extract JSON from the response. The model may wrap it in markdown
	// fences (```json ... ```) or add prose around it. Find the first {...}
	// block and parse that. This is more robust than response_format:
	// json_object mode, which causes 400 errors on certain models/prompts.
	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return JobMeta{}, &jsonParseError{err: fmt.Errorf("no JSON object found in response (content: %s)", truncate(content, 200))}
	}

	var meta JobMeta
	if err := json.Unmarshal([]byte(jsonStr), &meta); err != nil {
		// JSON parse error — wrap as jsonParseError so Extract() retries.
		return JobMeta{}, &jsonParseError{err: fmt.Errorf("parse groq JSON: %w (content: %s)", err, truncate(jsonStr, 200))}
	}

	// Validate required fields. Without response_format constraints, the
	// model may omit fields — defaults keep the structure valid.
	if meta.Level == "" {
		meta.Level = LevelUnknown
	}
	if meta.Tags == nil {
		meta.Tags = map[string][]string{}
	}
	// Validate expertise: AI may return non-canonical values (e.g. "security"
	// instead of "support_security"). Fall back to keyword detection.
	if meta.Expertise == "" || !IsValidExpertise(meta.Expertise) {
		meta.Expertise = DetectExpertise(title, description)
	}
	meta.Model = g.model

	return meta, nil
}

// extractJSON finds the first {...} block in the response content.
// Handles markdown fences, leading/trailing prose, and nested braces.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
