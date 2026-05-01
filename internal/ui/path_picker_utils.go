package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/x/ansi"
)

func (m *pathPickerModel) updatePickerHeight() {
	if m.height <= 0 {
		m.picker.SetHeight(minPickerHeight)
		return
	}

	bodyRows := framePaneHeight(m.height)
	if bodyRows < minPickerHeight+parentRowCount {
		bodyRows = minPickerHeight + parentRowCount
	}

	overhead := m.browserOverheadRows()
	pickerRows := bodyRows - overhead
	if pickerRows < minPickerHeight {
		pickerRows = minPickerHeight
	}
	m.picker.SetHeight(pickerRows)
}

func (m pathPickerModel) browserOverheadRows() int {
	if m.sourceDir == "" {
		return 4
	}
	return 12 + m.targetPreviewLineCount()
}

func (m pathPickerModel) targetPreviewLineCount() int {
	if m.sourceBusy || m.sourceEvErr != "" || len(m.sourceCreateDirs) == 0 {
		return 1
	}
	lines := len(m.sourceCreateDirs)
	if lines > targetPreviewMaxDirs {
		lines = targetPreviewMaxDirs + 1
	}
	return lines
}

func (m *pathPickerModel) clampPaneScrolls() {
	height := m.paneHeight() - 3
	if height < 1 {
		height = 1
	}

	leftTotal := len(m.sourcePaneLines())
	maxLeft := leftTotal - height
	if maxLeft < 0 {
		maxLeft = 0
	}
	if m.leftScroll < 0 {
		m.leftScroll = 0
	}
	if m.leftScroll > maxLeft {
		m.leftScroll = maxLeft
	}

	rightTotal := len(m.analysisPaneLines())
	maxRight := rightTotal - height
	if maxRight < 0 {
		maxRight = 0
	}
	if m.rightScroll < 0 {
		m.rightScroll = 0
	}
	if m.rightScroll > maxRight {
		m.rightScroll = maxRight
	}
}

func (m pathPickerModel) sourcePaneLines() []string {
	label := styleLabel()
	value := styleValue()
	muted := styleMuted()

	statsLine := "Stats: loading..."
	if m.sourceErr != "" {
		statsLine = "Stats: unavailable (" + m.sourceErr + ")"
	} else if m.sourceStat.dir != "" {
		statsLine = fmt.Sprintf(
			"Stats: entries=%d | files(recursive)=%d | size(recursive)=%s",
			m.sourceStat.directEntries,
			m.sourceStat.recursiveFiles,
			formatBytes(m.sourceStat.recursiveSize),
		)
	}
	eventLine := m.sourceLine
	if m.sourceBusy {
		eventLine = "Events: analyzing source..."
	}
	if m.sourceEvErr != "" {
		eventLine = "Events: unavailable (" + m.sourceEvErr + ")"
	}

	lines := []string{
		label.Render("Step 1 - Source Directory"),
		value.Render(m.sourceDir),
		"",
		value.Render(statsLine),
	}
	if len(m.sourceStat.previewEntries) > 0 {
		lines = append(lines, muted.Render("Preview: "+strings.Join(m.sourceStat.previewEntries, ", ")))
	}
	lines = append(lines, "")
	lines = append(lines, value.Render(eventLine))
	if len(m.sourceSegs) > 0 {
		lines = append(lines, "")
		lines = append(lines, label.Render("Segment Preview"))
		for _, seg := range m.sourceSegs {
			lines = append(lines, muted.Render("- "+seg))
		}
	}
	return lines
}

func (m pathPickerModel) analysisPaneLines() []string {
	label := styleLabel()
	value := styleValue()
	muted := styleMuted()
	confirm := styleAccent()

	lines := []string{
		label.Render("Step 1 - Source Directory"),
		"",
		confirm.Render(confirmActionLabel),
		"",
		"",
		value.Render(m.selectedDirPath()),
		muted.Render("Default: " + m.defaultDir),
		"",
		value.Render(m.statsLine()),
		muted.Render(m.previewLine()),
		"",
		value.Render(m.eventStatsLine()),
		muted.Render("Press 'e' to refresh event analysis."),
	}
	if len(m.eventSegs) > 0 {
		lines = append(lines, "")
		lines = append(lines, label.Render("Segment Preview"))
		for _, seg := range m.eventSegs {
			lines = append(lines, muted.Render("- "+seg))
		}
	}
	return lines
}

func (m *pathPickerModel) syncSelectionOnKey(key string) {
	if len(m.entries) == 0 {
		m.entryIdx = 0
		return
	}
	last := len(m.entries) - 1
	pageSize := m.picker.Height
	if pageSize <= 0 {
		pageSize = minPickerHeight
	}
	switch key {
	case "down", "j", "ctrl+n":
		m.entryIdx++
	case "up", "k", "ctrl+p":
		m.entryIdx--
	case "J", "pgdown":
		m.entryIdx += pageSize
	case "K", "pgup":
		m.entryIdx -= pageSize
	case "g":
		m.entryIdx = 0
	case "G":
		m.entryIdx = last
	}
	m.entryIdx = clampIndex(m.entryIdx, len(m.entries))
}

func listVisibleEntries(dir string, showHidden bool) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	filtered := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if !showHidden {
			hidden, _ := filepicker.IsHidden(entry.Name())
			if hidden {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].IsDir() == filtered[j].IsDir() {
			return filtered[i].Name() < filtered[j].Name()
		}
		return filtered[i].IsDir()
	})
	return filtered, nil
}

func clampIndex(idx, size int) int {
	if size <= 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= size {
		return size - 1
	}
	return idx
}

func clampLine(s string, width int) string {
	if width <= 0 {
		return s
	}
	if len([]rune(s)) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	r := []rune(s)
	return string(r[:width-1]) + "…"
}

func clampPickerView(view string, width, maxLines int) string {
	if width <= 0 {
		width = 1
	}
	if width > 1 {
		width--
	}
	if maxLines < 1 {
		maxLines = 1
	}
	lines := strings.Split(view, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	for i, line := range lines {
		lines[i] = ansi.TruncateWc(strings.TrimRight(line, " \t"), width, "…")
	}
	return strings.Join(lines, "\n")
}

func collectDirStats(dir string) (dirStats, error) {
	stats := dirStats{
		dir:            dir,
		previewEntries: make([]string, 0, statsMaxPreviewEntries),
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return stats, err
	}
	stats.directEntries = len(entries)
	for i, entry := range entries {
		if i >= statsMaxPreviewEntries {
			break
		}
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		stats.previewEntries = append(stats.previewEntries, name)
	}

	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		stats.recursiveFiles++
		stats.recursiveSize += info.Size()
		return nil
	})
	if walkErr != nil {
		return stats, walkErr
	}
	return stats, nil
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
