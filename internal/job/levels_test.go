package job

import "testing"

// Data-layer tests. The Go regex engine that consumes LevelRules lives in
// regex.go (private); its tests live in regex_test.go.
// This file guards the data contract only.

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
			t.Errorf("LevelRule %q has empty WordPattern — repository & ai regex derivation breaks", r.Name)
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

func TestLevelUnknownConstant(t *testing.T) {
	if LevelUnknown != "Unknown" {
		t.Errorf("LevelUnknown = %q, want %q", LevelUnknown, "Unknown")
	}
	if LevelUnknown == "" {
		t.Error("LevelUnknown must not be the empty string — distinct from DB's 'not yet classified' state")
	}
}
