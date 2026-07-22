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
	// noExpRe is DERIVED from internal/domain.LevelRules (single source of
	// truth for the no-experience-phrase pattern). Compiled once at package
	// init to avoid re-allocating on each detectJobMeta call. Keep the
	// canonical source in domain.LevelRules; if a new level gains its own
	// no-exp phrase there, extend the helper rather than this literal.
	noExpRe = regexp.MustCompile(`(?i)\b` + domain.NoExpAlternationForRegex() + `\b`)
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

func (s *JobService) FetchAndProcessJobs(ctx context.Context, level, jobType, location string, includeUnknown bool) ([]domain.JobEntry, error) {
	// Pass includeUnknown through to the repository so it can drop the
	// DB-side level predicate entirely when the caller wants unclassified
	// rows to reach service-layer classification (see FetchRawJobs docs).
	rawJobs, err := s.repo.FetchRawJobs(ctx, level, jobType, location, includeUnknown)
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

		// Scan for skill/tool tags. Reuse the same HTML-stripped plain text
		// used by detectJobMeta so attribute values can't false-match.
		j.Tags = DetectTags(htmlRe.ReplaceAllString(j.Title+"\n"+j.Description, " "))

		// Strict level filtering: when the caller asks for a specific level,
		// any JD that detected to a different level is skipped. Unknown is
		// skipped too unless the caller explicitly opted in via
		// includeUnknown (scratch: "I want to see unclassified JDs anyway").
		if level != "" {
			if strings.EqualFold(j.Level, level) {
				// exact match — keep
			} else if strings.EqualFold(j.Level, domain.LevelUnknown) && includeUnknown {
				// keep
			} else {
				continue
			}
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
	// domain.FindLevelWord is the single source of truth for level-word
	// detection — it returns LevelUnknown when no level word matches, which
	// we use as the signal to try the years-of-experience and "no experience"
	// fallbacks below.
	level = domain.FindLevelWord(plain)
	if level == domain.LevelUnknown {
		// Pick the highest year-of-experience mention across the whole text.
		// JDs commonly list parallel skill requirements (e.g. "1 year React
		// AND 5 years Node"); the seniority bar is set by the maximum, not
		// the leftmost or the sum.
		matches := yearsRe.FindAllStringSubmatch(hay, -1)
		maxYears := -1
		for _, m := range matches {
			var n int
			if _, err := fmt.Sscanf(m[1], "%d", &n); err == nil && n > maxYears {
				maxYears = n
			}
		}
		if maxYears >= 0 {
			switch {
			case maxYears == 0:
				level = "Intern"
			case maxYears <= 2:
				level = "Fresher"
			case maxYears <= 4:
				level = "Junior"
			default:
				level = "Senior"
			}
		} else if noExpRe.MatchString(hay) {
			level = "Intern"
		}
		// else: level stays domain.LevelUnknown — sentinel beats empty
		// string: a real Level value prevents the fetch filter from silently
		// treating "unknown" as "matches any level the user asked for".
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
