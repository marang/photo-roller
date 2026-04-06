package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type textPromptModel struct {
	title    string
	subtitle string
	input    textinput.Model
	value    string
	cancel   bool
}

func RunTextPrompt(title, subtitle, initialValue, placeholder string) (string, error) {
	ti := textinput.New()
	ti.SetValue(initialValue)
	ti.Placeholder = placeholder
	ti.Focus()
	ti.Prompt = "> "
	ti.Cursor.Style = styleLabel()
	ti.CharLimit = 4096

	model := textPromptModel{
		title:    title,
		subtitle: subtitle,
		input:    ti,
	}

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return "", err
	}
	out := finalModel.(textPromptModel)
	if out.cancel {
		return "", fmt.Errorf("input cancelled")
	}
	return strings.TrimSpace(out.value), nil
}

func (m textPromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m textPromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+d":
			m.cancel = true
			return m, tea.Quit
		case "enter":
			m.value = m.input.Value()
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m textPromptModel) View() string {
	title := styleTitle().Render(m.title)
	sub := styleMuted().Render(m.subtitle)
	hint := styleMuted().Render("Edit and press Enter. Ctrl+D cancel.")
	return title + "\n" + sub + "\n\n" + m.input.View() + "\n" + hint + "\n"
}
