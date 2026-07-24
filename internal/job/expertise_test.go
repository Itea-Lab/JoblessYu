package job

import "testing"

func TestDetectExpertise(t *testing.T) {
	cases := []struct {
		name        string
		title       string
		description string
		want        string
	}{
		{"backend dev", "Backend Developer", "We need someone with Go experience", "web_dev"},
		{"devops", "DevOps Engineer", "Manage AWS infrastructure and CI/CD pipelines", "cloud_devops"},
		{"qa tester", "QA Engineer", "Manual and automation testing", "testing_qa"},
		{"data scientist", "Data Scientist", "Machine learning and NLP", "data_ai"},
		{"it support", "IT Support", "Helpdesk for internal users", "support_security"},
		{"project manager", "Project Manager", "Scrum and agile delivery", "management"},
		{"unknown", "Sales Representative", "Sell products to customers", ExpertiseUnknown},
		{"empty", "", "", ExpertiseUnknown},
		{"mobile ios", "iOS Developer", "Swift and Objective-C", "mobile_game"},
		{"embedded", "Firmware Engineer", "STM32 microcontroller development", "embedded_iot"},
		{"ux designer", "UX/UI Designer", "Figma and user research", "design_ux"},
		{"security", "Security Engineer", "Penetration testing and SOC", "support_security"},
		{"architect", "Solution Architect", "Enterprise architecture design", "architecture"},
		{"consultant", "IT Consultant", "Pre-sales technical consulting", "consulting_sales"},
		// False-positive regression tests (word-boundary fixes H1/H2)
		{"html not ml", "HTML Developer", "Build websites with HTML and CSS", "web_dev"},
		{"linux not ux", "Systems Engineer", "Work with Linux servers", ExpertiseUnknown},
		{"iraq not qa", "Field Engineer", "Previously worked in Iraq", ExpertiseUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DetectExpertise(c.title, c.description)
			if got != c.want {
				t.Errorf("DetectExpertise(%q, %q) = %q, want %q", c.title, c.description, got, c.want)
			}
		})
	}
}

func TestExpertiseLabel(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{"web_dev", "Web Development"},
		{"cloud_devops", "Cloud & DevOps"},
		{"", "Unknown"},
		{"unknown", "Unknown"},
		{"invalid_value", "Unknown"},
	}
	for _, c := range cases {
		got := ExpertiseLabel(c.value)
		if got != c.want {
			t.Errorf("ExpertiseLabel(%q) = %q, want %q", c.value, got, c.want)
		}
	}
}
