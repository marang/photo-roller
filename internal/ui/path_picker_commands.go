package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/marang/photo-roller/internal/app"
	"github.com/marang/photo-roller/internal/config"
)

func (m *pathPickerModel) reloadStatsCmd() tea.Cmd {
	m.statToken++
	m.loading = true
	m.statsErr = ""
	return m.loadStatsCmd()
}

func (m *pathPickerModel) reloadBrowserEntriesCmd() tea.Cmd {
	m.browserTok++
	m.entryErr = ""
	return m.loadBrowserEntriesCmd()
}

func (m pathPickerModel) loadBrowserEntriesCmd() tea.Cmd {
	token := m.browserTok
	dir := m.picker.CurrentDirectory
	showHidden := m.picker.ShowHidden
	return func() tea.Msg {
		entries, err := listVisibleEntries(dir, showHidden)
		return browserEntriesMsg{
			token:   token,
			dir:     dir,
			entries: entries,
			err:     err,
		}
	}
}

func (m pathPickerModel) loadStatsCmd() tea.Cmd {
	token := m.statToken
	dir := m.picker.CurrentDirectory
	return func() tea.Msg {
		stats, err := collectDirStats(dir)
		return dirStatsMsg{
			token: token,
			stats: stats,
			err:   err,
		}
	}
}

func (m *pathPickerModel) reloadEventCmd() tea.Cmd {
	dir := m.picker.CurrentDirectory
	if cached, ok := m.eventCache[dir]; ok {
		m.eventBusy = false
		m.eventErr = ""
		m.eventLine = cached.summary
		m.eventSegs = cached.segmentNames
		return nil
	}
	m.eventToken++
	m.eventBusy = true
	m.eventErr = ""
	return m.loadEventCmd()
}

func (m *pathPickerModel) applyCachedEventForDir(dir string) {
	if cached, ok := m.eventCache[dir]; ok {
		m.eventBusy = false
		m.eventErr = ""
		m.eventLine = cached.summary
		m.eventSegs = cached.segmentNames
		return
	}
	m.eventBusy = false
	m.eventErr = ""
	m.eventLine = "Events: press 'e' to analyze current directory"
	m.eventSegs = nil
}

func (m pathPickerModel) loadEventCmd() tea.Cmd {
	token := m.eventToken
	dir := m.picker.CurrentDirectory
	return func() tea.Msg {
		cfg := analysisConfig(dir)
		result, err := app.BuildPlan(context.Background(), cfg)
		if err != nil {
			return eventStatsMsg{token: token, err: err}
		}
		analysis := summarizeEventAnalysis(result)
		return eventStatsMsg{
			token:    token,
			dir:      dir,
			analysis: analysis,
		}
	}
}

func (m pathPickerModel) loadSourceStatsCmd() tea.Cmd {
	token := m.sourceTok
	dir := m.sourceDir
	return func() tea.Msg {
		stats, err := collectDirStats(dir)
		return sourceDirStatsMsg{
			token: token,
			stats: stats,
			err:   err,
		}
	}
}

func (m pathPickerModel) loadSourceEventCmd() tea.Cmd {
	token := m.sourceTok
	dir := m.sourceDir
	return func() tea.Msg {
		result, err := app.BuildPlan(context.Background(), analysisConfig(dir))
		if err != nil {
			return sourceEventStatsMsg{token: token, err: err}
		}
		return sourceEventStatsMsg{
			token:      token,
			analysis:   summarizeEventAnalysis(result),
			createDirs: buildTargetCreatePreview(result),
		}
	}
}

func analysisConfig(dir string) config.Config {
	return config.Config{
		Source:            dir,
		Target:            dir,
		Lang:              config.DefaultLang,
		Geocoder:          "none",
		GeocodeCache:      "",
		GeohashPrecision:  config.DefaultGeohashPrecision,
		SegmentGapMinutes: config.DefaultSegmentGapMinutes,
	}
}

func summarizeEventAnalysis(result app.ScanResult) eventAnalysis {
	es := app.BuildEventStats(result)
	parts := make([]string, 0, 3)
	max := 3
	if len(es.ByType) < max {
		max = len(es.ByType)
	}
	for i := 0; i < max; i++ {
		parts = append(parts, fmt.Sprintf("%s=%d", es.ByType[i].Label, es.ByType[i].Count))
	}
	if len(parts) == 0 {
		parts = append(parts, "none")
	}
	segmentNames := make([]string, 0, eventMaxSegments)
	for _, day := range result.Days {
		for _, seg := range day.Segments {
			if len(segmentNames) >= eventMaxSegments {
				break
			}
			segmentNames = append(segmentNames, day.FolderName+"/"+seg.FolderName)
		}
		if len(segmentNames) >= eventMaxSegments {
			break
		}
	}
	return eventAnalysis{
		summary:      fmt.Sprintf("Events: segments=%d types=%d top[%s]", es.TotalSegments, es.DistinctTypes, strings.Join(parts, ", ")),
		segmentNames: segmentNames,
	}
}

func buildTargetCreatePreview(result app.ScanResult) []string {
	out := make([]string, 0, len(result.Days)*2)
	for _, day := range result.Days {
		for _, seg := range day.Segments {
			out = append(out, fmt.Sprintf("%s/%s (%d files)", day.FolderName, seg.FolderName, len(seg.Files)))
		}
	}
	return out
}
