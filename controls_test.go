//go:build windows
// +build windows

package directr

import "testing"

func TestControlTypeMappings(t *testing.T) {
	buttonType, ok := ControlTypeFromName("button")
	if !ok {
		t.Fatalf("expected button mapping to resolve")
	}
	if got := ControlTypeToName(buttonType); got != "button" {
		t.Fatalf("expected button, got %q", got)
	}
	if _, ok := ControlTypeFromName("unknown"); ok {
		t.Fatalf("expected unknown control type to be false")
	}
}

func TestControlTypeFromNameNormalization(t *testing.T) {
	if got, ok := ControlTypeFromName("  BuTtOn "); !ok || ControlTypeToName(got) != "button" {
		t.Fatalf("expected trimmed/case-insensitive button mapping, got %v %v", got, ok)
	}
	if got, ok := ControlTypeFromName("image"); !ok || ControlTypeToName(got) != "img" {
		t.Fatalf("expected image alias mapping, got %v %v", got, ok)
	}
}
