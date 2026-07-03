package domain

import (
	"regexp"
	"strings"
)

// LevelRule defines detection patterns for a single seniority level.
//
//	Name        — canonical level label ("Intern", "Fresher", "Junior", "Senior").
//	WordPattern — portable regex source (no anchors) for the level word.
//	              Used in two contexts:
//	                Go (service layer):   embedded as (?i)\b(<WordPattern>)\b
//	                Postgres (repo):      embedded as \m(<WordPattern>)\M with ~*
//	              Keep this source free of Go/PCRE-specific features so both
//	              engines compile it the same way.
//	YearPattern — Postgres regex (~*) matching a years-of-experience mention
//	              corresponding to this level, or empty if the level has no
//	              year bucket (currently all four levels have one).
//	NoExpPattern — portable regex source (no anchors, no boundaries) matching a
//	              "no experience required"-style phrase that implies this level
//	              without naming it (e.g. "(no|zero)\\s+(experience|exp)" for Intern).
//	              Empty means the level has no no-exp phrase (currently only
//	              Intern has one). Used in two contexts:
//	                Go (service layer):   embedded as (?i)\b(<NoExpPattern>)\b
//	                Postgres (repo):      embedded as \m(<NoExpPattern>)\M with ~*
//	              Same portability contract as WordPattern.
//
// Add a new level here and both the repository (SQL) and service (in-memory
// detection) layers pick it up automatically — this is the single source of
// truth. Do not duplicate level patterns in either consumer.
type LevelRule struct {
	Name         string
	WordPattern  string
	YearPattern  string
	NoExpPattern string
}

// LevelRules is the single source of truth for level detection. Order
// matters for service-layer FindLevelWord (first matching rule wins),
// but the current alternations are mutually exclusive so order is moot.
var LevelRules = []LevelRule{
	{Name: "Intern", WordPattern: `intern(ship)?`, YearPattern: `\m0\s*\+?\s*years?`, NoExpPattern: `(no|zero)\s+(experience|exp)`},
	{Name: "Fresher", WordPattern: `fresher`, YearPattern: `\m(1|2)\s*\+?\s*years?`},
	{Name: "Junior", WordPattern: `junior`, YearPattern: `\m(3|4)\s*\+?\s*years?`},
	{Name: "Senior", WordPattern: `senior`, YearPattern: `\m([5-9]|\d{2})\s*\+?\s*years?`},
}

// LevelUnknown is the canonical sentinel returned when detection finds no
// level signal (no level word, no years-of-experience, no "no experience"
// phrase). It is a real Level value, distinct from the empty string used
// by the DB column to mean "not yet classified".
const LevelUnknown = "Unknown"

// levelWordRE matches any level word across all rules, case-insensitively
// with Go's \b word boundaries. The matched substring can be resolved to
// a Name via FindLevelWord below.
var levelWordRE = regexp.MustCompile(`(?i)\b(` + LevelWordAlternationForRegex() + `)\b`)

// perRuleWordRE[i] matches Rule i's word pattern inside word boundaries.
// Pre-compiled so FindLevelWord is allocation-free per call.
var perRuleWordRE = compilePerRuleWordRE()

// LevelWordAlternationForRegex returns the level word patterns joined by
// "|", intended to be embedded inside a larger regex by the consumer, e.g.
//
//	levelRe = regexp.MustCompile(`(?i)\b(` + domain.LevelWordAlternationForRegex() + `)\b`)
//
// Callers are responsible for wrapping it with their engine's word-boundary
// and case-sensitivity syntax. Avoid hot-loop calls; the slice is small but
// the join allocates.
func LevelWordAlternationForRegex() string {
	parts := make([]string, len(LevelRules))
	for i, r := range LevelRules {
		parts[i] = r.WordPattern
	}
	return strings.Join(parts, "|")
}

// NoExpAlternationForRegex returns the non-empty NoExpPattern entries joined
// by "|", intended to be embedded inside a larger regex by the consumer —
// same contract as LevelWordAlternationForRegex. Callers wrap with their
// engine's word-boundary + case-insensitivity syntax. Avoid hot-loop calls;
// the slice is small but the join allocates. Returns "" when no rule has a
// NoExpPattern (today only Intern does, so the result is non-empty).
func NoExpAlternationForRegex() string {
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

// FindLevelWord scans text for the first level word and returns the
// corresponding LevelRule.Name. Returns LevelUnknown when no level word
// is found anywhere in the text.
func FindLevelWord(text string) string {
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
