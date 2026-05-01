package ui

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (m pathPickerModel) View() string {
	width := normalizeScreenWidth(m.width)
	if m.sourceDir != "" {
		leftWidth, rightWidth := splitStandardPaneWidths(width)
		body := joinPanes(m.sourcePane(leftWidth, m.paneFocus.Is(0)), m.browserPane(rightWidth, m.paneFocus.Is(1)), defaultPaneGap)
		hint := styleMuted().Render(fmt.Sprintf("Keys: Tab switch pane, Enter open, / root, d default (%s), ↑/↓ or PgUp/PgDn scroll focused pane, Ctrl+S confirm, b back, Ctrl+D quit.", m.defaultDir))
		return renderScreenFrame(width, frameIntro(m.title, m.subtitle), body, hint)
	}

	body := m.analysisPane(width, true)
	hint := styleMuted().Render(fmt.Sprintf("Keys: Enter open, / root, d default (%s), e event-analyze, ↑/↓ or PgUp/PgDn navigate/scroll, Ctrl+S confirm, Ctrl+D quit.", m.defaultDir))
	return renderScreenFrame(width, frameIntro(m.title, m.subtitle), body, hint)
}

func (m pathPickerModel) browserPane(width int, active bool) string {
	paneWidth := width
	if paneWidth < defaultPaneMinWidth {
		paneWidth = defaultPaneMinWidth
	}
	textWidth := paneWidth - 6
	if textWidth < 10 {
		textWidth = 10
	}
	current := styleAccent()
	label := styleLabel()
	muted := styleMuted()
	confirm := styleAccent()
	entry := m.picker.Styles.Directory
	selectedLine := current.Render(clampLine("Selected: "+m.selectedDirPath(), textWidth))
	parentEntryRow := m.picker.Styles.Cursor.Render(" ") + " " + entry.Render("..")
	if m.focusOnParentRow() {
		parentEntryRow = m.picker.Styles.Cursor.Render(m.picker.Cursor) + m.picker.Styles.Selected.Render(" ..")
	}
	pickerView := m.picker.View()
	if m.focusOnParentRow() {
		inactive := m.picker
		inactive.Cursor = " "
		inactiveStyles := inactive.Styles
		inactiveStyles.Selected = styleDimValue()
		inactiveStyles.Cursor = styleDimValue()
		inactive.Styles = inactiveStyles
		pickerView = inactive.View()
	}

	if m.sourceDir != "" {
		previewLines := []string{label.Render("Preview")}
		if m.sourceBusy {
			previewLines = append(previewLines, muted.Render("Analyzing source plan..."))
		} else if m.sourceEvErr != "" {
			previewLines = append(previewLines, muted.Render(clampLine("Unavailable: "+m.sourceEvErr, textWidth)))
		} else if len(m.sourceCreateDirs) == 0 {
			previewLines = append(previewLines, muted.Render("No planned directories."))
		} else {
			maxPreview := targetPreviewMaxDirs
			if len(m.sourceCreateDirs) < maxPreview {
				maxPreview = len(m.sourceCreateDirs)
			}
			for i := 0; i < maxPreview; i++ {
				previewLines = append(previewLines, muted.Render(clampLine("- "+filepath.Join(m.selectedDirPath(), m.sourceCreateDirs[i]), textWidth)))
			}
			if len(m.sourceCreateDirs) > maxPreview {
				previewLines = append(previewLines, muted.Render(fmt.Sprintf("... +%d more", len(m.sourceCreateDirs)-maxPreview)))
			}
		}

		content := label.Render("Step 2 - Target Directory") + "\n\n" + confirm.Render(confirmActionLabel) + "\n\n\n" + selectedLine + "\n\n" + strings.Join(previewLines, "\n") + "\n\n" + label.Render("Browser") + "\n\n" + parentEntryRow + "\n" + clampPickerView(pickerView, textWidth, m.picker.Height)
		return renderPane(content, paneWidth, m.paneHeight(), active)
	}

	header := label.Render("Browser")
	content := header + "\n" + selectedLine + "\n\n" + parentEntryRow + "\n" + clampPickerView(pickerView, textWidth, m.picker.Height)
	return renderPane(content, paneWidth, m.paneHeight(), active)
}

func (m pathPickerModel) statsLine() string {
	if m.loading {
		return "Stats: loading..."
	}
	if m.statsErr != "" {
		return "Stats: unavailable (" + m.statsErr + ")"
	}
	return fmt.Sprintf(
		"Stats: entries=%d | files(recursive)=%d | size(recursive)=%s",
		m.stats.directEntries,
		m.stats.recursiveFiles,
		formatBytes(m.stats.recursiveSize),
	)
}

func (m pathPickerModel) previewLine() string {
	if len(m.stats.previewEntries) == 0 {
		return "Preview: -"
	}
	return "Preview: " + strings.Join(m.stats.previewEntries, ", ")
}

func (m pathPickerModel) eventStatsLine() string {
	if m.eventBusy {
		return "Events: analyzing..."
	}
	if m.eventErr != "" {
		return "Events: unavailable (" + m.eventErr + ")"
	}
	return m.eventLine
}

func (m pathPickerModel) analysisPane(width int, active bool) string {
	paneWidth := width
	if paneWidth < defaultPaneMinWidth {
		paneWidth = defaultPaneMinWidth
	}
	textWidth := paneWidth - 4
	if textWidth < 10 {
		textWidth = 10
	}
	label := styleLabel()
	value := styleValue()
	muted := styleMuted()
	confirm := styleAccent()
	entry := m.picker.Styles.Directory

	lines := []string{
		label.Render("Step 1 - Source Directory"),
		"",
		confirm.Render(confirmActionLabel),
		"",
		"",
		value.Render(clampLine("Selected: "+m.selectedDirPath(), textWidth)),
		muted.Render(clampLine("Default: "+m.defaultDir, textWidth)),
		"",
		label.Render("Browser"),
		"",
	}
	parentRow := entry.Render("..")
	if m.focusOnParentRow() {
		parentRow = m.picker.Styles.Cursor.Render(m.picker.Cursor) + m.picker.Styles.Selected.Render(" ..")
	}
	lines = append(lines, parentRow)
	pickerView := m.picker.View()
	if m.focusOnParentRow() {
		inactive := m.picker
		inactive.Cursor = " "
		inactiveStyles := inactive.Styles
		inactiveStyles.Selected = styleDimValue()
		inactiveStyles.Cursor = styleDimValue()
		inactive.Styles = inactiveStyles
		pickerView = inactive.View()
	}
	for _, row := range strings.Split(clampPickerView(pickerView, textWidth, m.picker.Height), "\n") {
		lines = append(lines, row)
	}
	lines = append(lines, "")
	lines = append(lines,
		value.Render(clampLine(m.statsLine(), textWidth)),
		muted.Render(clampLine(m.previewLine(), textWidth)),
		"",
		value.Render(clampLine(m.eventStatsLine(), textWidth)),
		muted.Render("Press 'e' to refresh event analysis."),
	)
	if len(m.eventSegs) > 0 {
		lines = append(lines, "")
		lines = append(lines, label.Render("Segment Preview"))
		for _, seg := range m.eventSegs {
			lines = append(lines, muted.Render(clampLine("- "+seg, textWidth)))
		}
	}
	return renderScrollablePane(lines, paneWidth, m.paneHeight(), m.rightScroll, active)
}

func (m pathPickerModel) sourcePane(width int, active bool) string {
	paneWidth := width
	if paneWidth < defaultPaneMinWidth {
		paneWidth = defaultPaneMinWidth
	}
	textWidth := paneWidth - 4
	if textWidth < 10 {
		textWidth = 10
	}
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
		value.Render(clampLine(m.sourceDir, textWidth)),
		"",
		value.Render(clampLine(statsLine, textWidth)),
	}
	if len(m.sourceStat.previewEntries) > 0 {
		lines = append(lines, muted.Render(clampLine("Preview: "+strings.Join(m.sourceStat.previewEntries, ", "), textWidth)))
	}
	lines = append(lines, "")
	lines = append(lines, value.Render(clampLine(eventLine, textWidth)))
	if len(m.sourceSegs) > 0 {
		lines = append(lines, "")
		lines = append(lines, label.Render("Segment Preview"))
		for _, seg := range m.sourceSegs {
			lines = append(lines, muted.Render(clampLine("- "+seg, textWidth)))
		}
	}
	return renderScrollablePane(lines, paneWidth, m.paneHeight(), m.leftScroll, active)
}

func (m pathPickerModel) paneHeight() int {
	return framePaneHeight(m.height)
}

func (m pathPickerModel) selectedDirPath() string {
	return m.picker.CurrentDirectory
}
