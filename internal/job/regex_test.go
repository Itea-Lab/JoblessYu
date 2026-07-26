package job

import (
	"context"
	"strings"
	"testing"
)

func newExtractor() *RegexExtractor {
	return NewRegexExtractor()
}

func detectLevel(t *testing.T, title, desc string) string {
	t.Helper()
	meta, _ := newExtractor().Extract(context.Background(), title, desc)
	return meta.Level
}

func TestRegexExtractor_Unknown(t *testing.T) {
	cases := []struct {
		name  string
		title string
		desc  string
	}{
		{
			name:  "skill-heavy JD with no level word and no years",
			title: "DevSecOps Engineer (AWS, IAM/CloudFormation, English)",
			desc:  "Work with global clients on AWS environments. Strong CloudFormation, IAM, CI/CD pipelines, Wiz, Prisma Cloud. No mention of any seniority tier.",
		},
		{
			name:  "JD mentioning international must not classify as Intern",
			title: "Internal Tools Engineer",
			desc:  "International client. Work with our internal tooling team.",
		},
		{
			name:  "bare JD body",
			title: "Backend Engineer",
			desc:  "We are hiring. Apply now.",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectLevel(t, c.title, c.desc)
			if got != LevelUnknown {
				t.Errorf("Extract level = %q, want %q", got, LevelUnknown)
			}
		})
	}
}

func TestRegexExtractor_InternationalNotIntern(t *testing.T) {
	desc := `Join our international team. Internal tools, global clients.
You will collaborate across international offices. React, Node, AWS.
This is a full-time role.`

	lvl := detectLevel(t, "Engineer", desc)
	if strings.EqualFold(lvl, "Intern") {
		t.Fatalf("regression: international/internal JD classified as Intern")
	}
}

func TestRegexExtractor_LevelWords(t *testing.T) {
	cases := []struct {
		title string
		desc  string
		want  string
	}{
		{"Internship — Backend Engineer", "", "Intern"},
		{"Backend Engineer", "We are seeking an intern for...", "Intern"},
		{"Junior Backend Engineer", "", "Junior"},
		{"Backend Engineer", "Junior devs encouraged to apply", "Junior"},
		{"Senior Backend Engineer", "", "Senior"},
		{"Backend Engineer", "Looking for a senior with cloud experience", "Senior"},
		{"Fresher Backend Engineer", "", "Fresher"},
		{"Backend Engineer", "We are hiring a fresher for this role", "Fresher"},
	}
	for _, c := range cases {
		t.Run(c.title+"/"+c.desc, func(t *testing.T) {
			if got := detectLevel(t, c.title, c.desc); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestRegexExtractor_YearsOfExperience(t *testing.T) {
	cases := []struct {
		yearsDesc string
		want      string
	}{
		{"0 years of experience", "Intern"},
		{"1 year of Go", "Fresher"},
		{"2+ yrs experience", "Fresher"},
		{"3 years of professional experience", "Junior"},
		{"4 years experience", "Junior"},
		{"5+ years experience", "Senior"},
		{"8 years experience", "Senior"},
		{"12 years of experience", "Senior"},
	}
	for _, c := range cases {
		t.Run(c.yearsDesc, func(t *testing.T) {
			if got := detectLevel(t, "Engineer", c.yearsDesc); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestRegexExtractor_NoExperience(t *testing.T) {
	cases := []struct {
		name string
		desc string
	}{
		{"no_experience", "No experience required. We will train you."},
		{"zero_experience", "Zero exp needed; we'll mentor you."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := detectLevel(t, "Backend Engineer", c.desc); got != "Intern" {
				t.Errorf("got %q, want Intern", got)
			}
		})
	}
}

func TestRegexExtractor_LevelUnknownContract(t *testing.T) {
	desc := "AWS, CloudFormation, IAM, CI/CD, Wiz, Prisma Cloud. Fast-paced startup."
	if lvl := detectLevel(t, "Cloud Engineer", desc); lvl != LevelUnknown {
		t.Fatalf("expected %q sentinel, got %q — service cannot apply strict filtering without it", LevelUnknown, lvl)
	}
}

func TestRegexExtractor_ReturnsFullMeta(t *testing.T) {
	meta, err := newExtractor().Extract(context.Background(), "Senior Go Engineer", "5+ years of Go. AWS, Docker, Kubernetes. Full-time.")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Level != "Senior" {
		t.Errorf("Level = %q, want Senior", meta.Level)
	}
	if meta.Type != "Full-time" {
		t.Errorf("Type = %q, want Full-time", meta.Type)
	}
	if len(meta.Tags) == 0 {
		t.Error("Tags empty; expected Cloud/Languages/Containers categories")
	}
	if meta.Salary != "" {
		t.Errorf("Salary = %q, want empty (regex cannot extract)", meta.Salary)
	}
	if meta.Remote {
		t.Error("Remote = true, want false (regex cannot extract)")
	}
	if meta.Summary != "" {
		t.Errorf("Summary = %q, want empty (regex cannot extract)", meta.Summary)
	}
	if meta.Model != "regex" {
		t.Errorf("Model = %q, want \"regex\"", meta.Model)
	}
}

func TestRegexExtractor_DetectType(t *testing.T) {
	ext := newExtractor()
	cases := []struct {
		desc string
		want string
	}{
		{"This is a full-time role in HCM.", "Full-time"},
		{"Looking for a part time developer.", "Part-time"},
		{"We are hiring software engineers.", ""},
	}
	for _, c := range cases {
		got := ext.detectType(strings.ToLower(c.desc))
		if got != c.want {
			t.Errorf("detectType(%q) = %q, want %q", c.desc, got, c.want)
		}
	}
}

func TestDetectTags_KMSDevSecOpsJD(t *testing.T) {
	jd := `DEVSECOPS ENGINEER (AWS, IAM/CLOUDFORMATION, ENGLISH)
KMS Technology - Ho Chi Minh City

Responsibilities
Work directly with global clients and onshore SRE / Platform teams to investigate AWS environments.
Analyze and remediate AWS security findings, including misconfigurations, exposed credentials,
overly permissive IAM policies, and related vulnerabilities. Design, implement, and review AWS IAM
policies, roles, SCPs, and permission boundaries following least-privilege principles. Contribute to
or review Infrastructure as Code, especially CloudFormation. Maintain and improve secure, scalable
AWS infrastructure in multi-account / multi-environment setups. Strong AWS networking knowledge,
including VPC, Security Groups, VPN, Transit Gateway. Experience with Wiz or similar CSPM
tools such as Prisma Cloud, AWS Security Hub. Proficiency in scripting languages such as Python, Bash,
or PowerShell. CI/CD pipelines. Experience with multi-account AWS setups. Familiarity with containers,
microservices, and cloud deployment practices is a plus Experience using AI chat tools (ChatGPT, Claude,
Gemini) for research. PHP applications on AWS during infrastructure migration.`

	got := DetectTags(jd)

	cases := []struct {
		category string
		want     []string
	}{
		{"Cloud", []string{"AWS", "IAM", "KMS", "SCP", "VPC", "Security Groups", "Transit Gateway"}},
		{"IaC", []string{"CloudFormation", "IaC"}},
		{"Pipeline", []string{"CI/CD"}},
		{"Containers", []string{"Microservices"}},
		{"Security", []string{"CSPM", "DevSecOps", "IAM", "Least Privilege", "Prisma Cloud", "SCP", "Wiz", "AWS Security Hub", "Permission Boundary"}},
		{"Languages", []string{"Bash", "PHP", "PowerShell", "Python"}},
		{"AI", []string{"ChatGPT", "Claude", "Gemini"}},
	}
	for _, c := range cases {
		gotTags, ok := got[c.category]
		if !ok {
			t.Errorf("category %q missing; got %v", c.category, got)
			continue
		}
		if !subset(c.want, gotTags) {
			t.Errorf("category %q: got %v, want (at least) %v", c.category, gotTags, c.want)
		}
	}

	if _, ok := flatten(got)["Intern"]; ok {
		t.Errorf("Intern surfaced as a skill tag — it should only be a Level")
	}
}

func TestDetectTags_NoFalsePositiveLevel(t *testing.T) {
	jd := `International client requires an Internal tools engineer.
Work with our international team. No mention of interns or internships here.`

	got := DetectTags(jd)

	for cat := range got {
		if cat == "Languages" && containsString(got[cat], "Go") {
			t.Errorf("'Go' tag should not trigger for word 'internal'")
		}
	}

	flat := flatten(got)
	for _, levelTag := range []string{"Intern", "Junior", "Senior", "Fresher"} {
		if _, ok := flat[levelTag]; ok {
			t.Errorf("level %q surfaced as a skill tag; tags = %v", levelTag, got)
		}
	}
}

func TestDetectTags_Empty(t *testing.T) {
	got := DetectTags("")
	if len(got) != 0 {
		t.Errorf("empty input = %v, want empty map", got)
	}
}

func TestDetectTags_CanonicalForm(t *testing.T) {
	got := DetectTags("Experience with cfn and github actions and k8s.")
	if !containsString(got["Pipeline"], "GitHub Actions") {
		t.Errorf("GitHub Actions not canonicalized; got %v", got["Pipeline"])
	}
	if !containsString(got["Containers"], "Kubernetes") {
		t.Errorf("k8s → Kubernetes canonicalization failed; got %v", got["Containers"])
	}
}

// helpers

func subset(want, got []string) bool {
	set := make(map[string]struct{}, len(got))
	for _, g := range got {
		set[g] = struct{}{}
	}
	for _, w := range want {
		if _, ok := set[w]; !ok {
			return false
		}
	}
	return true
}

func containsString(list []string, target string) bool {
	for _, l := range list {
		if l == target {
			return true
		}
	}
	return false
}

func flatten(tags map[string][]string) map[string]struct{} {
	out := make(map[string]struct{})
	for _, list := range tags {
		for _, t := range list {
			out[t] = struct{}{}
		}
	}
	return out
}

// --- Level-word regex engine tests (moved from domain/levels_test.go) ---
// These exercise the Go regex engine that consumes LevelRules.
// The data-layer tests (LevelRules content + LevelUnknown const) stay in
// domain/levels_test.go.

func TestNoExpAlternation(t *testing.T) {
	got := noExpAlternation()
	if got == "" {
		t.Fatalf("noExpAlternation() = empty; at least Intern must contribute a no-exp phrase")
	}
	// The current canonical Intern no-exp source. Pinning it catches
	// accidental edits to the Intern NoExpPattern without a matching update
	// to consumers (RegexExtractor.noExpRe is byte-identical to this source
	// wrapped in word boundaries).
	want := `(no|zero)\s+(experience|exp)`
	if got != want {
		t.Errorf("noExpAlternation() = %q, want %q", got, want)
	}
}

func TestFindLevelWord(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"We are hiring an intern for the team.", "Intern"},
		{"Join our internship program.", "Intern"},
		{"Looking for a junior dev.", "Junior"},
		{"Senior backend engineer wanted.", "Senior"},
		{"We need a fresher with Python skills.", "Fresher"},
		{"International client, internal team.", LevelUnknown},
		{"Seniority is not required.", LevelUnknown}, // "seniority" must NOT match senior
		{"", LevelUnknown},
	}
	for _, c := range cases {
		t.Run(c.text, func(t *testing.T) {
			if got := findLevelWord(c.text); got != c.want {
				t.Errorf("findLevelWord(%q) = %q, want %q", c.text, got, c.want)
			}
		})
	}
}

// TestFindLevelWord_InternalNotIntern is a focused regression test for the
// original "international → Intern" misclassification that motivated the
// refactoring to a single source of truth.
func TestFindLevelWord_InternalNotIntern(t *testing.T) {
	if got := findLevelWord("International team, internal tools, no interns here."); got == "Intern" {
		t.Errorf("regression: got Intern from international/internal; got=%q", got)
	}
}
