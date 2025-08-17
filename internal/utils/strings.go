// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TruncateWithEllipsis truncates a string to fit within maxWidth using display width (handles unicode/emoji properly).
// This should be used for terminal display where visual width matters.
func TruncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Use runes to handle unicode properly when truncating
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		truncated := string(runes[:i]) + "..."
		if lipgloss.Width(truncated) <= maxWidth {
			return truncated
		}
	}
	return "..."
}

// TruncateSimple truncates a string based on byte length.
// This is faster but doesn't handle unicode display width correctly.
// Use this for non-display purposes or when you know the string is ASCII.
func TruncateSimple(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// TruncateMultiline truncates a string handling newlines and tabs.
// Takes only the first line and converts tabs to spaces.
// Uses rune counting for proper unicode handling.
func TruncateMultiline(s string, maxWidth int) string {
	// Handle newlines by taking only the first line
	lines := strings.Split(s, "\n")
	if len(lines) > 0 {
		s = lines[0]
	}

	// Convert tabs to spaces for consistent display
	s = strings.ReplaceAll(s, "\t", "    ")

	// Use rune counting for proper unicode handling
	runes := []rune(s)
	if len(runes) <= maxWidth {
		return s
	}

	// Leave room for ellipsis
	if maxWidth > 3 {
		return string(runes[:maxWidth-3]) + "..."
	}
	return "..."
}
