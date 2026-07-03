package domain

import "testing"

func TestLevelRulesSingleSource(t *testing.T) {
	// Guards the "single source of truth" contract: the four canonical
	// levels must all be registered, each with a non-empty WordPattern and
	// YearPattern. If a future edit drops one or leaves a pattern empty,
	// repository-derived maps silently miss that level — this test catches it.
	//
	// NoExpPattern is intentionally optional — only Intern has one today —
	// so we assert the Intern-specific contract rather than "all non-empty".
	want := map[string]bool{"Intern": true, "Fresher": true, "Junior": true, "Senior": true}
	got := make(map[string]bool, len(LevelRules))
	internRule := ""
	for _, r := range LevelRules {
		got[r.Name] = true
		if r.WordPattern == "" {
			t.Errorf("LevelRule %q has empty WordPattern — repository & service regex derivation breaks", r.Name)
		}
		if r.YearPattern == "" {
			t.Errorf("LevelRule %q has empty YearPattern — repository year-of-experience fallback breaks", r.Name)
		}
		if r.Name == "Intern" {
			internRule = r.NoExpPattern
		}
	}
	for lvl := range want {
		if !got[lvl] {
			t.Errorf("LevelRules missing %q — single source of truth is incomplete", lvl)
		}
	}
	if internRule == "" {
		t.Error(`LevelRule "Intern" has empty NoExpPattern — "no experience required" JDs silently filtered at the DB stage under include_unknown=false`)
	}
}

func TestNoExpAlternationForRegex(t *testing.T) {
	got := NoExpAlternationForRegex()
	if got == "" {
		t.Fatalf("NoExpAlternationForRegex() = empty; at least Intern must contribute a no-exp phrase")
	}
	// The current canonical Intern no-exp source. Pinning it catches
	// accidental edits to the Intern NoExpPattern without a matching update
	// to consumers (service noExpRe is byte-identical to this source
	// wrapped in word boundaries).
	want := `(no|zero)\s+(experience|exp)`
	if got != want {
		t.Errorf("NoExpAlternationForRegex() = %q, want %q", got, want)
	}
}

func TestLevelUnknownConstant(t *testing.T) {
	if LevelUnknown != "Unknown" {
		t.Errorf("LevelUnknown = %q, want %q", LevelUnknown, "Unknown")
	}
	if LevelUnknown == "" {
		t.Error("LevelUnknown must not be the empty string — distinct from DB's 'not yet classified' state")
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
			if got := FindLevelWord(c.text); got != c.want {
				t.Errorf("FindLevelWord(%q) = %q, want %q", c.text, got, c.want)
			}
		})
	}
}

// TestFindLevelWord_InternalNotIntern is a focused regression test for the
// original "international → Intern" misclassification that motivated the
// refactoring to a single source of truth.
func TestFindLevelWord_InternalNotIntern(t *testing.T) {
	if got := FindLevelWord("International team, internal tools, no interns here."); got == "Intern" {
		t.Errorf("regression: got Intern from international/internal; got=%q", got)
	}
}
