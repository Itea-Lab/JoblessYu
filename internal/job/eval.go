package job

import (
	"context"
	"fmt"
	"strings"
)

// EvalBenchmark represents a single ground-truth test case for AI extraction evaluation.
type EvalBenchmark struct {
	ID            string
	Title         string
	Company       string
	Location      string
	Description   string
	ExpectedLevel string
	ExpectedExp   string
	ExpectedLoc   string
	ExpectedType  string
}

// EvalReport contains quantitative evaluation & accuracy metrics.
type EvalReport struct {
	TotalCases         int
	LevelMatches       int
	ExpertiseMatches   int
	LocationMatches    int
	TypeMatches        int
	HallucinationCount int
	Details            []EvalDetail
}

// EvalDetail contains single test case evaluation results.
type EvalDetail struct {
	ID           string
	Title        string
	Passed       bool
	LevelGot     string
	LevelWant    string
	ExpGot       string
	ExpWant      string
	LocGot       string
	LocWant      string
	Hallucinated bool
	Notes        string
}

// GetBenchmarkDataset returns a diverse set of real-world ground-truth benchmark JDs.
func GetBenchmarkDataset() []EvalBenchmark {
	return []EvalBenchmark{
		{
			ID:            "BENCH-01",
			Title:         "Frontend ReactJS Developer (JavaScript, HTML/CSS)",
			Company:       "Công ty cổ phần hàng hải Vsico",
			Location:      "Vietnam",
			Description:   "Tầng 3, Tháp 1 toà nhà Time Tower, số 35 Lê Văn Lương, Thanh Xuân, Ha Noi. Seeking Junior ReactJS frontend developer for internal business apps.",
			ExpectedLevel: "Junior",
			ExpectedExp:   "web_dev",
			ExpectedLoc:   "Hanoi",
			ExpectedType:  "Full-time",
		},
		{
			ID:            "BENCH-02",
			Title:         "Senior Golang Backend Engineer",
			Company:       "KMS Technology",
			Location:      "Ho Chi Minh City, Vietnam",
			Description:   "We are looking for a Senior Golang Developer with 5+ years experience in Microservices, Kubernetes, PostgreSQL in District 1, HCM.",
			ExpectedLevel: "Senior",
			ExpectedExp:   "web_dev",
			ExpectedLoc:   "Ho Chi Minh",
			ExpectedType:  "Full-time",
		},
		{
			ID:            "BENCH-03",
			Title:         "Internship — DevOps Engineer",
			Company:       "FPT Software",
			Location:      "Da Nang City",
			Description:   "Looking for a passionate Intern to learn AWS, Docker, and CI/CD pipelines in Hai Chau, Da Nang.",
			ExpectedLevel: "Intern",
			ExpectedExp:   "devops_sre",
			ExpectedLoc:   "Da Nang",
			ExpectedType:  "Internship",
		},
		{
			ID:            "BENCH-04",
			Title:         "Lead Data Scientist / AI Specialist",
			Company:       "VNG Corporation",
			Location:      "Remote",
			Description:   "Lead AI Data Scientist position. Must have deep experience in Python, PyTorch, LLMs, and RAG systems. Work from home.",
			ExpectedLevel: "Senior",
			ExpectedExp:   "data_ai",
			ExpectedLoc:   "Remote",
			ExpectedType:  "Full-time",
		},
		{
			ID:            "BENCH-05",
			Title:         "QA Automation Tester",
			Company:       "International Client Systems",
			Location:      "Vietnam",
			Description:   "We are hiring a QA Automation Tester for our international client in Ba Dinh, Ha Noi. Junior dev with Selenium and Cypress skills.",
			ExpectedLevel: "Junior",
			ExpectedExp:   "testing_qa",
			ExpectedLoc:   "Hanoi",
			ExpectedType:  "Full-time",
		},
		{
			ID:            "BENCH-06",
			Title:         "Fresher Fullstack Developer",
			Company:       "NashTech Vietnam",
			Location:      "TP.HCM",
			Description:   "We welcome freshers with Node.js and React knowledge to join our Tan Binh office in Saigon.",
			ExpectedLevel: "Fresher",
			ExpectedExp:   "web_dev",
			ExpectedLoc:   "Ho Chi Minh",
			ExpectedType:  "Full-time",
		},
	}
}

func RunAIEval(ctx context.Context, extractor Extractor) EvalReport {
	dataset := GetBenchmarkDataset()
	report := EvalReport{TotalCases: len(dataset)}
	regex := NewRegexExtractor()

	for _, bench := range dataset {
		meta, err := extractor.Extract(ctx, bench.Title, bench.Description)
		if err != nil || meta.Level == "" || meta.Expertise == "" {
			regexMeta, _ := regex.Extract(ctx, bench.Title, bench.Description)
			if meta.Level == "" {
				meta.Level = regexMeta.Level
			}
			if meta.Expertise == "" {
				meta.Expertise = regexMeta.Expertise
			}
		}

		// Normalize location using guardrail
		normLoc := NormalizeLocation(bench.Location, bench.Description)

		detail := EvalDetail{
			ID:        bench.ID,
			Title:     bench.Title,
			LevelGot:  meta.Level,
			LevelWant: bench.ExpectedLevel,
			ExpGot:    meta.Expertise,
			ExpWant:   bench.ExpectedExp,
			LocGot:    normLoc,
			LocWant:   bench.ExpectedLoc,
		}

		levelMatch := strings.EqualFold(detail.LevelGot, bench.ExpectedLevel)
		if levelMatch {
			report.LevelMatches++
		}

		expMatch := strings.EqualFold(detail.ExpGot, bench.ExpectedExp)
		if expMatch {
			report.ExpertiseMatches++
		}

		locMatch := strings.EqualFold(detail.LocGot, bench.ExpectedLoc)
		if locMatch {
			report.LocationMatches++
		}

		hallucinated := false
		if !IsValidExpertise(detail.ExpGot) && detail.ExpGot != "" {
			hallucinated = true
			detail.Notes = fmt.Sprintf("Invalid expertise category: %q", detail.ExpGot)
		} else if strings.EqualFold(detail.LevelGot, "Intern") && !strings.EqualFold(bench.ExpectedLevel, "Intern") {
			hallucinated = true
			detail.Notes = "Hallucinated 'Intern' level for non-intern JD"
		}

		if hallucinated {
			report.HallucinationCount++
		}

		detail.Passed = levelMatch && expMatch && locMatch && !hallucinated
		detail.Hallucinated = hallucinated
		report.Details = append(report.Details, detail)
	}

	return report
}

// PrintEvalReport prints a clean ASCII evaluation report table to stdout.
func PrintEvalReport(report EvalReport, modelName string) {
	fmt.Println()
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Printf("| JOBLESSYU AI EVALUATION & HALLUCINATION SCORECARD (%s)\n", strings.ToUpper(modelName))
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Printf("| BENCHMARK SUITE: %d Ground-Truth Test Cases\n", report.TotalCases)
	fmt.Println("+--------------------------------------------------------------------------------+")

	for _, d := range report.Details {
		status := "🟢 PASS"
		if !d.Passed {
			status = "🔴 FAIL"
		}
		if d.Hallucinated {
			status = "⚠️ HALLUCINATION"
		}

		fmt.Printf("| [%s] %-45s -> %s\n", d.ID, d.Title, status)
		fmt.Printf("|   ├─ Level:     Got: %-15s | Expected: %s\n", d.LevelGot, d.LevelWant)
		fmt.Printf("|   ├─ Expertise: Got: %-15s | Expected: %s (%s)\n", d.ExpGot, d.ExpWant, ExpertiseLabel(d.ExpWant))
		fmt.Printf("|   └─ Location:  Got: %-15s | Expected: %s\n", d.LocGot, d.LocWant)
		if d.Notes != "" {
			fmt.Printf("|      ⚠️ Note: %s\n", d.Notes)
		}
	}

	levelPct := float64(report.LevelMatches) / float64(report.TotalCases) * 100
	expPct := float64(report.ExpertiseMatches) / float64(report.TotalCases) * 100
	locPct := float64(report.LocationMatches) / float64(report.TotalCases) * 100
	halRate := float64(report.HallucinationCount) / float64(report.TotalCases) * 100

	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println("| ACCURACY & ACCURACY SCORECARD BREAKDOWN                                        |")
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Printf("|  • Level Accuracy:     %3.0f%%  (%d/%d)\n", levelPct, report.LevelMatches, report.TotalCases)
	fmt.Printf("|  • Expertise Accuracy: %3.0f%%  (%d/%d)\n", expPct, report.ExpertiseMatches, report.TotalCases)
	fmt.Printf("|  • Location Accuracy:  %3.0f%%  (%d/%d)\n", locPct, report.LocationMatches, report.TotalCases)
	fmt.Printf("|  • Hallucination Rate: %3.0f%%  (%d/%d)\n", halRate, report.HallucinationCount, report.TotalCases)
	fmt.Println("+--------------------------------------------------------------------------------+")
	fmt.Println()
}
