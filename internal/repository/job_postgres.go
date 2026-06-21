package repository

import (
	"context"
	"fmt"
	"strings"
	"JoblessYu/internal/domain"
	"github.com/jackc/pgx/v5"
)

type JobRepository struct {
	dbURL string
}

func NewJobRepository(dbURL string) *JobRepository {
	return &JobRepository{dbURL: dbURL}
}

// FetchRawJobs fetches jobs from DB before markdown conversion and detailed filtering.
func (r *JobRepository) FetchRawJobs(ctx context.Context, level, jobType, location string) ([]domain.JobEntry, error) {
	conn, err := pgx.Connect(ctx, r.dbURL)
	if err != nil {
		return nil, err
	}
	defer conn.Close(ctx)

	args := []interface{}{}
	argIdx := 1
	query := `SELECT COALESCE(title, ''), COALESCE(company, ''), COALESCE(location, ''), job_url, COALESCE(job_type, ''), COALESCE(description, '') FROM jobs`

	var conditions []string
	if level != "" {
		conditions = append(conditions, fmt.Sprintf(`(title ILIKE $%d OR description ILIKE $%d)`, argIdx, argIdx+1))
		args = append(args, "%"+level+"%", "%"+level+"%")
		argIdx += 2
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

	rows, err := conn.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.JobEntry
	for rows.Next() {
		var title, company, loc, url, jobTypeDB, description string
		if err := rows.Scan(&title, &company, &loc, &url, &jobTypeDB, &description); err != nil {
			continue
		}
		
		jobs = append(jobs, domain.JobEntry{
			Title:       title,
			Company:     company,
			Location:    loc,
			URL:         url,
			Description: description,
		})
	}

	return jobs, nil
}
