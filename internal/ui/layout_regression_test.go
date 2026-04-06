package ui

import (
	"strings"
	"testing"
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
