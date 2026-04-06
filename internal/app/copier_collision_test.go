package app

import (
	"testing"

	"github.com/marang/photo-roller/internal/config"
)

func TestResolveTargetName(t *testing.T) {
	name, err := resolveTargetName("A.JPG", 1, config.CollisionModeSuffix)
	if err != nil {
		t.Fatal(err)
	}
	if name != "A.JPG" {
		t.Fatalf("unexpected name %q", name)
	}

	name, err = resolveTargetName("A.JPG", 2, config.CollisionModeSuffix)
	if err != nil {
		t.Fatal(err)
	}
	if name != "A_2.JPG" {
		t.Fatalf("unexpected suffix name %q", name)
	}

	if _, err = resolveTargetName("A.JPG", 2, config.CollisionModeFail); err == nil {
		t.Fatal("expected collision error in fail mode")
	}
}
