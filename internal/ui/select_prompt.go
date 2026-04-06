package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	minPromptWidth       = 60
	defaultPromptWidth   = 100
	minSummaryPaneHeight = 8
)

const WizardBackChoice = "__wizard_back__"

type SelectOption struct {
	Title       string
	Description string
	Value       string
}

type selectPromptModel struct {
	title      string
	subtitle   string
	left       string
	summary    string
	options    []SelectOption
	cursor     int
	width      int
	height     int
	choice     string
	cancel     bool
	scroll     int
	leftFocus  bool
	leftCursor int
	leftScroll int
}

func RunSelectPrompt(title, subtitle string, options []SelectOption) (string, error) {
	model := selectPromptModel{
		title:    title,
		subtitle: "",
		summary:  subtitle,
		options:  options,
	}
	return runSelectPromptModel(model)
}

func RunWizardSelectPrompt(title, subtitle, leftSummary, rightSummary string, options []SelectOption) (string, error) {
	model := selectPromptModel{
		title:    title,
		subtitle: subtitle,
		left:     leftSummary,
		summary:  rightSummary,
		options:  options,
	}
	return runSelectPromptModel(model)
}

func runSelectPromptModel(model selectPromptModel) (string, error) {
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return "", err
	}
	out := finalModel.(selectPromptModel)
	if out.cancel {
		return "", fmt.Errorf("selection cancelled")
	}
	return out.choice, nil
}

func (m selectPromptModel) Init() tea.Cmd {
	return nil
}

func (m selectPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+d":
			m.cancel = true
			return m, tea.Quit
		case "b":
			if strings.TrimSpace(m.left) != "" {
				m.choice = WizardBackChoice
				return m, tea.Quit
			}
		case "tab":
			if strings.TrimSpace(m.left) != "" {
				m.leftFocus = !m.leftFocus
				if m.leftFocus {
					m.leftCursor = clampLeftCursor(m.leftCursor, len(m.leftHeadingIndices()))
				}
			}
		case "up", "k":
			if m.leftFocus {
				m.leftCursor--
				m.leftCursor = clampLeftCursor(m.leftCursor, len(m.leftHeadingIndices()))
				return m, nil
			}
			if len(m.options) > 0 {
				m.cursor--
				if m.cursor < 0 {
					m.cursor = len(m.options) - 1
				}
			}
		case "down", "j":
			if m.leftFocus {
				m.leftCursor++
				m.leftCursor = clampLeftCursor(m.leftCursor, len(m.leftHeadingIndices()))
				return m, nil
			}
			if len(m.options) > 0 {
				m.cursor++
				if m.cursor >= len(m.options) {
					m.cursor = 0
				}
			}
		case "pgdown", "J":
			if m.leftFocus {
				lines := strings.Split(m.leftRenderedText(), "\n")
				m.leftScroll += 5
				maxScroll := len(lines) - m.summaryPaneHeight()
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.leftScroll > maxScroll {
					m.leftScroll = maxScroll
				}
			} else {
				lines := strings.Split(m.summary, "\n")
				m.scroll += 5
				maxScroll := len(lines) - m.summaryPaneHeight()
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.scroll > maxScroll {
					m.scroll = maxScroll
				}
			}
		case "pgup", "K":
			if m.leftFocus {
				m.leftScroll -= 5
				if m.leftScroll < 0 {
					m.leftScroll = 0
				}
			} else {
				m.scroll -= 5
				if m.scroll < 0 {
					m.scroll = 0
				}
			}
		case "enter":
			if m.leftFocus {
				return m, nil
			}
			if len(m.options) == 0 {
				return m, nil
			}
			m.choice = m.options[m.cursor].Value
			return m, tea.Quit
		case "ctrl+s":
			if m.leftFocus {
				return m, nil
			}
			if len(m.options) == 0 {
				return m, nil
			}
			m.choice = m.options[m.cursor].Value
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectPromptModel) View() string {
	width := m.width
	if width <= 0 {
		width = defaultPromptWidth
	}
	if width < minPromptWidth {
		width = minPromptWidth
	}
	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	hintText := "Keys: ↑/↓ select option, Enter confirm, PgUp/PgDn scroll summary, Ctrl+D cancel."
	if strings.TrimSpace(m.left) != "" {
		hintText = "Keys: Tab focus left/right, ↑/↓ navigate, Enter/Ctrl+S confirm (right), PgUp/PgDn scroll focused pane, b back, Ctrl+D cancel."
	}
	hint := styleMuted().Render(clampLine(hintText, width-2))

	if strings.TrimSpace(m.left) == "" {
		body := m.renderPromptPane(contentWidth, m.summaryPaneHeight(), m.title, true)
		return renderScreenFrame(width, frameIntro(m.title, m.subtitle), body, hint)
	}

	leftWidth, rightWidth := splitStandardPaneWidths(contentWidth)
	paneHeight := m.summaryPaneHeight()
	leftPane := m.renderLeftPane(leftWidth, paneHeight)
	rightPane := m.renderPromptPane(rightWidth, paneHeight, m.title, !m.leftFocus)
	body := joinPanes(leftPane, rightPane, defaultPaneGap)
	return renderScreenFrame(width, frameIntro(m.title, m.wizardIntro()), body, hint)
}

func (m selectPromptModel) wizardIntro() string {
	if strings.TrimSpace(m.subtitle) != "" {
		return m.subtitle
	}
	switch m.title {
	case "Step 3 - Collision Mode":
		return "Review detected filename collisions"
	case "Step 3 - Preflight Review":
		return "Validate the planned import before copying"
	case "Step 4 - Execute Import":
		return "Confirm and proceed with import workflow"
	default:
		return m.title
	}
}

func (m selectPromptModel) summaryPaneHeight() int {
	height := framePaneHeight(m.height)
	if height < minSummaryPaneHeight {
		height = minSummaryPaneHeight
	}
	return height
}

func (m selectPromptModel) renderPromptPane(width, paneHeight int, title string, active bool) string {
	label := styleLabel()
	textWidth := width - 4
	if textWidth < 20 {
		textWidth = 20
	}

	summaryLines := strings.Split(m.summary, "\n")
	if len(summaryLines) == 0 {
		summaryLines = []string{"-"}
	}
	for i, line := range summaryLines {
		summaryLines[i] = clampLine(line, textWidth)
	}

	optionRows := m.optionRows(textWidth)
	reserved := 2 + len(optionRows) // blank + options
	contentHeight := paneHeight - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	summaryHeight := contentHeight - reserved
	if summaryHeight < 1 {
		summaryHeight = 1
	}

	_, _, _, summaryBody := scrollWindow(summaryLines, m.scroll, summaryHeight)
	visibleSummary := strings.Split(summaryBody, "\n")
	for len(visibleSummary) < summaryHeight {
		visibleSummary = append(visibleSummary, "")
	}

	lines := []string{
		label.Render(title),
		"",
	}
	lines = append(lines, optionRows...)
	lines = append(lines, "")
	lines = append(lines, "")
	lines = append(lines, visibleSummary...)
	return renderPane(strings.Join(lines, "\n"), width, paneHeight, active)
}

func (m selectPromptModel) renderLeftPane(width, paneHeight int) string {
	label := styleLabel().Render("Confirmed")
	textWidth := width - 4
	if textWidth < 20 {
		textWidth = 20
	}
	lines := strings.Split(m.left, "\n")
	if len(lines) == 0 {
		lines = []string{"-"}
	}
	headings := m.leftHeadingIndices()
	selectedHeading := clampLeftCursor(m.leftCursor, len(headings))

	headingStyle := styleLabel()
	activeHeadingStyle := styleAccent()
	normalStyle := styleValue()
	focusTag := styleMuted()

	rendered := make([]string, 0, len(lines)+1)
	if m.leftFocus {
		rendered = append(rendered, focusTag.Render(clampLine("Focus: step headers", textWidth)))
		rendered = append(rendered, "")
	}
	for i, line := range lines {
		if isStepHeadingLine(line) {
			prefix := "  "
			style := headingStyle
			if m.leftFocus && headingLineOrdinal(i, headings) == selectedHeading {
				prefix = "> "
				style = activeHeadingStyle
			}
			rendered = append(rendered, prefix+style.Render(clampLine(line, textWidth-2)))
			continue
		}
		rendered = append(rendered, normalStyle.Render(clampLine(line, textWidth)))
	}
	contentHeight := paneHeight - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	_, _, _, body := scrollWindow(rendered, m.leftScroll, contentHeight)
	content := label + "\n\n" + body
	return renderPane(content, width, paneHeight, m.leftFocus)
}

func (m selectPromptModel) leftRenderedText() string {
	lines := strings.Split(m.left, "\n")
	return strings.Join(lines, "\n")
}

func (m selectPromptModel) leftHeadingIndices() []int {
	lines := strings.Split(m.left, "\n")
	idx := make([]int, 0, 8)
	for i, line := range lines {
		if isStepHeadingLine(line) {
			idx = append(idx, i)
		}
	}
	return idx
}

func isStepHeadingLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "Step ")
}

func headingLineOrdinal(lineIndex int, headingIndices []int) int {
	for i, idx := range headingIndices {
		if idx == lineIndex {
			return i
		}
	}
	return 0
}

func clampLeftCursor(cursor, headingCount int) int {
	if headingCount <= 0 {
		return 0
	}
	if cursor < 0 {
		return headingCount - 1
	}
	if cursor >= headingCount {
		return 0
	}
	return cursor
}

func (m selectPromptModel) optionRows(textWidth int) []string {
	muted := styleMuted()
	selectedStyle := styleAccent()
	cursorStyle := styleAccent()
	rows := make([]string, 0, len(m.options)*2)
	for i, option := range m.options {
		prefix := "  "
		titleText := option.Title
		if strings.EqualFold(strings.TrimSpace(titleText), "confirm") {
			titleText = "Confirm [Ctrl+S]"
		}
		title := clampLine(titleText, textWidth-3)
		desc := clampLine(option.Description, textWidth-4)
		if i == m.cursor {
			prefix = cursorStyle.Render("> ")
			title = selectedStyle.Render(title)
			desc = selectedStyle.Render(desc)
		}
		rows = append(rows, prefix+title)
		if strings.TrimSpace(desc) != "" {
			rows = append(rows, "   "+desc)
		}
	}
	if len(rows) == 0 {
		rows = append(rows, muted.Render("No actions available."))
	}
	return rows
}
