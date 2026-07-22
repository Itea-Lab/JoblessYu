package job

import (
	"context"
	"fmt"
	"regexp"
	"strings"

)

// RegexExtractor is the no-dependency, always-succeeds Extractor. It
// replicates the level/type/tag detection that lived in
// service.JobService.detectJobMeta + service.DetectTags before Slice B.
//
// It populates only Level, Type, and Tags. Salary, Remote, and Summary
// are left zero-valued — those require the Groq extractor (Slice D).
type RegexExtractor struct {
	noExpRe *regexp.Regexp
	typeRe  *regexp.Regexp
	yearsRe *regexp.Regexp
	htmlRe  *regexp.Regexp
}

// NewRegexExtractor returns a ready-to-use RegexExtractor. Level-word and
// no-exp-phrase patterns are derived from LevelRules (the single
// source of truth for the pattern strings), so edits there propagate here
// without touching this file.
func NewRegexExtractor() *RegexExtractor {
	return &RegexExtractor{
		noExpRe: regexp.MustCompile(`(?i)\b` + noExpAlternation() + `\b`),
		typeRe:  regexp.MustCompile(`(?i)\b(full[ -]?time|part[ -]?time)\b`),
		yearsRe: regexp.MustCompile(`(?i)\b(\d{1,2})\s*\+?\s*(?:years?|yrs?)\b`),
		htmlRe:  regexp.MustCompile(`<[^>]*>`),
	}
}

func (r *RegexExtractor) Extract(_ context.Context, title, description string) (JobMeta, error) {
	plain := r.htmlRe.ReplaceAllString(title+"\n"+description, " ")
	hay := strings.ToLower(plain)

	level := r.detectLevel(plain, hay)
	jobType := r.detectType(hay)
	tags := DetectTags(r.htmlRe.ReplaceAllString(title+"\n"+description, " "))

	return JobMeta{
		Level: level,
		Type:  jobType,
		Tags:  tags,
		Model: "regex",
	}, nil
}

// detectLevel mirrors the pre-Slice-B logic: an explicit level word wins;
// otherwise years-of-experience buckets decide; otherwise the "no
// experience" phrase; otherwise LevelUnknown.
func (r *RegexExtractor) detectLevel(plain, hay string) string {
	level := findLevelWord(plain)
	if level != LevelUnknown {
		return level
	}

	matches := r.yearsRe.FindAllStringSubmatch(hay, -1)
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
			return "Intern"
		case maxYears <= 2:
			return "Fresher"
		case maxYears <= 4:
			return "Junior"
		default:
			return "Senior"
		}
	}
	if r.noExpRe.MatchString(hay) {
		return "Intern"
	}
	return LevelUnknown
}

func (r *RegexExtractor) detectType(hay string) string {
	m := r.typeRe.FindString(hay)
	if m == "" {
		return ""
	}
	switch strings.ToLower(m) {
	case "full-time", "full time", "fulltime":
		return "Fulltime"
	case "part-time", "part time", "parttime":
		return "Parttime"
	}
	return ""
}

// --- Level-word regex engine (private to this package) ---
//
// These helpers consume LevelRules (the data source of truth) and
// compile Go regexes from them. They live here, not in domain, because
// domain must stay pure data — no regexp imports, no package-init side
// effects. The repository reads the same LevelRules strings to build
// Postgres regex predicates (\m...\M), so the pattern source remains
// shared while the engines are colocated with their consumers.

// levelWordRE matches any level word across all rules, case-insensitively
// with Go's \b word boundaries. The matched substring can be resolved to
// a Name via findLevelWord below.
var levelWordRE = regexp.MustCompile(`(?i)\b(` + levelWordAlternation() + `)\b`)

// perRuleWordRE[i] matches Rule i's word pattern inside word boundaries.
// Pre-compiled so findLevelWord is allocation-free per call.
var perRuleWordRE = compilePerRuleWordRE()

func levelWordAlternation() string {
	parts := make([]string, len(LevelRules))
	for i, r := range LevelRules {
		parts[i] = r.WordPattern
	}
	return strings.Join(parts, "|")
}

func noExpAlternation() string {
	var parts []string
	for _, r := range LevelRules {
		if r.NoExpPattern != "" {
			parts = append(parts, r.NoExpPattern)
		}
	}
	return strings.Join(parts, "|")
}

func compilePerRuleWordRE() []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(LevelRules))
	for i, r := range LevelRules {
		out[i] = regexp.MustCompile(`(?i)\b` + r.WordPattern + `\b`)
	}
	return out
}

// findLevelWord scans text for the first level word and returns the
// corresponding LevelRule.Name. Returns LevelUnknown when no level
// word is found anywhere in the text.
func findLevelWord(text string) string {
	m := levelWordRE.FindString(text)
	if m == "" {
		return LevelUnknown
	}
	lower := strings.ToLower(m)
	for i, re := range perRuleWordRE {
		if re.MatchString(lower) {
			return LevelRules[i].Name
		}
	}
	return LevelUnknown
}
