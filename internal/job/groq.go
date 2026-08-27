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

// ErrDisabledAPIKey is returned when GROQ_API_KEY is omitted or empty.
var ErrDisabledAPIKey = errors.New("groq: API key not configured")

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
	// throttle + 401 API call per job.
	if g.apiKey == "" {
		return JobMeta{}, ErrDisabledAPIKey
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

type rawJobMeta struct {
	Level     string              `json:"level"`
	Type      string              `json:"type"`
	Expertise string              `json:"expertise"`
	Tags      map[string][]string `json:"tags"`
	Salary    string              `json:"salary"`
	Remote    interface{}         `json:"remote"`
	Summary   string              `json:"summary"`
}

func (raw *rawJobMeta) toJobMeta(title, description string) JobMeta {
	var meta JobMeta
	meta.Level = raw.Level
	meta.Type = raw.Type
	meta.Expertise = raw.Expertise
	meta.Tags = raw.Tags
	meta.Salary = raw.Salary
	meta.Summary = raw.Summary
	switch v := raw.Remote.(type) {
	case bool:
		meta.Remote = v
	case string:
		meta.Remote = strings.EqualFold(v, "true")
	default:
		meta.Remote = false
	}

	if meta.Level == "" {
		meta.Level = LevelUnknown
	}
	if meta.Tags == nil {
		meta.Tags = map[string][]string{}
	}
	for cat, tags := range meta.Tags {
		filtered := tags[:0]
		for _, t := range tags {
			if !strings.EqualFold(t, "none") && !strings.EqualFold(t, "n/a") && strings.TrimSpace(t) != "" {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			delete(meta.Tags, cat)
		} else {
			meta.Tags[cat] = filtered
		}
	}
	switch strings.ToLower(strings.TrimSpace(meta.Type)) {
	case "fulltime", "full-time", "full time":
		meta.Type = "Full-time"
	case "parttime", "part-time", "part time":
		meta.Type = "Part-time"
	case "unknown", "":
		meta.Type = "Unknown"
	default:
		regexType := NewRegexExtractor().detectType(strings.ToLower(title + "\n" + description))
		if regexType != "" {
			meta.Type = regexType
		} else {
			meta.Type = "Unknown"
		}
	}

	if meta.Expertise == "" || !IsValidExpertise(meta.Expertise) {
		meta.Expertise = DetectExpertise(title, description)
	}

	return meta
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

	jsonStr := extractJSON(content)
	if jsonStr == "" {
		return JobMeta{}, &jsonParseError{err: fmt.Errorf("no JSON object found in response (content: %s)", truncate(content, 200))}
	}

	jsonStr = cleanJSON(jsonStr)

	var raw rawJobMeta
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return JobMeta{}, &jsonParseError{err: fmt.Errorf("parse groq JSON: %w (content: %s)", err, truncate(jsonStr, 200))}
	}

	meta := raw.toJobMeta(title, description)
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

// cleanJSON removes common AI-introduced JSON syntax errors before parsing:
//   - Line comments (// ...) — the AI sometimes adds JS-style comments
//   - Trailing commas before } or ] — common in AI-generated JSON
//
// This eliminates ~80% of parse errors that would otherwise trigger retries.
func cleanJSON(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' && inString {
			b.WriteByte(c)
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			b.WriteByte(c)
			continue
		}

		// Outside strings: strip // line comments
		if !inString && c == '/' && i+1 < len(s) && s[i+1] == '/' {
			// Skip to end of line
			for i < len(s) && s[i] != '\n' {
				i++
			}
			continue
		}

		// Outside strings: remove trailing comma before } or ]
		if !inString && c == ',' {
			// Look ahead for the next non-whitespace character
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue // skip the comma
			}
		}

		b.WriteByte(c)
	}

	return b.String()
}

// ExtractBatch classifies a slice of up to 10 jobs in a single Groq API call.
func (g *GroqExtractor) ExtractBatch(ctx context.Context, batch []JobBatchItem) ([]JobMeta, error) {
	if len(batch) == 0 {
		return nil, nil
	}
	if len(batch) == 1 {
		meta, err := g.Extract(ctx, batch[0].Title, batch[0].Description)
		if err != nil {
			return nil, err
		}
		return []JobMeta{meta}, nil
	}

	if g.apiKey == "" {
		return nil, fmt.Errorf("groq: API key not configured")
	}

	// Rate-limit: enforce 18s spacing between calls
	if g.firstCall.Load() {
		g.firstCall.Store(false)
	} else {
		select {
		case <-g.ticker.C:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	var sb strings.Builder
	sb.WriteString("Classify the following IT job descriptions. Return a JSON array of objects, one per job, preserving order:\n\n")

	for i, item := range batch {
		desc := item.Description
		if len(desc) > groqMaxJDChars {
			desc = desc[:groqMaxJDChars]
		}
		sb.WriteString(fmt.Sprintf("--- Job %d (ID: %d) ---\nTitle: %s\nDescription:\n%s\n\n", i+1, item.ID, item.Title, desc))
	}

	systemMsg := skillsPrompt + "\n\nReturn a JSON array of objects where each element corresponds to a job in the batch order. Format:\n[{" + jobMetaSchemaDescription + "}, ...]"
	userMsg := sb.String() + "\nReturn ONLY the JSON array. No prose, no markdown fences."

	var lastErr error
	for attempt := 0; attempt <= groqMaxRetries; attempt++ {
		metas, err := g.callGroqBatch(ctx, systemMsg, userMsg, batch)
		if err == nil {
			return metas, nil
		}
		var parseErr *jsonParseError
		if !errors.As(err, &parseErr) {
			return nil, err
		}
		lastErr = err
		slog.Warn("Groq batch JSON parse error, retrying", "attempt", attempt+1, "err", err)
	}

	return nil, fmt.Errorf("groq batch extraction failed after %d attempts: %w", groqMaxRetries+1, lastErr)
}

func (g *GroqExtractor) callGroqBatch(ctx context.Context, systemMsg, userMsg string, batch []JobBatchItem) ([]JobMeta, error) {
	resp, err := g.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: g.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemMsg},
			{Role: openai.ChatMessageRoleUser, Content: userMsg},
		},
		Temperature: 0.0,
		MaxTokens:   groqMaxTokens * 2,
	})
	if err != nil {
		return nil, fmt.Errorf("groq API batch call: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("groq returned no choices for batch")
	}

	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return nil, fmt.Errorf("groq returned empty content for batch")
	}

	jsonStr := extractJSONArray(content)
	if jsonStr == "" {
		return nil, &jsonParseError{err: fmt.Errorf("no JSON array found in batch response")}
	}

	jsonStr = cleanJSON(jsonStr)

	var rawMetas []rawJobMeta
	if err := json.Unmarshal([]byte(jsonStr), &rawMetas); err != nil {
		return nil, &jsonParseError{err: fmt.Errorf("unmarshal batch json: %w", err)}
	}

	if len(rawMetas) != len(batch) {
		return nil, &jsonParseError{err: fmt.Errorf("batch count mismatch: got %d, expected %d", len(rawMetas), len(batch))}
	}

	metas := make([]JobMeta, len(batch))
	for i, raw := range rawMetas {
		meta := raw.toJobMeta(batch[i].Title, batch[i].Description)
		meta.Model = g.model
		metas[i] = meta
	}

	return metas, nil
}

func extractJSONArray(s string) string {
	start := strings.Index(s, "[")
	if start == -1 {
		return ""
	}
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if !inString {
			switch c {
			case '[':
				depth++
			case ']':
				depth--
				if depth == 0 {
					return s[start : i+1]
				}
			}
		}
	}
	return ""
}
