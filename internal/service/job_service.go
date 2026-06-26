package service

import (
	"context"
	"log"
	"strings"

	"JoblessYu/internal/domain"
	"JoblessYu/internal/repository"

	"github.com/JohannesKaufmann/html-to-markdown/v2"
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
		// Convert HTML to Markdown
		md, err := htmltomarkdown.ConvertString(j.Description)
		if err == nil {
			j.Description = md
		} else {
			log.Printf("HTML-to-markdown conversion failed for job %s at %s: %v", j.Title, j.Company, err)
		}

		detectedLevel, detectedType := s.detectJobMeta(j.Description)
		if detectedLevel == "" && level != "" {
			detectedLevel = level
		}

		j.Level = detectedLevel
		j.Type = detectedType

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

func (s *JobService) detectJobMeta(description string) (level, jobType string) {
	lower := strings.ToLower(description)

	switch {
	case strings.Contains(lower, "intern"):
		level = "Intern"
	case strings.Contains(lower, "junior"):
		level = "Junior"
	case strings.Contains(lower, "senior"):
		level = "Senior"
	}

	switch {
	case strings.Contains(lower, "fulltime"), strings.Contains(lower, "full-time"):
		jobType = "Fulltime"
	case strings.Contains(lower, "parttime"), strings.Contains(lower, "part-time"):
		jobType = "Parttime"
	}

	return
}
