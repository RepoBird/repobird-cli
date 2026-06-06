// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package output

import (
	"bytes"
	"testing"
)

func TestNewStylerHonorsColorMode(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		wantANSI  bool
		wantPlain string
	}{
		{name: "always emits ANSI", mode: ColorAlways, wantANSI: true, wantPlain: "Created"},
		{name: "never emits plain text", mode: ColorNever, wantANSI: false, wantPlain: "Created"},
		{name: "unknown mode falls back to auto plain for non-terminal writers", mode: "loud", wantANSI: false, wantPlain: "Created"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			styler := NewStyler(&out, tt.mode)
			got := styler.Success("Created")

			if !bytes.Contains([]byte(got), []byte(tt.wantPlain)) {
				t.Fatalf("expected %q to contain %q", got, tt.wantPlain)
			}
			if hasANSI(got) != tt.wantANSI {
				t.Fatalf("hasANSI(%q) = %v, want %v", got, hasANSI(got), tt.wantANSI)
			}
		})
	}
}

func TestStatusUsesSemanticColors(t *testing.T) {
	var out bytes.Buffer
	styler := NewStyler(&out, ColorAlways)

	tests := []string{"DONE", "FAILED", "PROCESSING", "QUEUED"}
	for _, status := range tests {
		got := styler.Status(status)
		if !hasANSI(got) {
			t.Fatalf("expected status %s to be colored, got %q", status, got)
		}
		if !bytes.Contains([]byte(got), []byte(status)) {
			t.Fatalf("expected status output to contain original text %q, got %q", status, got)
		}
	}
}

func TestColorModeFromEnv(t *testing.T) {
	t.Run("NO_COLOR disables color", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		t.Setenv("REPOBIRD_COLOR", "")

		if got := ModeFromEnv(ColorAlways); got != ColorNever {
			t.Fatalf("ModeFromEnv(always) = %q, want %q", got, ColorNever)
		}
	})

	t.Run("REPOBIRD_COLOR overrides config", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		t.Setenv("REPOBIRD_COLOR", ColorAlways)

		if got := ModeFromEnv(ColorNever); got != ColorAlways {
			t.Fatalf("ModeFromEnv(never) = %q, want %q", got, ColorAlways)
		}
	})
}
