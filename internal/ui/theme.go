package ui

import "github.com/charmbracelet/lipgloss"

const (
	colorTitle    = "86"
	colorMuted    = "244"
	colorValue    = "250"
	colorDimValue = "245"
	colorAccent   = "212"
	colorWarning  = "214"
	colorSpinner  = "69"
)

func styleTitle() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorTitle))
}

func styleLabel() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorTitle))
}

func styleMuted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorMuted))
}

func styleValue() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorValue))
}

func styleDimValue() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorDimValue))
}

func styleAccent() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorAccent)).Bold(true)
}

func styleWarning() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(colorWarning))
}
