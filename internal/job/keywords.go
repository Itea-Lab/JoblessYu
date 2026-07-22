package job

import (
	"regexp"
	"sort"
	"strings"
)

// KeywordCategory groups detection rules for a family of skills/tools.
// Each category has a single compiled regex with named alternatives; the
// matched literal (canonically-cased) is reported via MatchCategoryIndex.
//
// Categories are intentionally coarse-grained so the embed stays readable
// (e.g. we report "AWS" under "Cloud", not 30 individual AWS services as
// their own chips).
type KeywordCategory struct {
	Name    string
	Pattern *regexp.Regexp
	// Canonical forms keyed by lowercased match. If a match has no explicit
	// canonical form, the raw matched text is used as the tag.
	Canonical map[string]string
}

// keywordCategories is the data-driven keyword scanner. To add a new tag
// family, append an entry here — DetectTags does the rest.
//
// Each regex uses word boundaries (\b) and case-insensitive matching. Keep
// alternatives mutually-exclusive where possible; alternatives capture the
// variant to report (e.g. "ci/c?d" captures the actual "CI/CD" wording).
// Canonicalizing maps plural/variant spellings onto a single canonical chip
// (e.g. "cfn" → "CloudFormation", "github actions" → "GitHub Actions").
var keywordCategories = []KeywordCategory{
	{
		Name: "Cloud",
		Pattern: regexp.MustCompile(`(?i)\b(` +
			`aws|azure|gcp|google cloud|` +
			`amazon web services|microsoft azure|` +
			`vpcs?|ec2|s3|lambda|eks|ecs|fargate|rds|cloudfront|` +
			`transit gateways?|security groups?|route53|iams?|scps?|kms\b|secrets manager` +
			`)\b`),
		Canonical: map[string]string{
			"aws":                 "AWS",
			"amazon web services": "AWS",
			"azure":               "Azure",
			"gcp":                 "GCP",
			"google cloud":        "GCP",
			"vpc":                 "VPC",
			"vpcs":                "VPC",
			"ec2":                 "EC2",
			"s3":                  "S3",
			"lambda":              "Lambda",
			"eks":                 "EKS",
			"ecs":                 "ECS",
			"fargate":             "Fargate",
			"rds":                 "RDS",
			"cloudfront":          "CloudFront",
			"transit gateway":     "Transit Gateway",
			"transit gateways":    "Transit Gateway",
			"security group":      "Security Groups",
			"security groups":     "Security Groups",
			"route53":             "Route53",
			"iam":                 "IAM",
			"iams":                "IAM",
			"scp":                 "SCP",
			"scps":                "SCP",
			"kms":                 "KMS",
			"secrets manager":     "Secrets Manager",
		},
	},
	{
		Name: "IaC",
		Pattern: regexp.MustCompile(`(?i)\b(` +
			`cloudformation|terraform|cdk\b|infrastructure as code|ansible|pulumi` +
			`)\b`),
		Canonical: map[string]string{
			"cloudformation":         "CloudFormation",
			"terraform":              "Terraform",
			"cdk":                    "CDK",
			"infrastructure as code": "IaC",
			"ansible":                "Ansible",
			"pulumi":                 "Pulumi",
		},
	},
	{
		Name: "Pipeline",
		Pattern: regexp.MustCompile(`(?i)\b(` +
			`ci/c?d|continuous integration|continuous delivery|continuous deployment|` +
			`github actions|gitlab ci|jenkins|circleci|buildkite|argo ?cd|` +
			`spinnaker|teamcity|bitbucket pipelines` +
			`)\b`),
		Canonical: map[string]string{
			"ci/cd":                  "CI/CD",
			"ci cd":                  "CI/CD",
			"continuous integration": "CI",
			"continuous delivery":    "CD",
			"continuous deployment":  "CD",
			"github actions":         "GitHub Actions",
			"gitlab ci":              "GitLab CI",
			"jenkins":                "Jenkins",
			"circleci":               "CircleCI",
			"buildkite":              "Buildkite",
			"argocd":                "ArgoCD",
			"argo cd":               "ArgoCD",
			"spinnaker":             "Spinnaker",
			"teamcity":              "TeamCity",
			"bitbucket pipelines":   "Bitbucket Pipelines",
		},
	},
	{
		Name: "Containers",
		Pattern: regexp.MustCompile(`(?i)\b(` +
			`docker|kubernetes|k8s|helm|containerd|podman|microservices|` +
			`service mesh|istio|linkerd|serverless` +
			`)\b`),
		Canonical: map[string]string{
			"docker":        "Docker",
			"kubernetes":    "Kubernetes",
			"k8s":           "Kubernetes",
			"helm":          "Helm",
			"containerd":    "containerd",
			"podman":        "Podman",
			"microservices": "Microservices",
			"service mesh":  "Service Mesh",
			"istio":         "Istio",
			"linkerd":       "Linkerd",
			"serverless":    "Serverless",
		},
	},
	{
		Name: "Security",
		Pattern: regexp.MustCompile(`(?i)\b(` +
			`devsecops|cspm|wiz|prisma cloud|security hubs?|` +
			`least[ -]?privilege|zero trust|rbac|sso|mfa|` +
			`siem|soar|owasp|compliance|fedramp|iso ?27001|soc ?2|` +
			`scps?|iams?|permission boundar(?:y|ies)|service control polic(?:y|ies)|` +
			`vulnerability management|threat modeling|encryption` +
			`)\b`),
		Canonical: map[string]string{
			"devsecops":                "DevSecOps",
			"cspm":                     "CSPM",
			"wiz":                      "Wiz",
			"prisma cloud":             "Prisma Cloud",
			"security hub":             "AWS Security Hub",
			"security hubs":            "AWS Security Hub",
			"least privilege":          "Least Privilege",
			"least-privilege":          "Least Privilege",
			"zero trust":               "Zero Trust",
			"rbac":                     "RBAC",
			"sso":                      "SSO",
			"mfa":                      "MFA",
			"siem":                     "SIEM",
			"soar":                     "SOAR",
			"owasp":                    "OWASP",
			"compliance":               "Compliance",
			"fedramp":                  "FedRAMP",
			"iso27001":                 "ISO 27001",
			"iso 27001":                "ISO 27001",
			"soc2":                     "SOC 2",
			"soc 2":                    "SOC 2",
			"iam":                      "IAM",
			"iams":                     "IAM",
			"scp":                      "SCP",
			"scps":                     "SCP",
			"permission boundary":      "Permission Boundary",
			"permission boundaries":    "Permission Boundary",
			"service control policy":   "SCP",
			"service control policies": "SCP",
			"vulnerability management": "Vulnerability Management",
			"threat modeling":          "Threat Modeling",
			"encryption":               "Encryption",
		},
	},
	{
		Name: "Languages",
		Pattern: regexp.MustCompile(`(?i)\b(` +
			`python|golang|go lang|java\b|javascript|typescript|` +
			`c#|\.net|node\.js|nodejs|bash|powershell|shell scripting|` +
			`php|ruby|rust|kotlin|swift|scala` +
			`)\b`),
		Canonical: map[string]string{
			"python":          "Python",
			"golang":          "Go",
			"go lang":         "Go",
			"java":            "Java",
			"javascript":      "JavaScript",
			"typescript":      "TypeScript",
			"c#":              "C#",
			".net":            ".NET",
			"node.js":         "Node.js",
			"nodejs":          "Node.js",
			"bash":            "Bash",
			"powershell":      "PowerShell",
			"shell scripting": "Shell Scripting",
			"php":             "PHP",
			"ruby":            "Ruby",
			"rust":            "Rust",
			"kotlin":          "Kotlin",
			"swift":           "Swift",
			"scala":           "Scala",
		},
	},
	{
		Name: "Data/DB",
		Pattern: regexp.MustCompile(`(?i)\b(` +
			`postgresql|postgres|mysql|mariadb|mongodb|redis|` +
			`dynamodb|aurora|snowflake|bigquery|redshift|` +
			`elasticSearch|kafka|rabbitmq|graphql|etl|elt|airflow|dbt` +
			`)\b`),
		Canonical: map[string]string{
			"postgresql":    "PostgreSQL",
			"postgres":      "PostgreSQL",
			"mysql":         "MySQL",
			"mariadb":       "MariaDB",
			"mongodb":       "MongoDB",
			"redis":         "Redis",
			"dynamodb":      "DynamoDB",
			"aurora":        "Aurora",
			"snowflake":     "Snowflake",
			"bigquery":      "BigQuery",
			"redshift":      "Redshift",
			"elasticsearch": "Elasticsearch",
			"kafka":         "Kafka",
			"rabbitmq":      "RabbitMQ",
			"graphql":       "GraphQL",
			"etl":           "ETL",
			"elt":           "ELT",
			"airflow":       "Airflow",
			"dbt":           "dbt",
		},
	},
	{
		Name: "AI",
		Pattern: regexp.MustCompile(`(?i)\b(` +
			`chatgpt|claude|gemini|github copilot|cursor|` +
			`claude code|copilot|llm\b|prompt engineering|rag\b|` +
			`generative ai|genai|gen ai` +
			`)\b`),
		Canonical: map[string]string{
			"chatgpt":            "ChatGPT",
			"claude code":        "Claude Code",
			"claude":             "Claude",
			"gemini":             "Gemini",
			"github copilot":     "GitHub Copilot",
			"copilot":            "Copilot",
			"cursor":             "Cursor",
			"llm":                "LLM",
			"prompt engineering": "Prompt Engineering",
			"rag":                "RAG",
			"generative ai":      "Generative AI",
			"genai":              "GenAI",
			"gen ai":             "GenAI",
		},
	},
}

// DetectTags scans the (already HTML-stripped) text for known skill/tool
// keywords and returns a map of category → sorted unique tags. The returned
// map is always non-nil but may be empty.
//
// Detection is whole-word + case-insensitive; partial matches inside a
// larger token (e.g. "international" containing "intern") never trigger a
// tag. Within a category, each canonical tag is reported at most once even
// if multiple synonyms were matched.
func DetectTags(text string) map[string][]string {
	tags := make(map[string]map[string]struct{})

	for _, cat := range keywordCategories {
		matches := cat.Pattern.FindAllString(text, -1)
		if len(matches) == 0 {
			continue
		}
		seen := make(map[string]struct{}, len(matches))
		for _, m := range matches {
			key := strings.ToLower(strings.TrimSpace(m))
			canonical, ok := cat.Canonical[key]
			if !ok {
				canonical = strings.TrimSpace(m)
			}
			seen[canonical] = struct{}{}
		}
		if len(seen) > 0 {
			tags[cat.Name] = seen
		}
	}

	out := make(map[string][]string, len(tags))
	for cat, seen := range tags {
		list := make([]string, 0, len(seen))
		for tag := range seen {
			list = append(list, tag)
		}
		sort.Strings(list)
		out[cat] = list
	}
	return out
}
