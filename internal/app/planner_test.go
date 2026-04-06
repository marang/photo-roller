package app

import (
	"testing"
	"time"
)

func TestSplitByTimeGap(t *testing.T) {
	base := time.Date(2026, 3, 29, 8, 0, 0, 0, time.UTC)
	assets := []assetMeta{
		{Path: "a.jpg", CaptureTime: base},
		{Path: "b.jpg", CaptureTime: base.Add(20 * time.Minute)},
		{Path: "c.jpg", CaptureTime: base.Add(3 * time.Hour)},
	}
	segments := splitByTimeGap(assets, 90*time.Minute)
	if len(segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segments))
	}
	if len(segments[0]) != 2 || len(segments[1]) != 1 {
		t.Fatalf("unexpected segment lengths: %d / %d", len(segments[0]), len(segments[1]))
	}
}

func TestBuildDaySummary(t *testing.T) {
	got := buildDaySummary([]string{"hike", "gasthaus", "hike"})
	if got != "hike_and_gasthaus" {
		t.Fatalf("buildDaySummary returned %q", got)
	}
}
