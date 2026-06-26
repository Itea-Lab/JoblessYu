package domain

type JobEntry struct {
	Title       string
	Company     string
	Location    string
	URL         string
	Description string
	Level       string // detected: Intern, Junior, Senior
	Type        string // detected: Fulltime, Parttime
}
