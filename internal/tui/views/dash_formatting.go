// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
)

// getRepositoryStatusIcon returns an icon based on repository status
func (d *DashboardView) getRepositoryStatusIcon(repo *models.Repository) string {
	return "📁"
}

// getRunStatusIcon returns an icon based on run status
func (d *DashboardView) getRunStatusIcon(status models.RunStatus) string {
	switch status {
	case models.StatusQueued:
		return "⏳"
	case models.StatusInitializing:
		return "🔄"
	case models.StatusProcessing:
		return "⚙️"
	case models.StatusPostProcess:
		return "📝"
	case models.StatusDone:
		return "✅"
	case models.StatusFailed:
		return "❌"
	default:
		return "❓"
	}
}

// wrapText wraps text to fit within specified width
func (d *DashboardView) wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	currentLine := ""

	for _, word := range words {
		if len(currentLine) == 0 {
			currentLine = word
		} else if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

// wrapTextWithLimit wraps text to fit within width and max lines
func (d *DashboardView) wrapTextWithLimit(text string, width int, maxLines int) []string {
	if width <= 0 || maxLines <= 0 {
		return []string{}
	}

	// First wrap normally
	lines := d.wrapText(text, width)

	// If it fits within maxLines, return as is
	if len(lines) <= maxLines {
		return lines
	}

	// Truncate to maxLines with ellipsis
	result := lines[:maxLines-1]
	lastLine := lines[maxLines-1]
	if len(lastLine) > width-5 {
		lastLine = lastLine[:width-5]
	}
	result = append(result, lastLine+" (...)")

	return result
}

// applyItemHighlight applies the appropriate highlighting style to an item based on selection and focus state
func (d *DashboardView) applyItemHighlight(item string, isSelected bool, isFocused bool, maxWidth int) string {
	if isSelected {
		if isFocused {
			// Single blink: bright green briefly when clipboard manager is highlighting
			if d.clipboardManager.ShouldHighlight() {
				// Bright green flash
				return lipgloss.NewStyle().
					Width(maxWidth). // Use Width to ensure exact width
					MaxWidth(maxWidth).
					Inline(true).
					Background(lipgloss.Color("82")). // Bright green
					Foreground(lipgloss.Color("0")).  // Black text
					Bold(true).
					Render(item)
			} else {
				// Normal focused highlight
				return lipgloss.NewStyle().
					Width(maxWidth). // Use Width to ensure exact width
					MaxWidth(maxWidth).
					Inline(true).
					Background(lipgloss.Color("63")).
					Foreground(lipgloss.Color("255")).
					Render(item)
			}
		} else {
			// Selected but not focused
			return lipgloss.NewStyle().
				Width(maxWidth). // Use Width to ensure exact width
				MaxWidth(maxWidth).
				Inline(true).
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("255")).
				Render(item)
		}
	} else {
		// Non-selected items also need width constraint
		return lipgloss.NewStyle().
			Width(maxWidth). // Use Width to ensure exact width
			MaxWidth(maxWidth).
			Inline(true).
			Render(item)
	}
}
