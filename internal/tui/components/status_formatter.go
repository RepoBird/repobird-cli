// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package components

import (
	"fmt"
	"strings"
)

// StatusFormatter provides consistent formatting for status line components
type StatusFormatter struct {
	viewName string
	width    int
}

// NewStatusFormatter creates a new status formatter for a specific view
func NewStatusFormatter(viewName string, width int) *StatusFormatter {
	return &StatusFormatter{
		viewName: viewName,
		width:    width,
	}
}

// FormatViewName formats the view name consistently with brackets
// Examples: [DASH], [STATUS], [CREATE], [BULK], [ERROR]
func (f *StatusFormatter) FormatViewName() string {
	return fmt.Sprintf("[%s]", strings.ToUpper(f.viewName))
}

// FormatViewNameWithMode formats view name with optional mode indicator
// Examples: [CREATE] [INPUT], [BULK] [NAV]
func (f *StatusFormatter) FormatViewNameWithMode(mode string) string {
	base := f.FormatViewName()
	if mode != "" {
		return fmt.Sprintf("%s [%s]", base, strings.ToUpper(mode))
	}
	return base
}

// FormatHelp formats help text consistently
// Automatically truncates if needed based on available width
func (f *StatusFormatter) FormatHelp(leftContent, rightContent, helpText string) string {
	leftWidth := len(leftContent)
	rightWidth := len(rightContent)
	availableForHelp := f.width - leftWidth - rightWidth - 4 // 4 for padding/spacing

	if availableForHelp < len(helpText) {
		// Truncate or use shorter version
		return f.truncateHelp(helpText, availableForHelp)
	}
	return helpText
}

// truncateHelp intelligently truncates help text to fit available space
func (f *StatusFormatter) truncateHelp(helpText string, maxWidth int) string {
	if maxWidth < 15 {
		return "?:help"
	}
	if maxWidth < 25 {
		return "?:help q:quit"
	}
	if maxWidth < 40 {
		// Extract most important keys from the help text
		if strings.Contains(helpText, "enter") {
			return "enter:ok ?:help q:quit"
		}
		return "?:help q:quit"
	}

	// Otherwise truncate with ellipsis
	if len(helpText) > maxWidth {
		return helpText[:maxWidth-3] + "..."
	}
	return helpText
}

// StandardStatusLine creates a standard status line with consistent formatting
func (f *StatusFormatter) StandardStatusLine(leftContent, rightContent, helpText string) *StatusLine {
	// Ensure consistent formatting
	if leftContent == "" {
		leftContent = f.FormatViewName()
	}

	// Format help text based on available space
	formattedHelp := f.FormatHelp(leftContent, rightContent, helpText)

	return NewStatusLine().
		SetWidth(f.width).
		SetLeft(leftContent).
		SetRight(rightContent).
		SetHelp(formattedHelp).
		ResetStyle() // Ensure consistent styling
}
