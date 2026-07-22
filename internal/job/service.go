package job

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// regexRetryCooldown is how long to wait before retrying Groq on rows that
// were previously classified by the regex fallback. Prevents burning Groq
// quota on every /jobs query for transiently-failed rows, while still
// giving them a second chance after the original failure may have passed.
const regexRetryCooldown = time.Hour

// jobStore is the contract JobService needs from the persistence layer.
// Defined here (consumer-side) per Go's "accept interfaces" convention;
// repository.JobRepository satisfies it structurally.
type jobStore interface {
	FetchRawJobs(ctx context.Context, q JobQuery) ([]JobEntry, error)
	MarkAIProcessed(ctx context.Context, jobID int64, meta JobMeta) error
}

type JobService struct {
	store     jobStore
	extractor Extractor
}

func NewJobService(store jobStore, extractor Extractor) *JobService {
	return &JobService{store: store, extractor: extractor}
}

// FetchAndProcessJobs fetches raw rows from the store, runs the AI extractor
// on un-enriched rows (lazy enrichment), applies strict level/type filtering,
// and returns the enriched set.
//
// Already-enriched rows (ai_processed_at IS NOT NULL) skip extraction
// entirely — their level/tags come straight from the DB. This keeps
// Groq API calls to a minimum: each job is enriched exactly once.
func (s *JobService) FetchAndProcessJobs(ctx context.Context, q JobQuery) ([]JobEntry, error) {
	rawJobs, err := s.store.FetchRawJobs(ctx, q)
	if err != nil {
		return nil, err
	}

	var filtered []JobEntry

	for _, j := range rawJobs {
		// Decide whether to (re)extract: un-enriched rows always extract;
		// regex-classified rows retry after a cooldown (transient Groq
		// failures may succeed on retry); AI-classified rows skip forever.
		needsExtract := !j.AIProcessed
		if j.AIProcessed && j.AIModel == "regex" && time.Since(j.AIProcessedAt) > regexRetryCooldown {
			needsExtract = true
		}

		if needsExtract {
			// Un-enriched row: extract metadata via AI (or regex fallback).
			// AI value wins for un-enriched rows — the DB has nothing yet.
			meta, extractErr := s.extractor.Extract(ctx, j.Title, j.Description)
			if extractErr != nil {
				slog.Warn("extractor failed, skipping enrichment",
					"job_id", j.ID, "err", extractErr)
				// Continue with whatever fields we have (Level may be empty).
				// The row still gets filtered below — if Level is empty and
				// the user asked for a specific level, it won't match.
			} else {
				j.Level = meta.Level
				j.Type = meta.Type
				j.Tags = meta.Tags
				j.Summary = meta.Summary
				j.Salary = meta.Salary
				j.Remote = meta.Remote
				j.AIModel = meta.Model

				// Persist so future queries skip extraction.
				if err := s.store.MarkAIProcessed(ctx, j.ID, meta); err != nil {
					slog.Warn("failed to persist AI metadata",
						"job_id", j.ID, "err", err)
				}
			}
		}

		// Strict level filtering: when the caller asks for a specific level,
		// any JD that detected to a different level is skipped. Unknown is
		// skipped too unless the caller explicitly opted in via
		// IncludeUnknown.
		if q.Level != "" {
			if strings.EqualFold(j.Level, q.Level) {
				// exact match — keep
			} else if strings.EqualFold(j.Level, LevelUnknown) && q.IncludeUnknown {
				// keep
			} else {
				continue
			}
		}
		if q.JobType != "" && j.Type != "" && !strings.EqualFold(j.Type, q.JobType) {
			continue
		}

		filtered = append(filtered, j)
	}

	return filtered, nil
}
