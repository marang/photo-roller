package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestPathPickerViewStartsWithSubtitle(t *testing.T) {
	p := newConfiguredDirPicker(".")
	m := pathPickerModel{
		subtitle:   "Path to camera/SD DCIM root",
		picker:     p,
		defaultDir: ".",
		focus:      NewFocusRing(2),
		paneFocus:  NewFocusRing(2),
	}
	m.focus.Focus(focusPicker)
	m.width = 120
	m.height = 30

	view := m.View()
	first := strings.Split(view, "\n")[0]
	if !strings.Contains(first, "Path to camera/SD DCIM root") {
		t.Fatalf("expected subtitle in first line, got: %q", first)
	}
}

func TestWizardPromptViewStartsWithIntroText(t *testing.T) {
	m := selectPromptModel{
		title:    "Step 3 - Preflight Review",
		subtitle: "Validate the planned import before copying",
		left:     "Step 1 - Source Directory\n/media/camera/DCIM",
		summary:  "Source: /media/camera/DCIM\nTarget: /mnt/data/assets/__albums",
		options: []SelectOption{
			{Title: "Confirm", Value: "confirm"},
		},
		width:  120,
		height: 30,
	}

	view := m.View()
	first := strings.Split(view, "\n")[0]
	if !strings.Contains(first, "Validate the planned import before copying") {
		t.Fatalf("expected intro text in first line, got: %q", first)
	}
}

func TestProgressViewStartsWithIntroText(t *testing.T) {
	m := NewProgressModel(nil, 10, "Step 1 - Source Directory\n/media/camera/DCIM", "Copying planned files to target")
	m.width = 120
	m.height = 30

	view := m.View()
	first := strings.Split(view, "\n")[0]
	if !strings.Contains(first, "Copying planned files to target") {
		t.Fatalf("expected intro text in first line, got: %q", first)
	}
}

func TestWizardPromptConfirmLabelIsSingleLine(t *testing.T) {
	m := selectPromptModel{
		title:    "Step 3 - Preflight Review",
		subtitle: "Validate the planned import before copying",
		left:     "Step 1 - Source Directory\n/src\nStep 2 - Target Directory\n/dst",
		summary:  "Source: /src\nTarget: /dst",
		options: []SelectOption{
			{Title: "Confirm", Value: "confirm"},
		},
		width:  120,
		height: 30,
	}

	view := m.View()
	if !strings.Contains(view, "> Confirm [Ctrl+S]") {
		t.Fatalf("expected unified confirm label, got:\n%s", view)
	}
	if strings.Contains(view, "\nCtrl+S\n") {
		t.Fatalf("did not expect standalone Ctrl+S line, got:\n%s", view)
	}
}

func TestWizardPromptDoesNotQuitOnCommonNonQuitKeys(t *testing.T) {
	for _, key := range []string{"ctrl+c", "q", "esc"} {
		m := selectPromptModel{
			options: []SelectOption{
				{Title: "Confirm", Value: "confirm"},
			},
		}
		updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		if cmd != nil {
			t.Fatalf("key %q returned unexpected command", key)
		}
		if updated.(selectPromptModel).cancel {
			t.Fatalf("key %q should not cancel", key)
		}
	}
}
