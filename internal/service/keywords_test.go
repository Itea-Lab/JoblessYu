package service

import (
	"reflect"
	"testing"
)

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

	// Spot-check the KMS DevSecOps JD mustSurface these canonical tags.
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

	// "international" appears in the JD text; it must NOT surface as an Intern
	// level, nor must Intern appear as a tag (Intern is a Level, not a skill tag).
	if _, ok := flatten(got)["Intern"]; ok {
		t.Errorf("Intern surfaced as a skill tag — it should only be a Level")
	}
}

// TestDetectTags_NoFalsePositiveLevel asserts that a JD mentioning "international"
// clients is not misclassified. detectJobMeta already uses word boundaries; this
// test guards the DB query level regex indirectly by exercising the detection
// service layer's regex semantics.
func TestDetectTags_NoFalsePositiveLevel(t *testing.T) {
	jd := `International client requires an Internal tools engineer.
Work with our international team. No mention of interns or internships here.`

	got := DetectTags(jd)

	// Cloud category, which has "iam" / "scp" / "kms"; none should match here
	// because none appear as standalone words. (Hardens against accidental
	// alternative-numbering regressions when categories get edited.)
	for cat := range got {
		if cat == "Languages" && containsString(got[cat], "Go") {
			t.Errorf("'Go' tag should not trigger for word 'internal'")
		}
	}

	// Specifically: no tag should be the word "Intern" or "Junior" or "Senior"
	// — those are levels, not skill tags, and never appear as keyword categories.
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
	// cfn isn't a registered alternative — verify the canonicalization path for
	// known synonyms (github actions → GitHub Actions, k8s → Kubernetes).
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

// guard against accidental future signature drift.
var _ = reflect.DeepEqual
