package ui

import (
	"fmt"
	"strings"

	"github.com/marang/photo-roller/internal/app"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type eventMsg app.ProgressEvent
type doneMsg struct{}
type errMsg struct{ err error }

type Model struct {
	spin        spinner.Model
	bar         progress.Model
	events      <-chan app.ProgressEvent
	left        string
	stepTitle   string
	subtitle    string
	current     string
	segment     string
	last        string
	done        int
	total       int
	warnings    int
	width       int
	height      int
	finished    bool
	runErr      error
	focus       FocusRing
	leftScroll  int
	rightScroll int
}

func NewProgressModel(events <-chan app.ProgressEvent, total int, leftSummary, subtitle string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorSpinner))

	b := progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())
	focus := NewFocusRing(2)
	focus.Focus(1)
	return Model{
		spin:      s,
		bar:       b,
		events:    events,
		total:     total,
		left:      leftSummary,
		stepTitle: "Step 4 - Execute Import",
		subtitle:  subtitle,
		focus:     focus,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spin.Tick, waitEventCmd(m.events))
}

func waitEventCmd(events <-chan app.ProgressEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return doneMsg{}
		}
		return eventMsg(ev)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case progress.FrameMsg:
		var cmd tea.Cmd
		updatedBar, next := m.bar.Update(msg)
		m.bar = updatedBar.(progress.Model)
		cmd = next
		return m, cmd
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		barWidth := msg.Width - 10
		if barWidth < 20 {
			barWidth = 20
		}
		m.bar.Width = barWidth
		return m, nil
	case eventMsg:
		ev := app.ProgressEvent(msg)
		switch ev.Kind {
		case app.EventDayStart:
			m.current = ev.Day
			m.segment = ""
		case app.EventSegmentStart:
			if ev.Segment != "" {
				m.segment = ev.Segment
			}
		case app.EventFileDone:
			m.done = ev.Done
			m.total = ev.Total
			if ev.Segment != "" {
				m.segment = ev.Segment
			}
		case app.EventWarning:
			m.warnings++
			if ev.Segment != "" {
				m.segment = ev.Segment
			}
			if ev.Message != "" {
				m.last = ev.Message
			}
		case app.EventDone:
			m.done = ev.Done
			m.total = ev.Total
		}

		var pct float64
		if m.total > 0 {
			pct = float64(m.done) / float64(m.total)
		}
		return m, tea.Batch(m.spin.Tick, m.bar.SetPercent(pct), waitEventCmd(m.events))
	case doneMsg:
		m.finished = true
		return m, tea.Quit
	case errMsg:
		m.runErr = msg.err
		m.finished = true
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+d":
			m.finished = true
			return m, tea.Quit
		case "tab":
			if strings.TrimSpace(m.left) != "" {
				m.focus.Next()
			}
		case "up", "k":
			if strings.TrimSpace(m.left) == "" {
				return m, nil
			}
			if m.focus.Is(0) {
				m.leftScroll--
				if m.leftScroll < 0 {
					m.leftScroll = 0
				}
			} else {
				m.rightScroll--
				if m.rightScroll < 0 {
					m.rightScroll = 0
				}
			}
		case "down", "j":
			if strings.TrimSpace(m.left) == "" {
				return m, nil
			}
			if m.focus.Is(0) {
				m.leftScroll++
			} else {
				m.rightScroll++
			}
		case "pgdown", "J":
			if strings.TrimSpace(m.left) == "" {
				return m, nil
			}
			scrollStep := 5
			if m.focus.Is(0) {
				m.leftScroll += scrollStep
			} else {
				m.rightScroll += scrollStep
			}
		case "pgup", "K":
			if strings.TrimSpace(m.left) == "" {
				return m, nil
			}
			scrollStep := 5
			if m.focus.Is(0) {
				m.leftScroll -= scrollStep
				if m.leftScroll < 0 {
					m.leftScroll = 0
				}
			} else {
				m.rightScroll -= scrollStep
				if m.rightScroll < 0 {
					m.rightScroll = 0
				}
			}
		}
		m.clampScrolls()
	}
	return m, nil
}

func (m Model) View() string {
	if m.left != "" {
		return m.twoPaneView()
	}

	title := styleTitle().Render("PhotoRoller Apply")
	status := fmt.Sprintf("%s Working", m.spin.View())
	if m.finished {
		status = "Done"
	}

	current := "Current day: -"
	if m.current != "" {
		current = "Current day: " + m.current
	}

	segment := "Current segment: -"
	if m.segment != "" {
		segment = "Current segment: " + m.segment
	}

	warnStyle := styleWarning()
	warnings := "Warnings: 0"
	if m.warnings > 0 {
		warnings = warnStyle.Render(fmt.Sprintf("Warnings: %d", m.warnings))
	}

	lines := []string{
		title,
		status,
		m.bar.View(),
		fmt.Sprintf("Progress: %d/%d", m.done, m.total),
		current,
		segment,
		warnings,
	}
	if m.last != "" {
		lines = append(lines, "Status: "+m.last)
	}
	return strings.Join(lines, "\n") + "\n"
}

func (m Model) twoPaneView() string {
	width := m.width
	if width <= 0 {
		width = 120
	}
	leftWidth, rightWidth := splitStandardPaneWidths(width)
	height := m.height
	if height <= 0 {
		height = 30
	}
	paneHeight := framePaneHeight(height)
	contentHeight := paneHeight - 3
	if contentHeight < 1 {
		contentHeight = 1
	}

	leftHeadingStyle := styleLabel()
	leftActiveHeadingStyle := styleAccent()
	leftNormalStyle := styleValue()
	leftTextWidth := leftWidth - 4
	if leftTextWidth < 20 {
		leftTextWidth = 20
	}

	leftLinesRaw := strings.Split(m.left, "\n")
	if len(leftLinesRaw) == 0 {
		leftLinesRaw = []string{"-"}
	}
	renderedLeft := make([]string, 0, len(leftLinesRaw))
	for _, line := range leftLinesRaw {
		clamped := clampLine(line, leftTextWidth)
		if isStepHeadingLine(line) {
			if m.focus.Is(0) {
				renderedLeft = append(renderedLeft, leftActiveHeadingStyle.Render(clamped))
			} else {
				renderedLeft = append(renderedLeft, leftHeadingStyle.Render(clamped))
			}
			continue
		}
		renderedLeft = append(renderedLeft, leftNormalStyle.Render(clamped))
	}
	_, _, _, leftBody := scrollWindow(renderedLeft, m.leftScroll, contentHeight)
	leftContent := leftBody
	leftPane := renderPane(leftContent, leftWidth, paneHeight, m.focus.Is(0))

	status := fmt.Sprintf("%s Working", m.spin.View())
	if m.finished {
		status = "Done"
	}
	current := "Current day: -"
	if m.current != "" {
		current = "Current day: " + m.current
	}
	segment := "Current segment: -"
	if m.segment != "" {
		segment = "Current segment: " + m.segment
	}
	warnStyle := styleWarning()
	warnings := "Warnings: 0"
	if m.warnings > 0 {
		warnings = warnStyle.Render(fmt.Sprintf("Warnings: %d", m.warnings))
	}
	rightTextWidth := rightWidth - 4
	if rightTextWidth < 20 {
		rightTextWidth = 20
	}
	m.bar.Width = rightTextWidth - 2
	if m.bar.Width < 10 {
		m.bar.Width = 10
	}

	rightHeaderStyle := styleLabel()
	rightContent := []string{
		clampLine(status, rightTextWidth),
		m.bar.View(),
		clampLine(fmt.Sprintf("Progress: %d/%d", m.done, m.total), rightTextWidth),
		clampLine(current, rightTextWidth),
		clampLine(segment, rightTextWidth),
		clampLine(warnings, rightTextWidth),
	}
	if m.last != "" {
		rightContent = append(rightContent, clampLine("Status: "+m.last, rightTextWidth))
	}
	_, _, _, rightBody := scrollWindow(rightContent, m.rightScroll, contentHeight)
	rightText := rightHeaderStyle.Render(clampLine(m.stepTitle, rightTextWidth)) + "\n\n" + rightBody
	rightPane := renderPane(rightText, rightWidth, paneHeight, m.focus.Is(1))

	hintText := "Keys: Ctrl+D cancel."
	if strings.TrimSpace(m.left) != "" {
		focus := "right"
		if m.focus.Is(0) {
			focus = "left"
		}
		hintText = fmt.Sprintf("Focus: %s pane | Keys: Tab switch pane, ↑/↓ or PgUp/PgDn scroll focused pane, Ctrl+D cancel.", focus)
	}
	hint := styleMuted().Render(clampLine(hintText, width-2))
	body := joinPanes(leftPane, rightPane, defaultPaneGap)
	return renderScreenFrame(width, frameIntro(m.stepTitle, m.subtitle), body, hint)
}

func (m *Model) clampScrolls() {
	if strings.TrimSpace(m.left) == "" {
		m.leftScroll = 0
		m.rightScroll = 0
		return
	}
	contentHeight := m.progressContentHeight()
	leftTotal := len(strings.Split(m.left, "\n"))
	rightTotal := 6
	if m.last != "" {
		rightTotal++
	}

	maxLeft := leftTotal - contentHeight
	if maxLeft < 0 {
		maxLeft = 0
	}
	maxRight := rightTotal - contentHeight
	if maxRight < 0 {
		maxRight = 0
	}
	if m.leftScroll < 0 {
		m.leftScroll = 0
	}
	if m.rightScroll < 0 {
		m.rightScroll = 0
	}
	if m.leftScroll > maxLeft {
		m.leftScroll = maxLeft
	}
	if m.rightScroll > maxRight {
		m.rightScroll = maxRight
	}
}

func (m Model) progressContentHeight() int {
	height := m.height
	if height <= 0 {
		height = 30
	}
	paneHeight := framePaneHeight(height)
	contentHeight := paneHeight - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	return contentHeight
}
