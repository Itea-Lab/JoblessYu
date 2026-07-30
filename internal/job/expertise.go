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
		Label:    "IT Executive & Management",
		Keywords: []string{"project manager", "product manager", "cto", "cio", "ciso", "cdo", "director", "vp", "head of", "program manager", "pmo", "manager"},
	},
	{
		Value:    "web_dev",
		Label:    "Web Application Development",
		Keywords: []string{"backend", "frontend", "front-end", "back-end", "fullstack", "full-stack", "web developer", "node.js", "react", "vue", "angular", "html", "css", "javascript", "php", "wordpress"},
	},
	{
		Value:    "mobile_dev",
		Label:    "Mobile Application Development",
		Keywords: []string{"ios", "android", "mobile", "flutter", "react native", "swift", "kotlin", "mobile developer"},
	},
	{
		Value:    "enterprise",
		Label:    "Core / Enterprise Systems",
		Keywords: []string{"erp", "crm", "sap", "oracle", "banking system", "salesforce", "dynamics", "integration", "legacy"},
	},
	{
		Value:    "lowcode_nocode",
		Label:    "Low-Code / No-Code Development",
		Keywords: []string{"low-code", "no-code", "rpa", "uipath", "automation anywhere", "power apps", "mendix", "outsystems"},
	},
	{
		Value:    "architecture",
		Label:    "Technical Architecture",
		Keywords: []string{"architect", "solution architect", "enterprise architect", "technical architect", "software architect"},
	},
	{
		Value:    "blockchain",
		Label:    "Blockchain Development",
		Keywords: []string{"blockchain", "smart contract", "solidity", "web3", "crypto", "ethereum", "rust"},
	},
	{
		Value:    "game_dev",
		Label:    "Game Development",
		Keywords: []string{"game", "unity", "unreal", "godot", "game designer", "game producer", "game tester", "vr", "ar"},
	},
	{
		Value:    "testing_qa",
		Label:    "Software Testing & Quality Assurance",
		Keywords: []string{"qa", "tester", "test automation", "quality assurance", "sdet", "manual test", "automation test", "pqa", "performance test"},
	},
	{
		Value:    "data_analytics",
		Label:    "Data Analytics & BI",
		Keywords: []string{"data analyst", "bi analyst", "bi developer", "analytics engineer", "data visualization", "tableau", "power bi", "looker"},
	},
	{
		Value:    "data_engineering",
		Label:    "Data Engineering",
		Keywords: []string{"data engineer", "big data", "dataops", "mlops engineer", "database engineer", "etl", "spark", "hadoop", "airflow"},
	},
	{
		Value:    "data_ai",
		Label:    "Data Science & AI / ML",
		Keywords: []string{"machine learning", "ai engineer", "data scientist", "ml", "deep learning", "computer vision", "nlp", "ai researcher"},
	},
	{
		Value:    "data_governance",
		Label:    "Data Management & Governance",
		Keywords: []string{"data architect", "data governance", "data steward", "data quality", "dba", "database administrator"},
	},
	{
		Value:    "cloud",
		Label:    "Cloud Computing",
		Keywords: []string{"cloud engineer", "aws", "azure", "gcp", "cloud architect", "cloud practitioner"},
	},
	{
		Value:    "systems_network",
		Label:    "Systems & Network Admin",
		Keywords: []string{"network engineer", "system administrator", "systems engineer", "sysadmin", "sysops", "infrastructure", "linux administrator", "windows server"},
	},
	{
		Value:    "devops_sre",
		Label:    "DevOps & Site Reliability (SRE)",
		Keywords: []string{"devops", "kubernetes", "terraform", "sre", "site reliability", "ci/cd", "jenkins", "release manager"},
	},
	{
		Value:    "support_helpdesk",
		Label:    "IT Support & Helpdesk",
		Keywords: []string{"it support", "helpdesk", "help desk", "it administrator", "field support", "technical customer support"},
	},
	{
		Value:    "cybersecurity",
		Label:    "Cybersecurity",
		Keywords: []string{"security engineer", "cybersecurity", "penetration", "soc analyst", "application security", "devsecops", "security consultant"},
	},
	{
		Value:    "compliance_risk",
		Label:    "IT Compliance & Risk Management",
		Keywords: []string{"compliance", "grc", "it auditor", "risk manager", "security compliance"},
	},
	{
		Value:    "embedded_iot",
		Label:    "Embedded, IoT & Robotics",
		Keywords: []string{"embedded", "firmware", "iot", "robotics", "real-time", "microcontroller", "rtos", "stm32", "arduino", "edge computing"},
	},
	{
		Value:    "product_mgmt",
		Label:    "Product Management",
		Keywords: []string{"product manager", "product owner", "product analyst"},
	},
	{
		Value:    "project_mgmt",
		Label:    "Project Management & Tech Comm",
		Keywords: []string{"project manager", "program manager", "scrum master", "agile coach", "brse", "bridge system engineer", "business analyst", "it communicator", "technical writer"},
	},
	{
		Value:    "design_ux",
		Label:    "Design & User Experience",
		Keywords: []string{"ux", "ui designer", "product designer", "graphic designer", "figma", "ux/ui", "user experience", "motion designer", "ux researcher"},
	},
	{
		Value:    "consulting_sales",
		Label:    "IT Consulting & Sales",
		Keywords: []string{"consultant", "pre-sales", "presales", "technical account", "solution consultant", "it consulting", "erp consultant"},
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
