package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/marang/photo-roller/internal/app"
	"github.com/marang/photo-roller/internal/config"
)

const (
	statsMaxPreviewEntries = 6
	eventMaxSegments       = 8
	targetPreviewMaxDirs   = 10
	parentRowCount         = 1
	minPickerHeight        = 6
	focusParentRow         = 0
	focusPicker            = 1
	selectKey              = "ctrl+s"
	confirmActionLabel     = "> Confirm [Ctrl+S]"
)

var upNavigationKeys = map[string]struct{}{
	"up":     {},
	"k":      {},
	"ctrl+p": {},
}

var downNavigationKeys = map[string]struct{}{
	"down":   {},
	"j":      {},
	"ctrl+n": {},
}

var ErrBackToSource = errors.New("back to source picker")

type dirStats struct {
	dir            string
	directEntries  int
	recursiveFiles int
	recursiveSize  int64
	previewEntries []string
}

type dirStatsMsg struct {
	token int
	stats dirStats
	err   error
}

type eventStatsMsg struct {
	token    int
	dir      string
	analysis eventAnalysis
	err      error
}

type browserEntriesMsg struct {
	token   int
	dir     string
	entries []os.DirEntry
	err     error
}

type sourceDirStatsMsg struct {
	token int
	stats dirStats
	err   error
}

type sourceEventStatsMsg struct {
	token      int
	analysis   eventAnalysis
	createDirs []string
	err        error
}

type eventAnalysis struct {
	summary      string
	segmentNames []string
}

type pathPickerModel struct {
	title            string
	subtitle         string
	picker           filepicker.Model
	defaultDir       string
	statToken        int
	stats            dirStats
	statsErr         string
	loading          bool
	eventToken       int
	eventLine        string
	eventSegs        []string
	eventErr         string
	eventBusy        bool
	eventCache       map[string]eventAnalysis
	width            int
	height           int
	selected         string
	focus            FocusRing
	paneFocus        FocusRing
	leftScroll       int
	rightScroll      int
	browserTok       int
	entries          []os.DirEntry
	entryErr         string
	entryIdx         int
	sourceDir        string
	sourceBusy       bool
	sourceTok        int
	sourceStat       dirStats
	sourceErr        string
	sourceLine       string
	sourceSegs       []string
	sourceEvErr      string
	sourceCreateDirs []string
	back             bool
	cancel           bool
}

func (m pathPickerModel) focusOnParentRow() bool {
	return m.focus.Is(focusParentRow)
}

func (m pathPickerModel) focusOnPicker() bool {
	return m.focus.Is(focusPicker)
}

func newConfiguredDirPicker(startDir string) filepicker.Model {
	p := filepicker.New()
	p.CurrentDirectory = startDir
	p.DirAllowed = true
	p.FileAllowed = false
	p.ShowHidden = false
	p.ShowPermissions = false
	p.ShowSize = false
	p.AutoHeight = false
	p.SetHeight(minPickerHeight)

	keyMap := p.KeyMap
	keyMap.Select = key.NewBinding(key.WithKeys(selectKey), key.WithHelp(selectKey, "select dir"))
	p.KeyMap = keyMap

	styles := p.Styles
	styles.Directory = styleDimValue()
	styles.File = styleDimValue()
	styles.Symlink = styleDimValue()
	styles.Selected = styleAccent()
	styles.Cursor = styleAccent()
	p.Styles = styles
	return p
}

func RunDirectoryPicker(title, subtitle, initialDir string) (string, error) {
	startDir := initialDir
	if startDir == "" {
		startDir = "."
	}
	if stat, err := os.Stat(startDir); err != nil || !stat.IsDir() {
		startDir = "."
	}
	absStart, err := filepath.Abs(startDir)
	if err == nil {
		startDir = absStart
	}

	p := newConfiguredDirPicker(startDir)

	model := pathPickerModel{
		title:      title,
		subtitle:   subtitle,
		picker:     p,
		defaultDir: startDir,
		statToken:  1,
		loading:    true,
		eventLine:  "Events: press 'e' to analyze current directory",
		eventCache: map[string]eventAnalysis{},
		focus:      NewFocusRing(2),
		paneFocus:  NewFocusRing(2),
		browserTok: 1,
	}
	model.focus.Focus(focusPicker)
	model.paneFocus.Focus(0)

	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return "", err
	}
	out := finalModel.(pathPickerModel)
	if out.cancel {
		return "", fmt.Errorf("directory selection cancelled")
	}
	if out.selected == "" {
		return "", fmt.Errorf("no directory selected")
	}
	return out.selected, nil
}

func RunTargetDirectoryPicker(title, subtitle, initialDir, sourceDir string) (string, error) {
	startDir := initialDir
	if startDir == "" {
		startDir = "."
	}
	if stat, err := os.Stat(startDir); err != nil || !stat.IsDir() {
		startDir = "."
	}
	absStart, err := filepath.Abs(startDir)
	if err == nil {
		startDir = absStart
	}
	absSource, err := filepath.Abs(sourceDir)
	if err == nil {
		sourceDir = absSource
	}

	p := newConfiguredDirPicker(startDir)

	model := pathPickerModel{
		title:      title,
		subtitle:   subtitle,
		picker:     p,
		defaultDir: startDir,
		statToken:  1,
		loading:    true,
		eventLine:  "Events: press 'e' to analyze current directory",
		eventCache: map[string]eventAnalysis{},
		focus:      NewFocusRing(2),
		paneFocus:  NewFocusRing(2),
		browserTok: 1,
		sourceDir:  sourceDir,
		sourceBusy: true,
		sourceTok:  1,
		sourceLine: "Events: analyzing source...",
	}
	model.focus.Focus(focusPicker)
	model.paneFocus.Focus(1)

	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return "", err
	}
	out := finalModel.(pathPickerModel)
	if out.cancel {
		return "", fmt.Errorf("directory selection cancelled")
	}
	if out.back {
		return "", ErrBackToSource
	}
	if out.selected == "" {
		return "", fmt.Errorf("no directory selected")
	}
	return out.selected, nil
}

func (m pathPickerModel) Init() tea.Cmd {
	if m.sourceDir != "" {
		return tea.Batch(m.picker.Init(), m.loadBrowserEntriesCmd(), m.loadSourceStatsCmd(), m.loadSourceEventCmd())
	}
	return tea.Batch(m.picker.Init(), m.loadStatsCmd(), m.loadBrowserEntriesCmd())
}

func (m pathPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updatePickerHeight()
		m.clampPaneScrolls()
	case dirStatsMsg:
		if msg.token != m.statToken {
			return m, nil
		}
		m.loading = false
		if msg.err != nil {
			m.statsErr = msg.err.Error()
			return m, nil
		}
		m.statsErr = ""
		m.stats = msg.stats
		m.clampPaneScrolls()
		return m, nil
	case eventStatsMsg:
		if msg.token != m.eventToken {
			return m, nil
		}
		m.eventBusy = false
		if msg.err != nil {
			m.eventErr = msg.err.Error()
			return m, nil
		}
		m.eventErr = ""
		m.eventLine = msg.analysis.summary
		m.eventSegs = msg.analysis.segmentNames
		m.eventCache[msg.dir] = msg.analysis
		m.clampPaneScrolls()
		return m, nil
	case browserEntriesMsg:
		if msg.token != m.browserTok {
			return m, nil
		}
		if msg.err != nil {
			m.entryErr = msg.err.Error()
			m.entries = nil
			m.entryIdx = 0
			return m, nil
		}
		m.entryErr = ""
		m.entries = msg.entries
		m.entryIdx = clampIndex(m.entryIdx, len(m.entries))
		return m, nil
	case sourceDirStatsMsg:
		if msg.token != m.sourceTok {
			return m, nil
		}
		if msg.err != nil {
			m.sourceErr = msg.err.Error()
			return m, nil
		}
		m.sourceErr = ""
		m.sourceStat = msg.stats
		m.clampPaneScrolls()
		return m, nil
	case sourceEventStatsMsg:
		if msg.token != m.sourceTok {
			return m, nil
		}
		m.sourceBusy = false
		if msg.err != nil {
			m.sourceEvErr = msg.err.Error()
			return m, nil
		}
		m.sourceEvErr = ""
		m.sourceLine = msg.analysis.summary
		m.sourceSegs = msg.analysis.segmentNames
		m.sourceCreateDirs = msg.createDirs
		m.updatePickerHeight()
		m.clampPaneScrolls()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "tab":
			if m.sourceDir != "" {
				m.paneFocus.Next()
			}
			return m, nil
		case "up", "k", "pgup", "K":
			if m.sourceDir != "" && m.paneFocus.Is(0) {
				if msg.String() == "pgup" || msg.String() == "K" {
					m.leftScroll -= 5
				} else {
					m.leftScroll--
				}
				if m.leftScroll < 0 {
					m.leftScroll = 0
				}
				m.clampPaneScrolls()
				return m, nil
			}
		case "down", "j", "pgdown", "J":
			if m.sourceDir != "" && m.paneFocus.Is(0) {
				if msg.String() == "pgdown" || msg.String() == "J" {
					m.leftScroll += 5
				} else {
					m.leftScroll++
				}
				m.clampPaneScrolls()
				return m, nil
			}
		}
		if msg.String() == selectKey {
			m.selected = m.selectedDirPath()
			if m.selected == "" {
				m.selected = m.picker.CurrentDirectory
			}
			return m, tea.Quit
		}
		if m.sourceDir != "" && msg.String() == "b" {
			m.back = true
			return m, tea.Quit
		}

		if m.focusOnParentRow() {
			switch msg.String() {
			case "ctrl+d":
				m.cancel = true
				return m, tea.Quit
			case "enter", "right", "l":
				m.focus.Next()
				m.picker.CurrentDirectory = filepath.Dir(m.picker.CurrentDirectory)
				m.applyCachedEventForDir(m.picker.CurrentDirectory)
				return m, tea.Batch(m.picker.Init(), m.reloadStatsCmd(), m.reloadBrowserEntriesCmd())
			case "down", "j", "ctrl+n":
				m.focus.Next()
				return m, nil
			case "/", "d":
				// handled below using existing logic
			default:
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+d":
			m.cancel = true
			return m, tea.Quit
		case "/":
			m.focus.Focus(focusPicker)
			m.picker.CurrentDirectory = string(filepath.Separator)
			m.applyCachedEventForDir(m.picker.CurrentDirectory)
			return m, tea.Batch(m.picker.Init(), m.reloadStatsCmd(), m.reloadBrowserEntriesCmd())
		case "d":
			m.focus.Focus(focusPicker)
			m.picker.CurrentDirectory = m.defaultDir
			m.applyCachedEventForDir(m.picker.CurrentDirectory)
			return m, tea.Batch(m.picker.Init(), m.reloadStatsCmd(), m.reloadBrowserEntriesCmd())
		case "e":
			return m, m.reloadEventCmd()
		}
	}

	beforeDir := m.picker.CurrentDirectory
	beforeView := m.picker.View()
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	afterDir := m.picker.CurrentDirectory

	if didSelect, path := m.picker.DidSelectFile(msg); didSelect {
		m.selected = path
		return m, tea.Quit
	}

	if beforeDir != afterDir {
		m.focus.Focus(focusPicker)
		m.applyCachedEventForDir(afterDir)
		m.entryIdx = 0
		m.updatePickerHeight()
		return m, tea.Batch(cmd, m.reloadStatsCmd(), m.reloadBrowserEntriesCmd())
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.focusOnPicker() {
		m.syncSelectionOnKey(keyMsg.String())
		if _, isUp := upNavigationKeys[keyMsg.String()]; isUp && beforeView == m.picker.View() {
			m.focus.Prev()
			return m, nil
		}
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok && m.focusOnParentRow() {
		if _, isDown := downNavigationKeys[keyMsg.String()]; isDown {
			m.focus.Next()
			return m, nil
		}
	}

	return m, cmd
}

func (m pathPickerModel) View() string {
	width := m.width
	width = normalizeScreenWidth(width)
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
	leftWidth := width
	if leftWidth < defaultPaneMinWidth {
		leftWidth = defaultPaneMinWidth
	}
	textWidth := leftWidth - 6
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
		return renderPane(content, leftWidth, m.paneHeight(), active)
	}

	header := label.Render("Browser")
	content := header + "\n" + selectedLine + "\n\n" + parentEntryRow + "\n" + clampPickerView(pickerView, textWidth, m.picker.Height)
	return renderPane(content, leftWidth, m.paneHeight(), active)
}

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
		cfg := config.Config{
			Source:            dir,
			Target:            dir,
			Lang:              config.DefaultLang,
			Geocoder:          "none",
			GeocodeCache:      "",
			GeohashPrecision:  config.DefaultGeohashPrecision,
			SegmentGapMinutes: config.DefaultSegmentGapMinutes,
		}
		result, err := app.BuildPlan(context.Background(), cfg)
		if err != nil {
			return eventStatsMsg{token: token, err: err}
		}
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
		summary := fmt.Sprintf("Events: segments=%d types=%d top[%s]", es.TotalSegments, es.DistinctTypes, strings.Join(parts, ", "))
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
		return eventStatsMsg{
			token: token,
			dir:   dir,
			analysis: eventAnalysis{
				summary:      summary,
				segmentNames: segmentNames,
			},
		}
	}
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
	rightWidth := width
	if rightWidth < defaultPaneMinWidth {
		rightWidth = defaultPaneMinWidth
	}
	textWidth := rightWidth - 4
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
	return renderScrollablePane(lines, rightWidth, m.paneHeight(), m.rightScroll, active)
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
		// Browser header + Selected + blank + ".." row.
		return 4
	}
	// Target right-pane fixed rows excluding filepicker:
	// Selected + blank + Step2 + selected + blank + Preview(header+N) + blank + Browser + blank + ".." + blank + Confirm + Ctrl+S
	return 12 + m.targetPreviewLineCount()
}

func (m pathPickerModel) targetPreviewLineCount() int {
	if m.sourceBusy || m.sourceEvErr != "" || len(m.sourceCreateDirs) == 0 {
		return 1
	}
	lines := len(m.sourceCreateDirs)
	if lines > targetPreviewMaxDirs {
		lines = targetPreviewMaxDirs + 1 // include "... +N more"
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
		cfg := config.Config{
			Source:            dir,
			Target:            dir,
			Lang:              config.DefaultLang,
			Geocoder:          "none",
			GeocodeCache:      "",
			GeohashPrecision:  config.DefaultGeohashPrecision,
			SegmentGapMinutes: config.DefaultSegmentGapMinutes,
		}
		result, err := app.BuildPlan(context.Background(), cfg)
		if err != nil {
			return sourceEventStatsMsg{token: token, err: err}
		}
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
		summary := fmt.Sprintf("Events: segments=%d types=%d top[%s]", es.TotalSegments, es.DistinctTypes, strings.Join(parts, ", "))
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
		return sourceEventStatsMsg{
			token: token,
			analysis: eventAnalysis{
				summary:      summary,
				segmentNames: segmentNames,
			},
			createDirs: buildTargetCreatePreview(result),
		}
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
	// Keep one column of slack so wrapped rendering never kicks in.
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
