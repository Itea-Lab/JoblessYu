package job

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// enricherStore is the contract BatchEnricher needs from the persistence layer.
type enricherStore interface {
	FetchUnenrichedJobs(ctx context.Context) ([]JobEntry, error)
	MarkAIProcessed(ctx context.Context, jobID int64, meta JobMeta) error
	DeleteJob(ctx context.Context, jobID int64) error
}

// BatchEnricher runs AI enrichment on un-enriched jobs. Called by the daily
// cron at 5:01 AM ICT. Has a 2-hour timeout via context deadline.
//
// Retry strategy:
//   - 429 (rate limit): parse retry_after from Groq error, sleep, retry same
//     job. Max 5 attempts. After exhaustion: skip (retry tomorrow).
//   - 400 (bad request): immediate retry once. After retry: skip (JD is
//     the problem, regex won't help).
//   - JSON parse error: skip (model returned unparseable output — JD issue).
//   - 401/403 (auth error): skip immediately (API key invalid).
//   - Network error / API key not configured: retry once, then use regex
//     fallback (Groq is unreachable — this is the only case where regex
//     is used as a last resort).
//   - JD < 50 chars: delete job immediately (useless without description).
type BatchEnricher struct {
	store     enricherStore
	extractor Extractor
	regex     *RegexExtractor // last-resort fallback, only when Groq is unreachable
}

func NewBatchEnricher(store enricherStore, extractor Extractor) *BatchEnricher {
	return &BatchEnricher{
		store:     store,
		extractor: extractor,
		regex:     NewRegexExtractor(),
	}
}

// Run fetches all un-enriched jobs and enriches them with AI.
// The context should have a deadline set (e.g., 2 hours).
func (e *BatchEnricher) Run(ctx context.Context) error {
	jobs, err := e.store.FetchUnenrichedJobs(ctx)
	if err != nil {
		return fmt.Errorf("fetch unenriched jobs: %w", err)
	}

	if len(jobs) == 0 {
		slog.Info("enricher: no un-enriched jobs found")
		return nil
	}

	slog.Info("enricher: starting batch processing", "jobs", len(jobs))
	start := time.Now()
	enriched, skipped, deleted := 0, 0, 0

	// Step 1: Filter out short/invalid jobs
	var validJobs []JobEntry
	for _, j := range jobs {
		if len(strings.TrimSpace(j.Description)) < groqMinJDLength {
			if err := e.store.DeleteJob(ctx, j.ID); err != nil {
				slog.Warn("enricher: failed to delete short-JD job", "job_id", j.ID, "err", err)
			} else {
				deleted++
			}
		} else {
			validJobs = append(validJobs, j)
		}
	}

	// Step 2: Process 1 job per request (with 18s throttle) to strictly prevent Groq 429/413 token limit errors
	for idx, j := range validJobs {
		if ctx.Err() != nil {
			slog.Warn("enricher: context cancelled, stopping enrichment", "processed", idx, "remaining", len(validJobs)-idx)
			break
		}

		meta, err := e.enrichWithRetry(ctx, j)
		if err != nil {
			slog.Warn("enricher: job enrichment failed, skipping", "job_id", j.ID, "title", j.Title, "err", err)
			skipped++
		} else {
			if j.Remote {
				meta.Remote = true
			}
			if err := e.store.MarkAIProcessed(ctx, j.ID, meta); err != nil {
				slog.Warn("enricher: failed to persist AI metadata", "job_id", j.ID, "err", err)
				skipped++
			} else {
				enriched++
				slog.Info("enricher: 1-job enriched successfully", "job_id", j.ID, "title", j.Title, "progress", fmt.Sprintf("%d/%d", idx+1, len(validJobs)))
			}
		}
	}

	elapsed := time.Since(start).Round(time.Second)
	slog.Info("enricher: batch processing complete",
		"enriched", enriched, "skipped", skipped, "deleted", deleted,
		"total", len(jobs), "elapsed", elapsed)

	return nil
}

// enrichWithRetry calls the Groq extractor with retry logic.
// Regex fallback is used ONLY when Groq is unreachable (network errors or
// API key not configured). 429, 400, and JSON parse errors do NOT trigger
// regex — those indicate Groq is still responding but the specific JD or
// model output is problematic.
func (e *BatchEnricher) enrichWithRetry(ctx context.Context, j JobEntry) (JobMeta, error) {
	const maxRetries429 = 5
	var lastErr error
	var shouldFallbackToRegex bool

	for attempt := 0; attempt <= maxRetries429; attempt++ {
		meta, err := e.extractor.Extract(ctx, j.Title, j.Description)
		if err == nil {
			return meta, nil
		}
		lastErr = err

		var apiErr *openai.APIError
		isAPIError := errors.As(err, &apiErr)
		var parseErr *jsonParseError
		isParseError := errors.As(err, &parseErr)

		switch {
		case isAPIError && apiErr.HTTPStatusCode == 429:
			// Rate limited — retry with backoff.
			if attempt < maxRetries429 {
				wait := parseRetryAfter(apiErr.Message)
				slog.Warn("enricher: 429 rate limited, retrying",
					"job_id", j.ID, "attempt", attempt+1, "wait", wait)
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return JobMeta{}, ctx.Err()
				}
				continue
			}
			// All 429 retries exhausted — skip (Groq still responding, just rate-limited).
			return JobMeta{}, fmt.Errorf("429 after %d retries: %w", maxRetries429, err)

		case isAPIError && apiErr.HTTPStatusCode == 400:
			// Bad request — retry once (model may produce different output).
			if attempt == 0 {
				slog.Warn("enricher: 400 bad request, retrying once", "job_id", j.ID)
				continue
			}
			// 400 after retry — the JD is the problem. No regex fallback.
			return JobMeta{}, fmt.Errorf("400 after retry: %w", err)

		case isParseError:
			// JSON parse error after Groq's own retry — the model returned
			// unparseable output. This is a model/JD issue, not a network issue.
			// Skip without regex fallback (same as 400).
			return JobMeta{}, fmt.Errorf("json parse error: %w", err)

		case isAPIError && (apiErr.HTTPStatusCode == 401 || apiErr.HTTPStatusCode == 403):
			// Auth error — API key invalid or revoked. No point retrying.
			return JobMeta{}, fmt.Errorf("groq auth error (%d): %w", apiErr.HTTPStatusCode, err)

		case errors.Is(err, ErrDisabledAPIKey):
			// Groq API key is explicitly omitted in config — fall back to regex directly without retrying.
			shouldFallbackToRegex = true

		default:
			// True network error or timeout.
			// Groq is unreachable — retry once, then fall back to regex.
			shouldFallbackToRegex = true
			if attempt == 0 {
				slog.Warn("enricher: network error, retrying once",
					"job_id", j.ID, "err", err)
				select {
				case <-time.After(5 * time.Second):
				case <-ctx.Done():
					return JobMeta{}, ctx.Err()
				}
				continue
			}
		}

		break
	}

	// Regex fallback: ONLY when Groq is unreachable (network errors or
	// API key not configured). Does NOT trigger for 429, 400, or JSON parse
	// errors — those indicate Groq is responding but the JD/model is the problem.
	if shouldFallbackToRegex {
		slog.Warn("enricher: Groq unreachable, using regex fallback",
			"job_id", j.ID, "title", j.Title, "err", lastErr)
		meta, err := e.regex.Extract(ctx, j.Title, j.Description)
		if err == nil {
			return meta, nil
		}
		return JobMeta{}, fmt.Errorf("regex fallback also failed: %w (original: %v)", err, lastErr)
	}

	return JobMeta{}, fmt.Errorf("enrichment failed: %w", lastErr)
}

// retryAfterRe matches "Please try again in 3.84s" from Groq error messages.
var retryAfterRe = regexp.MustCompile(`try again in (\d+\.?\d*)\s*s`)

// parseRetryAfter extracts the retry-after duration from a Groq 429 error
// message. Falls back to 5 seconds if parsing fails.
func parseRetryAfter(msg string) time.Duration {
	matches := retryAfterRe.FindStringSubmatch(msg)
	if len(matches) < 2 {
		return 5 * time.Second
	}
	secs, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 5 * time.Second
	}
	return time.Duration(secs*1000)*time.Millisecond + 1*time.Second
}
