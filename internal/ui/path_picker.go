package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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
