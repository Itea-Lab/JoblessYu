package job

import (
	"context"
	"log/slog"

)

// ChainExtractor tries the primary extractor first, and on any error
// falls back to the secondary. This is the "hybrid AI + regex" pattern:
// GroqExtractor is primary (accurate, but can fail/rate-limit),
// RegexExtractor is the fallback (always succeeds, less accurate).
//
// The returned JobMeta.Model field identifies which extractor produced
// the result, so the service layer can persist it for audit.
type ChainExtractor struct {
	primary  Extractor
	fallback Extractor
}

// NewChainExtractor wires a primary + fallback extraction chain.
func NewChainExtractor(primary, fallback Extractor) *ChainExtractor {
	return &ChainExtractor{primary: primary, fallback: fallback}
}

// Close stops the primary extractor's rate-limiter if it has one.
// Safe to call multiple times. Extractor interface doesn't include Close();
// callers check via type assertion.
func (c *ChainExtractor) Close() {
	if closer, ok := c.primary.(interface{ Close() }); ok {
		closer.Close()
	}
}

func (c *ChainExtractor) Extract(ctx context.Context, title, description string) (JobMeta, error) {
	meta, err := c.primary.Extract(ctx, title, description)
	if err != nil {
		slog.Warn("primary extractor failed, falling back to regex",
			"primary_model", c.primaryModelName(),
			"err", err,
		)
		return c.fallback.Extract(ctx, title, description)
	}
	return meta, nil
}

// primaryModelName returns a human-readable name for the primary extractor
// for logging. Uses a type switch since Extractor doesn't expose Name().
func (c *ChainExtractor) primaryModelName() string {
	switch c.primary.(type) {
	case *GroqExtractor:
		return "groq"
	case *RegexExtractor:
		return "regex"
	default:
		return "unknown"
	}
}
