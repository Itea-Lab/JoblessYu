package job

import (
	"context"

)

// Extractor is the contract for turning a raw job title + description into
// structured metadata (level, type, tags, salary, remote, summary).
//
// Implementations:
//   - RegexExtractor (fallback.go) — always succeeds, no external deps
//   - GroqExtractor (groq.go, Slice D) — calls Groq API, may fail
//
// The service layer calls Extract on every fetched job. When the primary
// extractor (Groq, Slice D) fails, it falls back to RegexExtractor so the
// bot always returns results even during an AI outage.
type Extractor interface {
	Extract(ctx context.Context, title, description string) (JobMeta, error)
}
