package job

// LevelRule defines detection patterns for a single seniority level.
//
//	Name        — canonical level label ("Intern", "Fresher", "Junior", "Senior").
//	WordPattern — portable regex source (no anchors) for the level word.
//	              Used in two contexts:
//	                Go (ai package):      embedded as (?i)\b(<WordPattern>)\b
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
//	                Go (ai package):      embedded as (?i)\b(<NoExpPattern>)\b
//	                Postgres (repo):      embedded as \m(<NoExpPattern>)\M with ~*
//	              Same portability contract as WordPattern.
//
// Add a new level here and both the repository (SQL) and ai (in-memory
// detection) layers pick it up automatically — this is the single source of
// truth for the pattern strings. The Go regex engine that consumes them
// lives in regex.go (private).
type LevelRule struct {
	Name         string
	WordPattern  string
	YearPattern  string
	NoExpPattern string
}

// LevelRules is the single source of truth for level detection patterns.
// Order matters for the ai package's findLevelWord (first matching rule
// wins), but the current alternations are mutually exclusive so order is
// moot.
var LevelRules = []LevelRule{
	{Name: "Intern", WordPattern: `intern(ship)?`, YearPattern: `\m0\s*\+?\s*years?`, NoExpPattern: `(no|zero)\s+(experience|exp)`},
	{Name: "Fresher", WordPattern: `fresher`, YearPattern: `\m(1|2)\s*\+?\s*years?`},
	{Name: "Junior", WordPattern: `junior`, YearPattern: `\m(3|4)\s*\+?\s*years?`},
	{Name: "Senior", WordPattern: `senior|lead|principal|staff`, YearPattern: `\m([5-9]|\d{2})\s*\+?\s*years?`},
}

// LevelUnknown is the canonical sentinel returned when detection finds no
// level signal (no level word, no years-of-experience, no "no experience"
// phrase). It is a real Level value, distinct from the empty string used
// by the DB column to mean "not yet classified".
const LevelUnknown = "Unknown"
