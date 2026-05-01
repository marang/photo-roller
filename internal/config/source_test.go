package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSourceDefaultFindsFirstUserDCIM(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "tester", "CARD_A", "DCIM")
	second := filepath.Join(root, "tester", "CARD_B", "DCIM")
	if err := os.MkdirAll(second, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(first, 0o755); err != nil {
		t.Fatal(err)
	}

	got, ok := detectSourceDefault([]string{root}, "tester")
	if !ok {
		t.Fatal("expected source detection to find a DCIM directory")
	}
	want := first + string(os.PathSeparator)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDetectSourceDefaultFallsBackWithoutMatches(t *testing.T) {
	got := DetectSourceDefault()
	if got == "" {
		t.Fatal("expected non-empty fallback source")
	}
}
