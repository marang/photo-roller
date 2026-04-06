package photoroller

import (
	"strings"
	"testing"
	"time"

	"github.com/marang/photo-roller/internal/app"
	"github.com/marang/photo-roller/internal/config"
)

func TestWizardState_LeftSummaryKeepsBaseSteps(t *testing.T) {
	cfg := config.Config{
		Source:        "/src/DCIM",
		Target:        "/dst/albums",
		CollisionMode: config.CollisionModeSuffix,
	}
	result := sampleResult()
	state := newWizardState(&cfg, result, nil)

	left := state.LeftSummary()
	requireContains(t, left, "Step 1 - Source Directory")
	requireContains(t, left, "/src/DCIM")
	requireContains(t, left, "Step 2 - Target Directory")
	requireContains(t, left, "/dst/albums")
	requireBefore(t, left, "Step 1 - Source Directory", "Stats:")
	requireBefore(t, left, "Stats:", "Step 2 - Target Directory")

	state.AddConfirmedStep("Step 3 - Preflight Review", "line1\nline2\nline3\nline4\nline5\nline6\nline7")
	left2 := state.LeftSummary()
	requireContains(t, left2, "Step 1 - Source Directory")
	requireContains(t, left2, "Step 2 - Target Directory")
	requireContains(t, left2, "Confirmed Steps")
	requireContains(t, left2, "Step 3 - Preflight Review")
}

func TestExpandSnapshotKeepsAllNonEmptyLines(t *testing.T) {
	snapshot := strings.Join([]string{
		"a", "b", "c", "d", "e", "f", "g", "h",
	}, "\n")
	lines := expandSnapshot(snapshot)
	if len(lines) != 8 {
		t.Fatalf("expected 8 lines, got %d: %#v", len(lines), lines)
	}
	if lines[0] != "- a" || lines[7] != "- h" {
		t.Fatalf("unexpected expanded lines: %#v", lines)
	}
}

func sampleResult() app.ScanResult {
	return app.ScanResult{
		TotalFiles: 3,
		Days: []app.DayPlan{
			{
				Date:       "2025-05-01",
				FolderName: "2025-05-01_no_gps",
				FilesCount: 3,
				Segments: []app.SegmentPlan{
					{
						Start:      time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC),
						End:        time.Date(2025, 5, 1, 10, 30, 0, 0, time.UTC),
						Label:      "no_gps",
						FolderName: "2025-05-01_1000-1030_no_gps",
						Files:      []string{"a.jpg", "b.jpg", "c.jpg"},
					},
				},
			},
		},
	}
}

func requireContains(t *testing.T, s, want string) {
	t.Helper()
	if !strings.Contains(s, want) {
		t.Fatalf("expected %q to contain %q", s, want)
	}
}

func requireBefore(t *testing.T, s, a, b string) {
	t.Helper()
	ai := strings.Index(s, a)
	bi := strings.Index(s, b)
	if ai < 0 || bi < 0 {
		t.Fatalf("expected both %q and %q in string", a, b)
	}
	if ai >= bi {
		t.Fatalf("expected %q to appear before %q", a, b)
	}
}
