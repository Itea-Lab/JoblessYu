package bot

import (
	"testing"
	"time"
)

func TestBuildStatusCardEmbed_OnlineState(t *testing.T) {
	details := StatusDetails{
		ActiveJobs:     342,
		LastScrapeTime: time.Date(2026, 8, 10, 5, 0, 0, 0, time.UTC),
		NextScrapeTime: time.Date(2026, 8, 11, 5, 0, 0, 0, time.UTC),
		RetentionDays:  30,
		Version:        "v1.2.0",
	}

	embed := BuildStatusCardEmbed(true, details)

	if embed == nil {
		t.Fatal("expected non-nil embed")
	}
	if embed.Title != "🟢 ONLINE" {
		t.Errorf("unexpected title: %s", embed.Title)
	}
	if embed.Color != 0x10B981 {
		t.Errorf("expected emerald color 0x10B981, got 0x%X", embed.Color)
	}
	if len(embed.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(embed.Fields))
	}
}

func TestBuildStatusCardEmbed_OfflineState(t *testing.T) {
	details := StatusDetails{
		ActiveJobs:    100,
		RetentionDays: 30,
		Version:       "v1.2.0",
	}

	embed := BuildStatusCardEmbed(false, details)

	if embed == nil {
		t.Fatal("expected non-nil embed")
	}
	if embed.Title != "🔴 OFFLINE" {
		t.Errorf("unexpected title: %s", embed.Title)
	}
	if embed.Color != 0xEF4444 {
		t.Errorf("expected crimson color 0xEF4444, got 0x%X", embed.Color)
	}
}

func TestBuildDailyAnnouncementEmbed(t *testing.T) {
	summary := DailyScrapeSummary{
		RunTime:         time.Date(2026, 8, 10, 5, 0, 0, 0, time.UTC),
		TotalDuration:   12 * time.Minute,
		JobspyCount:     80,
		CollyCount:      40,
		InsertedCount:   35,
		MergedCount:     15,
		EnrichedCount:   50,
		TotalActiveJobs: 342,
	}

	embed := BuildDailyAnnouncementEmbed(summary)

	if embed == nil {
		t.Fatal("expected non-nil embed")
	}
	if embed.Color != 0xF59E0B {
		t.Errorf("expected amber color 0xF59E0B, got 0x%X", embed.Color)
	}
	if embed.Title == "" {
		t.Error("expected non-empty title")
	}
	if len(embed.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(embed.Fields))
	}
}
