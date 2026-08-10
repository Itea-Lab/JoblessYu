package job

import (
	"context"
	"strings"
	"time"
)

// jobStore is the contract JobService needs from the persistence layer.
// Defined here (consumer-side) per Go's "accept interfaces" convention;
// JobRepository satisfies it structurally.
type jobStore interface {
	FetchRawJobs(ctx context.Context, q JobQuery) ([]JobEntry, error)
}

type JobService struct {
	store jobStore
}

func NewJobService(store jobStore) *JobService {
	return &JobService{store: store}
}

// FetchAndProcessJobs fetches AI-enriched jobs from the database and applies
// strict level/type filtering. This is a pure DB read — no Groq calls.
// AI enrichment happens in the daily BatchEnricher, not on-demand.
func (s *JobService) FetchAndProcessJobs(ctx context.Context, q JobQuery) ([]JobEntry, error) {
	rawJobs, err := s.store.FetchRawJobs(ctx, q)
	if err != nil {
		return nil, err
	}

	var filtered = make([]JobEntry, 0, len(rawJobs))
	for _, j := range rawJobs {
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
		if q.JobType != "" && !strings.EqualFold(j.Type, q.JobType) {
			continue
		}
		if q.Expertise != "" && !strings.EqualFold(j.Expertise, q.Expertise) {
			continue
		}

		filtered = append(filtered, j)
	}

	return filtered, nil
}

// GetScrapeStats delegates to the underlying store to retrieve active job count and last scrape timestamp.
func (s *JobService) GetScrapeStats(ctx context.Context) (int, time.Time, error) {
	if statsStore, ok := s.store.(interface {
		GetScrapeStats(ctx context.Context) (int, time.Time, error)
	}); ok {
		return statsStore.GetScrapeStats(ctx)
	}
	return 0, time.Time{}, nil
}

// GetPipelineSummaryStats delegates to underlying store to retrieve recent 24-hour pipeline metrics for static Card 2 initialization.
func (s *JobService) GetPipelineSummaryStats(ctx context.Context) (PipelineStats, error) {
	if statsStore, ok := s.store.(interface {
		GetPipelineSummaryStats(ctx context.Context) (PipelineStats, error)
	}); ok {
		return statsStore.GetPipelineSummaryStats(ctx)
	}
	return PipelineStats{}, nil
}

// ListenForJobChanges delegates to underlying store for real-time Postgres LISTEN jobs_changed notifications.
func (s *JobService) ListenForJobChanges(ctx context.Context, onChange func()) {
	if listenStore, ok := s.store.(interface {
		ListenForJobChanges(ctx context.Context, onChange func())
	}); ok {
		listenStore.ListenForJobChanges(ctx, onChange)
	}
}
