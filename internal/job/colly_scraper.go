package job

import (
	"context"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

const (
	itviecListingURL = "https://itviec.com/it-jobs"
	itviecMaxJobs    = 20
	itviecDelay      = 2 * time.Second
)

// jobIDRe matches the numeric job ID at the end of an ITViec job URL.
var jobIDRe = regexp.MustCompile(`-\d{3,}`)

// CollyScraper scrapes ITViec job listings using the Colly framework.
// ITViec is a Rails app with fully server-rendered HTML — no JS needed.
//
// Two-step approach with a single collector:
//  1. Visit the listing page → OnHTML finds job card links → visits each
//  2. Each job page → OnHTML extracts the full job description
type CollyScraper struct{}

func NewCollyScraper() *CollyScraper {
	return &CollyScraper{}
}

// ScrapeITViec scrapes the ITViec job listing page and returns individual
// job entries with full descriptions.
func (s *CollyScraper) ScrapeITViec(ctx context.Context) ([]JobEntry, error) {
	var jobs []JobEntry
	visited := make(map[string]bool)
	queuedCount := 0 // tracks URLs queued for visiting (not jobs extracted)

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	)
	c.SetRequestTimeout(30 * time.Second)
	c.Limit(&colly.LimitRule{
		DomainGlob:  "itviec.com",
		Delay:       itviecDelay,
		RandomDelay: 1 * time.Second,
	})

	// Handle job cards on the listing page → extract URL + check freshness → visit.
	// Limit to itviecMaxJobs URLs — stops collecting after the cap is reached.
	// Skips jobs posted more than 24 hours ago (matches jobspy's hours_old=24).
	c.OnHTML("div.job-card", func(e *colly.HTMLElement) {
		if queuedCount >= itviecMaxJobs {
			return
		}

		// Parse "Posted X hours/days ago" — skip stale jobs.
		postedText := e.ChildText(".small-text.text-dark-grey")
		if age := parsePostedAge(postedText); age > 24 {
			return
		}

		href := e.ChildAttr("h3 a", "href")
		if href == "" {
			return
		}
		// Only collect URLs with a numeric job ID.
		if !jobIDRe.MatchString(href) {
			return
		}
		// Normalize: strip query params, ensure absolute URL.
		if strings.HasPrefix(href, "/") {
			href = "https://itviec.com" + href
		}
		href = strings.Split(href, "?")[0]

		if visited[href] {
			return
		}
		visited[href] = true
		queuedCount++
		e.Request.Visit(href)
	})

	// Handle individual job pages → extract the full JD.
	c.OnHTML("body", func(e *colly.HTMLElement) {
		url := e.Request.URL.String()

		// Skip the listing page itself — only process individual job pages.
		if url == itviecListingURL || !jobIDRe.MatchString(url) {
			return
		}

		title := cleanJobTitle(strings.TrimSpace(e.DOM.Find("h1").First().Text()))
		if title == "" {
			title = cleanJobTitle(strings.TrimSpace(e.ChildText("h1")))
		}

		// Company: target employer link specifically.
		// Avoid e.ChildText on broad selectors which concatenates all company links on the page.
		var company string
		if text := strings.TrimSpace(e.ChildText(".employer-name")); text != "" {
			company = text
		} else if text := strings.TrimSpace(e.ChildText(".job-details__sub-title")); text != "" {
			company = text
		} else if text := strings.TrimSpace(e.DOM.Find(`a[href*="/companies/"]`).First().Text()); text != "" {
			company = text
		}
		if idx := strings.Index(company, "\n"); idx >= 0 {
			company = strings.TrimSpace(company[:idx])
		}
		if len(company) > 100 {
			company = company[:100]
		}

		// Location.
		var location string
		if text := strings.TrimSpace(e.ChildText(".job-info .location")); text != "" {
			location = text
		} else if text := strings.TrimSpace(e.DOM.Find(".location").First().Text()); text != "" {
			location = text
		} else if text := strings.TrimSpace(e.DOM.Find("[class*='location']").First().Text()); text != "" {
			location = text
		} else {
			location = "Vietnam"
		}
		if idx := strings.Index(location, "\n"); idx >= 0 {
			location = strings.TrimSpace(location[:idx])
		}
		if len(location) > 80 {
			location = location[:80]
		}

		// Working model: ITViec shows "At office" / "Remote" / "Hybrid" as a badge.
		// The CSS class .text-rich-grey.flex-shrink-0 is generic and matches multiple
		// metadata elements (salary, location, etc.), so we iterate and only accept
		// the FIRST valid working-model value to avoid concatenating unrelated text.
		var workingModel string
		var workingModelFound bool
		e.ForEach(".text-rich-grey.flex-shrink-0", func(_ int, el *colly.HTMLElement) {
			if workingModelFound {
				return
			}
			text := strings.TrimSpace(el.Text)
			switch strings.ToLower(text) {
			case "at office", "remote", "hybrid":
				workingModel = text
				workingModelFound = true
			}
		})
		remote := strings.EqualFold(workingModel, "Remote") || strings.EqualFold(workingModel, "Hybrid")

		// Description: try multiple targeted selectors.
		var description string
		for _, sel := range []string{
			".job-description",
			".job-detail__description",
			".job-details__content",
			".job-details",
			"[class*='description']",
		} {
			if text := strings.TrimSpace(e.ChildText(sel)); len(text) >= 50 {
				description = text
				break
			}
		}

		if title == "" || len(description) < 50 {
			slog.Warn("colly: invalid job page (missing title or description < 50 chars), skipping", "url", url)
			return
		}

		jobs = append(jobs, JobEntry{
			Title:       title,
			Company:     company,
			Location:    location,
			URL:         url,
			Site:        "itviec",
			Description: description,
			Remote:      remote,
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		slog.Warn("colly: request error", "url", r.Request.URL, "err", err)
	})

	// Start scraping from the listing page.
	if err := c.Visit(itviecListingURL); err != nil {
		return nil, err
	}
	c.Wait()

	slog.Info("colly: ITViec scrape complete", "jobs", len(jobs))
	return jobs, nil
}

// ScrapeJobs implements the generic scraper interface for the ScraperManager.
func (s *CollyScraper) ScrapeJobs(ctx context.Context) ([]JobEntry, error) {
	return s.ScrapeITViec(ctx)
}

// cleanJobTitle removes common suffixes like " | ITviec" from the page title.
func cleanJobTitle(title string) string {
	for _, sep := range []string{" | ITviec", " - ITviec", " | Jobs in Viet Nam"} {
		if idx := strings.Index(title, sep); idx > 0 {
			title = title[:idx]
			break
		}
	}
	return strings.TrimSpace(title)
}

// numberRe extracts the first integer from a string (e.g. "5 hours ago" → 5).
var numberRe = regexp.MustCompile(`\d+`)

// parsePostedAge extracts the posting age in hours from ITViec's "Posted X ago" text.
// Returns -1 if the text cannot be parsed (treated as "fresh, allow scraping").
//
// Examples:
//
//	"Posted 5 hours ago"     → 5
//	"Posted 30 minutes ago"  → 0
//	"Posted 1 day ago"       → 24
//	"Posted 2 days ago"      → 48
//	"" / unparseable          → -1
func parsePostedAge(text string) int {
	text = strings.ToLower(text)
	numStr := numberRe.FindString(text)
	if numStr == "" {
		return -1
	}
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return -1
	}
	switch {
	case strings.Contains(text, "minute"):
		return 0
	case strings.Contains(text, "hour"):
		return num
	case strings.Contains(text, "day"):
		return num * 24
	default:
		return -1
	}
}
