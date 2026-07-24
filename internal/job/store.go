package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

// levelWordPatterns, levelYearPatterns and levelNoExpPatterns are DERIVED from
// internal/LevelRules — the single source of truth for level
// detection. Do not edit them here; add/modify a LevelRule in domain
// instead. Postgres uses \m...\M (start/end-of-word) boundaries which
// behave equivalently to Go's (?i)\b...\b for the ASCII-only word
// shapes we use, so the same WordPattern/NoExpPattern source embeds safely
// in both engines' regexes.
var (
	levelWordPatterns = func() map[string]string {
		out := make(map[string]string, len(LevelRules))
		for _, r := range LevelRules {
			out[r.Name] = `\m(?:` + r.WordPattern + `)\M`
		}
		return out
	}()
	levelYearPatterns = func() map[string]string {
		out := make(map[string]string, len(LevelRules))
		for _, r := range LevelRules {
			if r.YearPattern != "" {
				out[r.Name] = r.YearPattern
			}
		}
		return out
	}()
	levelNoExpPatterns = func() map[string]string {
		out := make(map[string]string)
		for _, r := range LevelRules {
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
// When q.Level is set and q.IncludeUnknown is false (the default), the SQL
// WHERE clause restricts rows to those already carrying the requested level
// signal: the level word (in title or description), the years-of-experience
// regex (description), or the no-experience-phrase regex (title or
// description). All three predicates are OR'd so the service-layer
// classify-then-strict-filter sees a pre-filtered set that mirrors its own
// detection logic — no DB↔service drift.
//
// When q.IncludeUnknown is true AND a level is requested, the level predicate
// is omitted entirely from SQL — fetch is widened to all rows matching the
// other filters, and the service-layer strict filter then classifies each
// row, keeping the requested level OR Unknown rows. This preserves the
// include_unknown contract for rows the DB-side regex can't recognize as
// the requested level but which detectJobMeta would still flag Unknown.
//
// q.IncludeUnknown has no effect when q.Level is empty.
// q.Limit defaults to 20 when zero or negative.
func (r *JobRepository) FetchRawJobs(ctx context.Context, q JobQuery) ([]JobEntry, error) {
	args := []interface{}{}
	argIdx := 1
	query := `SELECT id, COALESCE(title, ''), COALESCE(company, ''), COALESCE(location, ''), job_url, COALESCE(NULLIF(job_type_normalized, ''), COALESCE(job_type, '')), COALESCE(description, ''), COALESCE(level, ''), ai_processed_at, COALESCE(tags, '{}'::jsonb), COALESCE(summary, ''), COALESCE(salary, ''), COALESCE(remote, false), COALESCE(ai_model, '') FROM jobs`

	var conditions []string
	if q.Level != "" && !q.IncludeUnknown && !q.AIEnabled {
		// Apply the DB-side level predicate only when the caller wants the
		// strict (narrow) path AND AI is not enabled. When includeUnknown is
		// set, drop the predicate so unclassified rows reach service-layer
		// classification. When AI is enabled, also drop the predicate — fetch
		// wider so the AI extractor can classify rows the regex would miss.
		wordPat := levelWordPatterns[q.Level]
		yearPat, hasYearPat := levelYearPatterns[q.Level]
		noExpPat, hasNoExpPat := levelNoExpPatterns[q.Level]

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
	if q.JobType != "" {
		conditions = append(conditions, fmt.Sprintf(`(job_type ILIKE $%d OR description ILIKE $%d)`, argIdx, argIdx+1))
		args = append(args, "%"+q.JobType+"%", "%"+q.JobType+"%")
		argIdx += 2
	}
	switch q.Location {
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

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	query += fmt.Sprintf(` ORDER BY fetched_at DESC LIMIT %d`, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobEntry
	for rows.Next() {
		var id int64
		var title, company, loc, url, jobTypeDB, description, level, summary, salary, aiModel string
		var remote bool
		var aiProcessedAt sql.NullTime
		var tagsJSON []byte
		if err := rows.Scan(&id, &title, &company, &loc, &url, &jobTypeDB, &description, &level, &aiProcessedAt, &tagsJSON, &summary, &salary, &remote, &aiModel); err != nil {
			log.Printf("job repository: scan error (skipping row): %v", err)
			continue
		}

		var tags map[string][]string
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &tags); err != nil {
				log.Printf("job repository: tags parse error (job %d): %v", id, err)
			}
		}

		entry := JobEntry{
			ID:          id,
			Title:       title,
			Company:     company,
			Location:    loc,
			URL:         url,
			Description: description,
			Type:        jobTypeDB,
			Level:       level,
			AIProcessed: aiProcessedAt.Valid,
			Summary:     summary,
			Salary:      salary,
			Remote:      remote,
			AIModel:     aiModel,
			Tags:        tags,
		}
		if aiProcessedAt.Valid {
			entry.AIProcessedAt = aiProcessedAt.Time
		}
		jobs = append(jobs, entry)
	}

	return jobs, rows.Err()
}

// MarkAIProcessed writes AI extraction results back to the jobs row so
// subsequent /jobs queries skip re-extraction (lazy enrichment). The
// meta.Model field identifies which extractor produced the result (e.g.
// "regex", "llama-3.1-8b-instant") and is stored in ai_model for audit.
func (r *JobRepository) MarkAIProcessed(ctx context.Context, jobID int64, meta JobMeta) error {
	tags := meta.Tags
	if tags == nil {
		tags = map[string][]string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE jobs
		 SET level               = $1,
		     job_type_normalized = $2,
		     tags                = $3,
		     summary             = $4,
		     salary              = $5,
		     remote              = $6,
		     ai_processed_at     = NOW(),
		     ai_model            = $7
		 WHERE id = $8`,
		meta.Level, meta.Type, tagsJSON, meta.Summary, meta.Salary, meta.Remote, meta.Model, jobID,
	)
	return err
}
