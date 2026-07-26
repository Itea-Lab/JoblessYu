package job

import (
	"regexp"
)

// ExpertiseUnknown is the sentinel for unclassifiable jobs.
const ExpertiseUnknown = "unknown"

// ExpertiseCategory defines a broad job category for the Discord dropdown
// and DB filtering. Each category has a DB value (lowercase with underscores),
// a display label, and keywords for regex-based detection.
type ExpertiseCategory struct {
	Value    string   // DB value, e.g. "web_dev"
	Label    string   // Display label, e.g. "Web Development"
	Keywords []string // Keywords for regex detection, lowercased
}

// ExpertiseCategories is the canonical list of 13 broad expertise categories.
// The AI prompt (skills.md) and regex fallback (DetectExpertise) both use
// these values. Discord UI will use Label for display, Value for filtering.
var ExpertiseCategories = []ExpertiseCategory{
	{
		Value:    "management",
		Label:    "Management & Executive",
		Keywords: []string{"project manager", "product manager", "cto", "cio", "ciso", "director", "vp", "head of", "program manager", "pmo"},
	},
	{
		Value:    "web_dev",
		Label:    "Web Development",
		Keywords: []string{"backend", "frontend", "front-end", "back-end", "fullstack", "full-stack", "web developer", "node.js", "react", "vue", "angular", "html", "css", "javascript", "php", "wordpress"},
	},
	{
		Value:    "mobile_game",
		Label:    "Mobile & Game Development",
		Keywords: []string{"ios", "android", "mobile", "flutter", "react native", "swift", "kotlin", "game", "unity", "unreal", "godot"},
	},
	{
		Value:    "enterprise",
		Label:    "Enterprise Systems",
		Keywords: []string{"erp", "crm", "sap", "oracle", "rpa", "low-code", "no-code", "banking system", "salesforce", "dynamics"},
	},
	{
		Value:    "architecture",
		Label:    "Architecture",
		Keywords: []string{"architect", "solution architect", "enterprise architect", "technical architect", "software architect"},
	},
	{
		Value:    "data_ai",
		Label:    "Data & AI",
		Keywords: []string{"data analyst", "data engineer", "machine learning", "ai engineer", "data scientist", "ml", "deep learning", "computer vision", "nlp", "big data"},
	},
	{
		Value:    "cloud_devops",
		Label:    "Cloud & DevOps",
		Keywords: []string{"devops", "cloud engineer", "aws", "azure", "gcp", "kubernetes", "terraform", "sre", "site reliability", "ci/cd", "jenkins"},
	},
	{
		Value:    "systems_network",
		Label:    "Systems & Network",
		Keywords: []string{"network engineer", "system administrator", "sysadmin", "sysops", "infrastructure", "linux administrator", "windows server"},
	},
	{
		Value:    "support_security",
		Label:    "IT Support & Security",
		Keywords: []string{"it support", "helpdesk", "help desk", "security engineer", "cybersecurity", "penetration", "soc analyst", "it helpdesk", "technical support"},
	},
	{
		Value:    "embedded_iot",
		Label:    "Embedded & IoT",
		Keywords: []string{"embedded", "firmware", "iot", "robotics", "real-time", "microcontroller", "rtos", "stm32", "arduino"},
	},
	{
		Value:    "testing_qa",
		Label:    "Testing & QA",
		Keywords: []string{"qa", "tester", "test automation", "quality assurance", "sdet", "manual test", "automation test"},
	},
	{
		Value:    "design_ux",
		Label:    "Design & UX",
		Keywords: []string{"ux", "ui designer", "product designer", "graphic designer", "figma", "ux/ui", "user experience", "user interface"},
	},
}

// ExpertiseLabelMap maps DB values to display labels for the Discord UI.
var ExpertiseLabelMap = func() map[string]string {
	m := make(map[string]string, len(ExpertiseCategories)+1)
	for _, c := range ExpertiseCategories {
		m[c.Value] = c.Label
	}
	m[ExpertiseUnknown] = "Unknown"
	return m
}()

// ExpertiseLabel returns the display label for a DB expertise value.
// Returns "Unknown" if the value is empty or unrecognized.
func ExpertiseLabel(value string) string {
	if value == "" {
		return "Unknown"
	}
	if label, ok := ExpertiseLabelMap[value]; ok {
		return label
	}
	return "Unknown"
}

// IsValidExpertise returns true if the value is one of the 13 canonical
// categories or "unknown". Used by groq.go to validate AI-returned values.
func IsValidExpertise(value string) bool {
	_, ok := ExpertiseLabelMap[value]
	return ok
}

// expertiseKeywordRes holds pre-compiled word-boundary regexes for each
// keyword, flattened across all categories. Using \b word boundaries prevents
// false matches (e.g. "ml" no longer matches "html", "ux" no longer matches
// "linux", "qa" no longer matches "Iraq").
var expertiseKeywordRes = func() []*regexp.Regexp {
	var res []*regexp.Regexp
	for _, cat := range ExpertiseCategories {
		for _, kw := range cat.Keywords {
			res = append(res, regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(kw)+`\b`))
		}
	}
	return res
}()

// expertiseKeywordOffsets maps category index → start offset in the flattened
// regex slice. Built once at init; used by DetectExpertise to iterate per-category.
var expertiseKeywordOffsets = func() []int {
	offsets := make([]int, len(ExpertiseCategories))
	idx := 0
	for i, cat := range ExpertiseCategories {
		offsets[i] = idx
		idx += len(cat.Keywords)
	}
	return offsets
}()

// DetectExpertise scans title+description for keyword matches using
// word-boundary regexes (prevents "ml" matching "html", etc.).
// Title is checked first (more specific signal), then description.
// Returns the first matching category value, or ExpertiseUnknown if no match.
//
// This is the regex-fallback path used by:
//   - RegexExtractor.Extract() as the expertise classifier
//   - groq.go callGroq() to correct invalid AI-returned expertise values
func DetectExpertise(title, description string) string {
	// Pass 1: check title only (most specific signal).
	for i, cat := range ExpertiseCategories {
		offset := expertiseKeywordOffsets[i]
		for j := range cat.Keywords {
			if expertiseKeywordRes[offset+j].MatchString(title) {
				return cat.Value
			}
		}
	}

	// Pass 2: check description (broader signal).
	for i, cat := range ExpertiseCategories {
		offset := expertiseKeywordOffsets[i]
		for j := range cat.Keywords {
			if expertiseKeywordRes[offset+j].MatchString(description) {
				return cat.Value
			}
		}
	}

	return ExpertiseUnknown
}
