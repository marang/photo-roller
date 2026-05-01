package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/marang/photo-roller/internal/config"
)

func TestResolveTargetName(t *testing.T) {
	name, err := resolveTargetName("A.JPG", 1, config.CollisionModeSuffix)
	if err != nil {
		t.Fatal(err)
	}
	if name != "A.JPG" {
		t.Fatalf("unexpected name %q", name)
	}

	name, err = resolveTargetName("A.JPG", 2, config.CollisionModeSuffix)
	if err != nil {
		t.Fatal(err)
	}
	if name != "A_2.JPG" {
		t.Fatalf("unexpected suffix name %q", name)
	}

	if _, err = resolveTargetName("A.JPG", 2, config.CollisionModeFail); err == nil {
		t.Fatal("expected collision error in fail mode")
	}
}

func TestApplyPlanSkipsExistingTargetsWithoutDeletionEligibility(t *testing.T) {
	source := t.TempDir()
	target := t.TempDir()
	src := filepath.Join(source, "DSCF0001.JPG")
	if err := os.WriteFile(src, []byte("photo-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ScanResult{
		TotalFiles: 1,
		Days: []DayPlan{
			{
				FolderName: "2026-05-01_no_gps",
				Segments: []SegmentPlan{
					{
						FolderName: "2026-05-01_1000-1000_no_gps",
						Files:      []string{src},
					},
				},
			},
		},
	}
	dstDir := filepath.Join(target, "2026-05-01_no_gps", "2026-05-01_1000-1000_no_gps")
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, "DSCF0001.JPG"), []byte("photo-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	withFakeRclone(t)
	events := make(chan ProgressEvent, 8)
	summary, err := ApplyPlanWithSummary(context.Background(), config.Config{
		Target:        target,
		CollisionMode: config.CollisionModeSuffix,
	}, result, events)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Copied) != 0 {
		t.Fatalf("expected no copied files, got %d", len(summary.Copied))
	}
	if len(summary.SkippedExisting) != 1 {
		t.Fatalf("expected one skipped existing file, got %d", len(summary.SkippedExisting))
	}

	deleteSummary, err := DeleteVerifiedSources(summary.Copied, source)
	if err != nil {
		t.Fatal(err)
	}
	if deleteSummary.DeletedFiles != 0 {
		t.Fatalf("expected no source deletion for skipped existing target, got %d", deleteSummary.DeletedFiles)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source should still exist: %v", err)
	}
}

func TestSendProgressEventHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := sendProgressEvent(ctx, make(chan ProgressEvent), ProgressEvent{Kind: EventDone})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestDeleteVerifiedSourcesIgnoresFilesOutsideSourceRoot(t *testing.T) {
	sourceRoot := t.TempDir()
	outside := t.TempDir()
	target := t.TempDir()
	src := filepath.Join(outside, "outside.JPG")
	dst := filepath.Join(target, "outside.JPG")
	if err := os.WriteFile(src, []byte("photo-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("photo-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	summary, err := DeleteVerifiedSources([]PlannedCopy{{Source: src, Target: dst}}, sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	if summary.DeletedFiles != 0 {
		t.Fatalf("expected no deletion outside source root, got %d", summary.DeletedFiles)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("outside source should still exist: %v", err)
	}
}

func withFakeRclone(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	script := filepath.Join(binDir, "rclone")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncp \"$2\" \"$3\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
