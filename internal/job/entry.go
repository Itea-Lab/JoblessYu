package job

import "time"

type JobEntry struct {
	ID            int64 // DB row id; needed by repository.MarkAIProcessed. Zero when not from DB.
	Title         string
	Company       string
	Location      string
	URL           string
	Site          string // source: "indeed", "linkedin", "itviec"
	Description   string
	Level         string              // detected: Intern, Junior, Senior, Unknown
	Type          string              // detected: Fulltime, Parttime (AI-normalized when available)
	Expertise     string              // broad category: web_dev, cloud_devops, etc. (empty when un-enriched)
	Tags          map[string][]string // detected skill/tool tags, keyed by category
	AIProcessed   bool                // true when ai_processed_at IS NOT NULL in DB
	AIProcessedAt time.Time           // when the row was last enriched (zero when never).
	Summary       string              // AI-generated JD summary (empty when regex-classified)
	Salary        string              // AI-extracted salary range (empty when not mentioned)
	Remote        bool                // AI-detected remote-eligible flag
	AIModel       string              // which extractor produced the result (e.g. "regex", "llama-3.1-8b-instant")
}

// JobMeta is the result of AI (or regex-fallback) extraction on a job
// description. It is the portable contract between the AI module and the
// service layer. Fields not extractable by the regex fallback (Salary,
// Remote, Summary) are left zero-valued and populated by the Groq extractor.
type JobMeta struct {
	Level     string
	Type      string
	Expertise string // broad category: web_dev, cloud_devops, etc.
	Tags      map[string][]string
	Salary    string // populated by AI; "" from regex fallback
	Remote    bool   // populated by AI; false from regex fallback
	Summary   string // populated by AI; "" from regex fallback
	Model     string // identifies which extractor produced this result (e.g. "regex", "llama-3.1-8b-instant")
}

// JobQuery is the filter contract between the bot/service and the
// repository. Keeping it in domain avoids a service→repository dependency
// for the query type alone.
type JobQuery struct {
	Level          string
	JobType        string
	Location       string
	Expertise      string // broad category filter: web_dev, cloud_devops, etc.
	IncludeUnknown bool
	Limit          int
	AIEnabled      bool // when true, skip DB-side regex pre-filter so AI sees more rows
}
