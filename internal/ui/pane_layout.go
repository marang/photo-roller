package ui

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	paneDefaultBorderColor = "244"
	paneActiveBorderColor  = "212"
	defaultPaneGap         = 2
	defaultPaneMinWidth    = 30
	defaultScreenWidth     = 120
	screenOuterRows        = 4
	screenSafetyRows       = 2
	minPaneHeight          = 8
)

func splitStandardPaneWidths(totalWidth int) (int, int) {
	return splitPaneWidths(totalWidth, defaultPaneMinWidth, defaultPaneMinWidth, defaultPaneGap)
}

func splitPaneWidths(totalWidth, leftMin, rightMin, gap int) (int, int) {
	if gap <= 0 {
		gap = defaultPaneGap
	}
	usable := totalWidth - gap
	if usable < leftMin+rightMin {
		usable = leftMin + rightMin
	}
	left := usable / 2
	if left < leftMin {
		left = leftMin
	}
	right := usable - left
	if right < rightMin {
		right = rightMin
		left = usable - right
		if left < leftMin {
			left = leftMin
		}
	}
	return left, right
}

func renderPane(content string, width, height int, active bool) string {
	borderColor := paneDefaultBorderColor
	if active {
		borderColor = paneActiveBorderColor
	}
	style := lipgloss.NewStyle().
		Width(width).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), true, false, true, false).
		BorderForeground(lipgloss.Color(borderColor))
	if height > 0 {
		style = style.Height(height)
	}
	return style.Render(content)
}

func joinPanes(left, right string, gap int) string {
	if gap <= 0 {
		gap = defaultPaneGap
	}
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		lipgloss.NewStyle().Width(gap).Render(""),
		right,
	)
}

func scrollWindow(lines []string, offset, height int) (int, int, int, string) {
	total := len(lines)
	if total == 0 {
		return 0, 0, 0, ""
	}
	if height < 1 {
		height = 1
	}
	maxOffset := total - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	end := offset + height
	if end > total {
		end = total
	}
	view := lines[offset:end]
	return offset, end, total, strings.Join(view, "\n")
}

func renderScrollablePane(lines []string, width, height, scroll int, active bool) string {
	if len(lines) == 0 {
		lines = []string{"-"}
	}
	contentHeight := height - 3
	if contentHeight < 1 {
		contentHeight = 1
	}
	_, _, _, body := scrollWindow(lines, scroll, contentHeight)
	bodyLines := strings.Split(body, "\n")
	for len(bodyLines) < contentHeight {
		bodyLines = append(bodyLines, "")
	}
	body = strings.Join(bodyLines, "\n")
	return renderPane(body, width, height, active)
}

func normalizeScreenWidth(width int) int {
	if width <= 0 {
		return defaultScreenWidth
	}
	return width
}

func framePaneHeight(totalHeight int) int {
	if totalHeight <= 0 {
		return 16
	}
	h := totalHeight - screenOuterRows - screenSafetyRows
	if h < minPaneHeight {
		h = minPaneHeight
	}
	return h
}

func renderScreenFrame(width int, intro, body, hint string) string {
	w := normalizeScreenWidth(width)
	top := clampLine(sanitizeHeaderText(intro), w-2)
	if top == "" {
		top = "Header unavailable"
	}
	bottom := clampLine(strings.TrimSpace(hint), w-2)
	return top + "\n\n" + body + "\n" + bottom
}

func frameIntro(title, subtitle string) string {
	sub := strings.TrimSpace(subtitle)
	if sub != "" {
		return sub
	}
	return strings.TrimSpace(title)
}

func sanitizeHeaderText(in string) string {
	plain := ansi.Strip(in)
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || unicode.IsControl(r) {
			return -1
		}
		return r
	}, plain)
}
