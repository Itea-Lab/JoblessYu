package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"JoblessYu/internal/domain"
	"JoblessYu/internal/repository"
)

var (
	levelRe = regexp.MustCompile(`(?i)\b(internships?|interns?|juniors?|seniors?)\b`)
	typeRe  = regexp.MustCompile(`(?i)\b(full[ -]?time|part[ -]?time)\b`)
	yearsRe = regexp.MustCompile(`(?i)\b(\d{1,2})\s*\+?\s*(?:years?|yrs?)\b`)
	htmlRe  = regexp.MustCompile(`<[^>]*>`)
)

type JobService struct {
	repo *repository.JobRepository
}

func NewJobService(repo *repository.JobRepository) *JobService {
	return &JobService{repo: repo}
}

func (s *JobService) FetchAndProcessJobs(ctx context.Context, level, jobType, location string) ([]domain.JobEntry, error) {
	rawJobs, err := s.repo.FetchRawJobs(ctx, level, jobType, location)
	if err != nil {
		return nil, err
	}

	var filtered []domain.JobEntry

	for _, j := range rawJobs {
		detectedLevel, detectedType := s.detectJobMeta(j.Title, j.Description)
		if j.Level == "" {
			j.Level = detectedLevel
		}
		// Prefer the authoritative job_type from the database; only fall back to
		// the description scan result when the DB column is empty.
		if j.Type == "" {
			j.Type = detectedType
		}

		if level != "" && j.Level != "" && !strings.EqualFold(j.Level, level) {
			continue
		}
		if jobType != "" && j.Type != "" && !strings.EqualFold(j.Type, jobType) {
			continue
		}

		filtered = append(filtered, j)
	}

	return filtered, nil
}

func (s *JobService) detectJobMeta(title, description string) (level, jobType string) {
	// Strip HTML tags so level/type words cannot match inside attribute values
	// (e.g. <div class="job-senior-card">). Only the visible body text remains.
	plain := htmlRe.ReplaceAllString(title+"\n"+description, " ")
	hay := strings.ToLower(plain)

	// Authoritative: an explicit level word wins, years-of-experience is
	// only consulted when no level word is found anywhere in the text.
	if m := levelRe.FindString(hay); m != "" {
		stem := strings.TrimSuffix(strings.ToLower(m), "s")
		switch stem {
		case "internship", "intern":
			level = "Intern"
		case "junior":
			level = "Junior"
		case "senior":
			level = "Senior"
		}
	}
	if level == "" {
		if m := yearsRe.FindStringSubmatch(hay); m != nil {
			var n int
			if _, err := fmt.Sscanf(m[1], "%d", &n); err == nil {
				switch {
				case n <= 1:
					level = "Fresher"
				case n <= 4:
					level = "Junior"
				default:
					level = "Senior"
				}
			}
		}
	}

	if m := typeRe.FindString(hay); m != "" {
		switch strings.ToLower(m) {
		case "full-time", "full time", "fulltime":
			jobType = "Fulltime"
		case "part-time", "part time", "parttime":
			jobType = "Parttime"
		}
	}

	return
}
