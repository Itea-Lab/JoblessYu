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

// levelYearPatterns maps a requested level to a case-insensitive PostgreSQL
// regex (~*) that matches a year-of-experience mention corresponding to that
// bucket. Bucket boundaries (per MVP spec):
//   Intern: 0 years | Fresher: 1-2 years | Junior: 3-4 years | Senior: 5+ years
// Pattern uses \m (start-of-word boundary) so "1 years" matches but "21 years"
// does not collapse into the 1-year bucket.
var levelYearPatterns = map[string]string{
	"Intern":  `\m0\s*\+?\s*years?`,
	"Fresher": `\m(1|2)\s*\+?\s*years?`,
	"Junior":  `\m(3|4)\s*\+?\s*years?`,
	"Senior":  `\m([5-9]|\d{2})\s*\+?\s*years?`,
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

// FetchRawJobs fetches jobs from DB before markdown conversion and detailed filtering.
func (r *JobRepository) FetchRawJobs(ctx context.Context, level, jobType, location string) ([]domain.JobEntry, error) {
	args := []interface{}{}
	argIdx := 1
	query := `SELECT COALESCE(title, ''), COALESCE(company, ''), COALESCE(location, ''), job_url, COALESCE(job_type, ''), COALESCE(description, '') FROM jobs`

	var conditions []string
	if level != "" {
		if pat, ok := levelYearPatterns[level]; ok {
			conditions = append(conditions, fmt.Sprintf(`(title ILIKE $%d OR description ILIKE $%d OR description ~* $%d)`, argIdx, argIdx+1, argIdx+2))
			args = append(args, "%"+level+"%", "%"+level+"%", pat)
			argIdx += 3
		} else {
			conditions = append(conditions, fmt.Sprintf(`(title ILIKE $%d OR description ILIKE $%d)`, argIdx, argIdx+1))
			args = append(args, "%"+level+"%", "%"+level+"%")
			argIdx += 2
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
