package job

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
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

var nonAlphaNumRe = regexp.MustCompile(`[^\w]+`)

func ComputeDedupHash(company, title, jobType string) string {
	normComp := strings.TrimSpace(nonAlphaNumRe.ReplaceAllString(strings.ToLower(company), ""))
	normTitle := strings.TrimSpace(nonAlphaNumRe.ReplaceAllString(strings.ToLower(title), ""))
	normType := strings.TrimSpace(nonAlphaNumRe.ReplaceAllString(strings.ToLower(jobType), ""))
	raw := normComp + ":" + normTitle + ":" + normType
	hash := md5.Sum([]byte(raw))
	return hex.EncodeToString(hash[:])
}

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
	query := `SELECT id, COALESCE(title, ''), COALESCE(company, ''), COALESCE(location, ''), job_url, COALESCE(NULLIF(job_type_normalized, ''), COALESCE(job_type, '')), COALESCE(description, ''), COALESCE(level, ''), ai_processed_at, COALESCE(tags, '{}'::jsonb), COALESCE(summary, ''), COALESCE(salary, ''), COALESCE(remote, false), COALESCE(ai_model, ''), COALESCE(expertise, ''), COALESCE(site, ''), COALESCE(alternate_urls, '[]'::jsonb) FROM jobs`

	// Only show AI-enriched jobs — un-enriched jobs are hidden from users
	// until the daily batch enrichment processes them.
	conditions := []string{"ai_processed_at IS NOT NULL"}
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
	if q.Expertise != "" {
		conditions = append(conditions, fmt.Sprintf(`expertise = $%d`, argIdx))
		args = append(args, q.Expertise)
		argIdx++
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
		var title, company, loc, url, jobTypeDB, description, level, summary, salary, aiModel, expertise, site string
		var remote bool
		var aiProcessedAt sql.NullTime
		var tagsJSON, altURLsJSON []byte
		if err := rows.Scan(&id, &title, &company, &loc, &url, &jobTypeDB, &description, &level, &aiProcessedAt, &tagsJSON, &summary, &salary, &remote, &aiModel, &expertise, &site, &altURLsJSON); err != nil {
			log.Printf("job repository: scan error (skipping row): %v", err)
			continue
		}

		var tags map[string][]string
		if len(tagsJSON) > 0 {
			if err := json.Unmarshal(tagsJSON, &tags); err != nil {
				log.Printf("job repository: tags parse error (job %d): %v", id, err)
			}
		}

		var altURLs []JobSource
		if len(altURLsJSON) > 0 {
			if err := json.Unmarshal(altURLsJSON, &altURLs); err != nil {
				log.Printf("job repository: altURLs parse error (job %d): %v", id, err)
			}
		}

		company = strings.TrimSpace(company)
		if idx := strings.Index(company, "\n"); idx >= 0 {
			company = strings.TrimSpace(company[:idx])
		}
		if len(company) > 100 {
			company = company[:100]
		}
		loc = strings.TrimSpace(loc)
		if idx := strings.Index(loc, "\n"); idx >= 0 {
			loc = strings.TrimSpace(loc[:idx])
		}
		if len(loc) > 80 {
			loc = loc[:80]
		}

		entry := JobEntry{
			ID:            id,
			Title:         title,
			Company:       company,
			Location:      loc,
			URL:           url,
			Site:          site,
			Description:   description,
			Type:          jobTypeDB,
			Level:         level,
			Expertise:     expertise,
			AIProcessed:   aiProcessedAt.Valid,
			Summary:       summary,
			Salary:        salary,
			Remote:        remote,
			AIModel:       aiModel,
			Tags:          tags,
			AlternateURLs: altURLs,
			DedupHash:     ComputeDedupHash(company, title, jobTypeDB),
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
 		     ai_model            = $7,
 		     expertise           = $8
 		 WHERE id = $9`,
		meta.Level, meta.Type, tagsJSON, meta.Summary, meta.Salary, meta.Remote, meta.Model, meta.Expertise, jobID,
	)
	return err
}

// FetchUnenrichedJobs returns all jobs where ai_processed_at IS NULL,
// ordered by fetched_at DESC. Used by the BatchEnricher to find jobs
// that need AI classification.
func (r *JobRepository) FetchUnenrichedJobs(ctx context.Context) ([]JobEntry, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, COALESCE(title, ''), COALESCE(company, ''), COALESCE(location, ''),
		        job_url, COALESCE(NULLIF(job_type_normalized, ''), COALESCE(job_type, '')),
		        COALESCE(description, ''), COALESCE(remote, false)
		 FROM jobs
		 WHERE ai_processed_at IS NULL
		 ORDER BY fetched_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []JobEntry
	for rows.Next() {
		var j JobEntry
		if err := rows.Scan(&j.ID, &j.Title, &j.Company, &j.Location, &j.URL, &j.Type, &j.Description, &j.Remote); err != nil {
			log.Printf("FetchUnenrichedJobs: scan error (skipping row): %v", err)
			continue
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// DeleteJob removes a single job by ID. Used by the enricher to delete
// jobs with empty/useless descriptions.
func (r *JobRepository) DeleteJob(ctx context.Context, jobID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, jobID)
	return err
}

// DeleteOldJobs removes jobs older than the retention period. Returns
// the number of rows deleted. Called by the weekly cleanup cron.
func (r *JobRepository) DeleteOldJobs(ctx context.Context, retentionDays int) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM jobs WHERE fetched_at < NOW() - $1 * interval '1 day'`,
		retentionDays)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// UpsertJobs bulk-inserts scraped jobs, updating existing rows on conflict.
// If the job description changes (employer updated the JD), all AI enrichment
// fields are reset to NULL — forcing re-enrichment in the next batch.
func (r *JobRepository) UpsertJobs(ctx context.Context, jobs []JobEntry) (int, error) {
	if len(jobs) == 0 {
		return 0, nil
	}

	inserted := 0
	for _, j := range jobs {
		comp := strings.TrimSpace(j.Company)
		if idx := strings.Index(comp, "\n"); idx >= 0 {
			comp = strings.TrimSpace(comp[:idx])
		}
		if len(comp) > 100 {
			comp = comp[:100]
		}
		loc := strings.TrimSpace(j.Location)
		if idx := strings.Index(loc, "\n"); idx >= 0 {
			loc = strings.TrimSpace(loc[:idx])
		}
		if len(loc) > 80 {
			loc = loc[:80]
		}

		dedupHash := ComputeDedupHash(comp, j.Title, j.Type)

		// Check for cross-site duplicate within 7 days
		var existingID int64
		var existingSite, existingURL string
		var existingAltJSON []byte
		err := r.pool.QueryRow(ctx,
			`SELECT id, site, job_url, COALESCE(alternate_urls, '[]'::jsonb)
			 FROM jobs
			 WHERE dedup_hash = $1 AND fetched_at >= NOW() - INTERVAL '7 days'
			 LIMIT 1`,
			dedupHash,
		).Scan(&existingID, &existingSite, &existingURL, &existingAltJSON)

		if err == nil && existingID > 0 {
			var altSources []JobSource
			_ = json.Unmarshal(existingAltJSON, &altSources)

			alreadyPresent := (j.URL == existingURL)
			for _, src := range altSources {
				if src.URL == j.URL {
					alreadyPresent = true
					break
				}
			}

			if !alreadyPresent {
				altSources = append(altSources, JobSource{Site: j.Site, URL: j.URL})
				newAltJSON, _ := json.Marshal(altSources)
				_, _ = r.pool.Exec(ctx,
					`UPDATE jobs SET alternate_urls = $1, fetched_at = NOW() WHERE id = $2`,
					newAltJSON, existingID,
				)
			}
			inserted++
			continue
		}

		_, err = r.pool.Exec(ctx,
			`INSERT INTO jobs (job_id, site, job_url, title, company, location, job_type, description, remote, dedup_hash)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 ON CONFLICT (site, job_url) DO UPDATE SET
			   title       = EXCLUDED.title,
			   company     = EXCLUDED.company,
			   location    = EXCLUDED.location,
			   job_type    = EXCLUDED.job_type,
			   description = EXCLUDED.description,
			   fetched_at  = NOW(),
			   dedup_hash  = EXCLUDED.dedup_hash,
			   remote      = jobs.remote OR EXCLUDED.remote,
			   ai_processed_at     = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
			                              THEN NULL ELSE jobs.ai_processed_at END,
			   level               = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
			                              THEN NULL ELSE jobs.level END,
			   job_type_normalized = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
			                              THEN NULL ELSE jobs.job_type_normalized END,
			   tags                = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
			                              THEN '{}'::jsonb ELSE jobs.tags END,
			   summary             = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
			                              THEN NULL ELSE jobs.summary END,
			   salary              = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
			                              THEN NULL ELSE jobs.salary END,
			   ai_model            = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
			                              THEN NULL ELSE jobs.ai_model END,
			   expertise           = CASE WHEN jobs.description IS DISTINCT FROM EXCLUDED.description
			                              THEN NULL ELSE jobs.expertise END`,
			j.URL, j.Site, j.URL, j.Title, comp, loc, j.Type, j.Description, j.Remote, dedupHash,
		)
		if err != nil {
			log.Printf("UpsertJobs: error on %s (skipping): %v", j.URL, err)
			continue
		}
		inserted++
	}
	return inserted, nil
}
