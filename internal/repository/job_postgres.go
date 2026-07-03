package repository

import (
	"context"
	"fmt"
	"log"
	"strings"

	"JoblessYu/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

// levelWordPatterns, levelYearPatterns and levelNoExpPatterns are DERIVED from
// internal/domain.LevelRules — the single source of truth for level
// detection. Do not edit them here; add/modify a LevelRule in domain
// instead. Postgres uses \m...\M (start/end-of-word) boundaries which
// behave equivalently to Go's (?i)\b...\b for the ASCII-only word
// shapes we use, so the same WordPattern/NoExpPattern source embeds safely
// in both engines' regexes.
var (
	levelWordPatterns = func() map[string]string {
		out := make(map[string]string, len(domain.LevelRules))
		for _, r := range domain.LevelRules {
			out[r.Name] = `\m(?:` + r.WordPattern + `)\M`
		}
		return out
	}()
	levelYearPatterns = func() map[string]string {
		out := make(map[string]string, len(domain.LevelRules))
		for _, r := range domain.LevelRules {
			if r.YearPattern != "" {
				out[r.Name] = r.YearPattern
			}
		}
		return out
	}()
	levelNoExpPatterns = func() map[string]string {
		out := make(map[string]string)
		for _, r := range domain.LevelRules {
			if r.NoExpPattern != "" {
				out[r.Name] = `\m(?:` + r.NoExpPattern + `)\M`
			}
		}
		return out
	}()
)

func NewJobRepository(ctx context.Context, dbURL string) (*JobRepository, error) {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	return &JobRepository{pool: pool}, nil
}

func (r *JobRepository) Close() {
	r.pool.Close()
}

// FetchRawJobs fetches jobs from DB before markdown conversion and detailed
// filtering.
//
// When level is set and includeUnknown is false (the default), the SQL
// WHERE clause restricts rows to those already carrying the requested level
// signal: the level word (in title or description), the years-of-experience
// regex (description), or the no-experience-phrase regex (title or
// description). All three predicates are OR'd so the service-layer
// classify-then-strict-filter sees a pre-filtered set that mirrors its own
// detection logic — no DB↔service drift.
//
// When includeUnknown is true AND a level is requested, the level predicate
// is omitted entirely from SQL — fetch is widened to all rows matching the
// other filters, and the service-layer strict filter then classifies each
// row, keeping the requested level OR Unknown rows. This preserves the
// include_unknown contract for rows the DB-side regex can't recognize as
// the requested level but which detectJobMeta would still flag Unknown.
//
// includeUnknown has no effect when level is empty.
func (r *JobRepository) FetchRawJobs(ctx context.Context, level, jobType, location string, includeUnknown bool) ([]domain.JobEntry, error) {
	args := []interface{}{}
	argIdx := 1
	query := `SELECT COALESCE(title, ''), COALESCE(company, ''), COALESCE(location, ''), job_url, COALESCE(job_type, ''), COALESCE(description, '') FROM jobs`

	var conditions []string
	if level != "" && !includeUnknown {
		// Apply the DB-side level predicate only when the caller wants the
		// strict (narrow) path. When includeUnknown is set, drop the
		// predicate so unclassified rows reach service-layer classification.
		wordPat := levelWordPatterns[level]
		yearPat, hasYearPat := levelYearPatterns[level]
		noExpPat, hasNoExpPat := levelNoExpPatterns[level]

		var levelConds []string
		if wordPat != "" {
			levelConds = append(levelConds, fmt.Sprintf(`title ~* $%d`, argIdx))
			args = append(args, wordPat)
			argIdx++
			levelConds = append(levelConds, fmt.Sprintf(`description ~* $%d`, argIdx))
			args = append(args, wordPat)
			argIdx++
		}
		if hasYearPat {
			levelConds = append(levelConds, fmt.Sprintf(`description ~* $%d`, argIdx))
			args = append(args, yearPat)
			argIdx++
		}
		// no-exp phrase is checked against BOTH title and description, mirroring
		// the service layer's detectJobMeta scan which combines title+desc.
		// This keeps DB admissibility in sync with service classification.
		if hasNoExpPat {
			levelConds = append(levelConds, fmt.Sprintf(`title ~* $%d`, argIdx))
			args = append(args, noExpPat)
			argIdx++
			levelConds = append(levelConds, fmt.Sprintf(`description ~* $%d`, argIdx))
			args = append(args, noExpPat)
			argIdx++
		}

		if len(levelConds) > 0 {
			conditions = append(conditions, "("+strings.Join(levelConds, " OR ")+")")
		}
	}
	if jobType != "" {
		conditions = append(conditions, fmt.Sprintf(`(job_type ILIKE $%d OR description ILIKE $%d)`, argIdx, argIdx+1))
		args = append(args, "%"+jobType+"%", "%"+jobType+"%")
		argIdx += 2
	}
	switch location {
	case "HCM":
		conditions = append(conditions, fmt.Sprintf(`(location ILIKE $%d OR location ILIKE $%d)`, argIdx, argIdx+1))
		args = append(args, "%HCM%", "%Ho Chi Minh%")
		argIdx += 2
	case "HN":
		conditions = append(conditions, fmt.Sprintf(`(location ILIKE $%d OR location ILIKE $%d OR location ILIKE $%d)`, argIdx, argIdx+1, argIdx+2))
		args = append(args, "%HN%", "%Ha Noi%", "%Hanoi%")
		argIdx += 3
	}

	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	query += ` ORDER BY fetched_at DESC LIMIT 20`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.JobEntry
	for rows.Next() {
		var title, company, loc, url, jobTypeDB, description string
		if err := rows.Scan(&title, &company, &loc, &url, &jobTypeDB, &description); err != nil {
			log.Printf("job repository: scan error (skipping row): %v", err)
			continue
		}

		jobs = append(jobs, domain.JobEntry{
			Title:       title,
			Company:     company,
			Location:    loc,
			URL:         url,
			Description: description,
			Type:        jobTypeDB,
		})
	}

	return jobs, rows.Err()
}
