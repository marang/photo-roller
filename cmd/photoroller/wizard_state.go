package photoroller

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/marang/photo-roller/internal/app"
	"github.com/marang/photo-roller/internal/config"
)

const (
	maxTopEventTypes        = 6
	maxPreflightDays        = 5
	maxSourcePreviewEntries = 6
)

type wizardState struct {
	cfg        *config.Config
	result     app.ScanResult
	collisions []app.CollisionStat
	confirmed  []confirmedStep
}

type confirmedStep struct {
	title    string
	snapshot string
}

func newWizardState(cfg *config.Config, result app.ScanResult, collisions []app.CollisionStat) wizardState {
	return wizardState{
		cfg:        cfg,
		result:     result,
		collisions: collisions,
	}
}

func (w *wizardState) LeftSummary() string {
	return w.buildLeftSummary()
}

func (w *wizardState) AddConfirmedStep(stepTitle, snapshot string) {
	for i := range w.confirmed {
		if w.confirmed[i].title == stepTitle {
			w.confirmed[i].snapshot = snapshot
			return
		}
	}
	w.confirmed = append(w.confirmed, confirmedStep{
		title:    stepTitle,
		snapshot: snapshot,
	})
}

func (w *wizardState) PreflightSummary() string {
	eventStats := app.BuildEventStats(w.result)

	lines := []string{
		fmt.Sprintf("Source: %s", w.cfg.Source),
		fmt.Sprintf("Target: %s", w.cfg.Target),
		fmt.Sprintf("Days: %d | Segments: %d | Files: %d", len(w.result.Days), eventStats.TotalSegments, w.result.TotalFiles),
		fmt.Sprintf("Distinct event types: %d", eventStats.DistinctTypes),
		fmt.Sprintf("Geocode requests: %d | Collisions: %d | Warnings: %d", w.result.GeocodeRequests, len(w.collisions), len(w.result.Warnings)),
		"Top event types:",
	}

	maxEventTypes := minInt(len(eventStats.ByType), maxTopEventTypes)
	for i := 0; i < maxEventTypes; i++ {
		e := eventStats.ByType[i]
		lines = append(lines, fmt.Sprintf("- %s: %d", e.Label, e.Count))
	}
	if len(eventStats.ByType) == 0 {
		lines = append(lines, "- none")
	}

	lines = append(lines, "Preview (first days):")

	maxDays := minInt(len(w.result.Days), maxPreflightDays)
	for i := 0; i < maxDays; i++ {
		day := w.result.Days[i]
		lines = append(lines, fmt.Sprintf("- %s (%d files, %d segments)", day.FolderName, day.FilesCount, len(day.Segments)))
	}
	if len(w.result.Days) > maxDays {
		lines = append(lines, fmt.Sprintf("- ... %d more days", len(w.result.Days)-maxDays))
	}

	return strings.Join(lines, "\n")
}

func (w *wizardState) ExecuteSummary() string {
	lines := []string{
		fmt.Sprintf("Source: %s", w.cfg.Source),
		fmt.Sprintf("Target: %s", w.cfg.Target),
		fmt.Sprintf("Collision mode: %s", w.cfg.CollisionMode),
		fmt.Sprintf("Files: %d | Days: %d", w.result.TotalFiles, len(w.result.Days)),
		"",
		"Next step:",
		"- apply copy with rclone",
		"- verify destination size for each planned file",
		"- optional cleanup of verified source files",
	}
	return strings.Join(lines, "\n")
}

func (w *wizardState) VerifySummary(verify app.VerifySummary, applyErr error) string {
	lines := []string{
		fmt.Sprintf("Source: %s", w.cfg.Source),
		fmt.Sprintf("Target: %s", w.cfg.Target),
		"",
		"Verification:",
		fmt.Sprintf("- planned files: %d", verify.Planned),
		fmt.Sprintf("- verified (size match): %d", verify.Verified),
		fmt.Sprintf("- missing targets: %d", verify.MissingTargets),
		fmt.Sprintf("- size mismatches: %d", verify.SizeMismatches),
		fmt.Sprintf("- source missing/stat errors: %d/%d", verify.SourceMissing, verify.SourceStatError),
		fmt.Sprintf("- target stat errors: %d", verify.TargetStatError),
	}
	if applyErr != nil {
		lines = append(lines, "", "Apply status: finished with copy warnings/errors")
	}
	if verify.FailedCount() > 0 {
		lines = append(lines, "Only verified files are eligible for deletion.")
	}
	return strings.Join(lines, "\n")
}

func (w *wizardState) buildBaseLeft() []string {
	lines := []string{
		"Step 1 - Source Directory",
		w.cfg.Source,
		"",
	}
	lines = append(lines, buildSourceDetailsLines(w.cfg.Source, w.result)...)
	lines = append(lines, "")
	lines = append(lines, "Step 2 - Target Directory")
	lines = append(lines, w.cfg.Target)
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Collision mode: %s", w.cfg.CollisionMode))
	lines = append(lines, "")
	lines = append(lines, "Create Preview")
	lines = append(lines, buildCreatePreviewLines(w.result, 0)...)
	return lines
}

func (w *wizardState) buildLeftSummary() string {
	lines := make([]string, 0, 64)
	lines = append(lines, w.buildBaseLeft()...)
	if len(w.confirmed) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Confirmed Steps")
		for _, step := range w.confirmed {
			lines = append(lines, step.title)
			lines = append(lines, expandSnapshot(step.snapshot)...)
			lines = append(lines, "")
		}
	}
	return strings.Join(lines, "\n")
}

func buildSourceDetailsLines(source string, result app.ScanResult) []string {
	entries, recursiveFiles, recursiveSize, preview := collectSourceStats(source)
	eventStats := app.BuildEventStats(result)
	parts := make([]string, 0, 3)
	max := 3
	if len(eventStats.ByType) < max {
		max = len(eventStats.ByType)
	}
	for i := 0; i < max; i++ {
		e := eventStats.ByType[i]
		parts = append(parts, fmt.Sprintf("%s=%d", e.Label, e.Count))
	}
	if len(parts) == 0 {
		parts = append(parts, "none")
	}

	lines := []string{
		fmt.Sprintf("Stats: entries=%d | files(recursive)=%d | size(recursive)=%s", entries, recursiveFiles, formatBytes(recursiveSize)),
	}
	if len(preview) > 0 {
		lines = append(lines, "Preview: "+strings.Join(preview, ", "))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Events: segments=%d types=%d top[%s]", eventStats.TotalSegments, eventStats.DistinctTypes, strings.Join(parts, ", ")))
	lines = append(lines, "Segment Preview")
	for _, day := range result.Days {
		for _, seg := range day.Segments {
			lines = append(lines, fmt.Sprintf("- %s/%s", day.FolderName, seg.FolderName))
		}
	}
	return lines
}

func collectSourceStats(source string) (int, int, int64, []string) {
	entries, err := os.ReadDir(source)
	if err != nil {
		return 0, 0, 0, nil
	}
	preview := make([]string, 0, maxSourcePreviewEntries)
	for i, entry := range entries {
		if i >= maxSourcePreviewEntries {
			break
		}
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		preview = append(preview, name)
	}
	sort.Strings(preview)

	recursiveFiles := 0
	var recursiveSize int64
	_ = filepath.WalkDir(source, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil || d.IsDir() {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		recursiveFiles++
		recursiveSize += info.Size()
		return nil
	})
	return len(entries), recursiveFiles, recursiveSize, preview
}

func formatBytes(size int64) string {
	const (
		kib = int64(1024)
		mib = kib * 1024
		gib = mib * 1024
		tib = gib * 1024
	)
	switch {
	case size >= tib:
		return fmt.Sprintf("%.2f TiB", float64(size)/float64(tib))
	case size >= gib:
		return fmt.Sprintf("%.2f GiB", float64(size)/float64(gib))
	case size >= mib:
		return fmt.Sprintf("%.2f MiB", float64(size)/float64(mib))
	case size >= kib:
		return fmt.Sprintf("%.2f KiB", float64(size)/float64(kib))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func buildCreatePreviewLines(result app.ScanResult, max int) []string {
	capHint := max + 1
	if max <= 0 {
		capHint = 16
	}
	out := make([]string, 0, capHint)
	count := 0
	for _, day := range result.Days {
		for _, seg := range day.Segments {
			if max > 0 && count >= max {
				break
			}
			out = append(out, fmt.Sprintf("- %s/%s (%d files)", day.FolderName, seg.FolderName, len(seg.Files)))
			count++
		}
		if max > 0 && count >= max {
			break
		}
	}
	if max > 0 {
		total := 0
		for _, day := range result.Days {
			total += len(day.Segments)
		}
		if total > count {
			out = append(out, fmt.Sprintf("- ... +%d more", total-count))
		}
	}
	if len(out) == 0 {
		out = append(out, "- no segments")
	}
	return out
}

func expandSnapshot(snapshot string) []string {
	lines := strings.Split(snapshot, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		out = append(out, "- "+trim)
	}
	if len(out) == 0 {
		out = append(out, "- (empty)")
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
