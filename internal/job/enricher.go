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
			// Rate limited — check if it's a daily limit or long wait.
			wait := parseRetryAfter(apiErr.Message)
			isDailyLimit := strings.Contains(strings.ToLower(apiErr.Message), "tokens per day") ||
				strings.Contains(strings.ToLower(apiErr.Message), "requests per day") ||
				strings.Contains(strings.ToLower(apiErr.Message), "tpd") ||
				strings.Contains(strings.ToLower(apiErr.Message), "rpd")

			if isDailyLimit || wait > 60*time.Second {
				slog.Warn("enricher: Groq rate limit / daily quota exceeded, falling back to regex",
					"job_id", j.ID, "wait", wait, "msg", apiErr.Message)
				shouldFallbackToRegex = true
				break
			}

			if attempt < maxRetries429 {
				slog.Warn("enricher: 429 rate limited, retrying",
					"job_id", j.ID, "attempt", attempt+1, "wait", wait)
				select {
				case <-time.After(wait):
				case <-ctx.Done():
					return JobMeta{}, ctx.Err()
				}
				continue
			}
			// All 429 retries exhausted — fall back to regex instead of failing the job.
			slog.Warn("enricher: 429 retries exhausted, falling back to regex", "job_id", j.ID)
			shouldFallbackToRegex = true

		case isAPIError && apiErr.HTTPStatusCode == 400:
			// Bad request — retry once (model may produce different output).
			if attempt == 0 {
				slog.Warn("enricher: 400 bad request, retrying once", "job_id", j.ID)
				continue
			}
			// 400 after retry — the JD is the problem. Fall back to regex.
			shouldFallbackToRegex = true

		case isParseError:
			// JSON parse error after Groq's own retry — fall back to regex.
			shouldFallbackToRegex = true

		case isAPIError && (apiErr.HTTPStatusCode == 401 || apiErr.HTTPStatusCode == 403):
			// Auth error — API key invalid or revoked. Fall back to regex.
			slog.Warn("enricher: Groq auth error, falling back to regex", "status", apiErr.HTTPStatusCode)
			shouldFallbackToRegex = true

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

	// Regex fallback: used when Groq is unreachable, quota is exceeded, or parsing fails.
	if shouldFallbackToRegex {
		slog.Warn("enricher: using regex fallback",
			"job_id", j.ID, "title", j.Title, "err", lastErr)
		meta, err := e.regex.Extract(ctx, j.Title, j.Description)
		if err == nil {
			return meta, nil
		}
		return JobMeta{}, fmt.Errorf("regex fallback also failed: %w (original: %v)", err, lastErr)
	}

	return JobMeta{}, fmt.Errorf("enrichment failed: %w", lastErr)
}

// retryAfterRe matches "Please try again in 3.84s", "26m44.88s", or "1h2m3s" from Groq error messages.
var retryAfterRe = regexp.MustCompile(`try again in (?:(\d+)h)?(?:(\d+)m)?(\d+(?:\.\d+)?)s`)

// parseRetryAfter extracts the retry-after duration from a Groq 429 error
// message. Falls back to 5 seconds if parsing fails.
func parseRetryAfter(msg string) time.Duration {
	matches := retryAfterRe.FindStringSubmatch(msg)
	if len(matches) < 4 {
		return 5 * time.Second
	}
	var total time.Duration
	if matches[1] != "" {
		if h, err := strconv.Atoi(matches[1]); err == nil {
			total += time.Duration(h) * time.Hour
		}
	}
	if matches[2] != "" {
		if m, err := strconv.Atoi(matches[2]); err == nil {
			total += time.Duration(m) * time.Minute
		}
	}
	if matches[3] != "" {
		if s, err := strconv.ParseFloat(matches[3], 64); err == nil {
			total += time.Duration(s * float64(time.Second))
		}
	}
	if total == 0 {
		return 5 * time.Second
	}
	return total + 1*time.Second
}
