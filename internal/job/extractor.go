package job

import "context"

// Extractor is the contract for turning a raw job title + description into
// structured metadata (level, type, expertise, tags, salary, remote, summary).
//
// Implementations:
//   - GroqExtractor (groq.go) — calls Groq API, may fail
//   - RegexExtractor (regex.go) — always succeeds, no external deps
//
// The BatchEnricher calls GroqExtractor as the primary extractor. When Groq
// is completely unavailable (all retries exhausted on 429/network errors),
// it falls back to RegexExtractor so jobs remain visible during AI outages.
type Extractor interface {
	Extract(ctx context.Context, title, description string) (JobMeta, error)
}

type JobBatchItem struct {
	ID          int64
	Title       string
	Description string
}

type BatchExtractor interface {
	Extractor
	ExtractBatch(ctx context.Context, batch []JobBatchItem) ([]JobMeta, error)
}
