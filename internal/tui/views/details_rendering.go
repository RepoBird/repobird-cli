// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/styles"
	"github.com/repobird/repobird-cli/internal/utils"
)

// renderStatusBar renders the status bar at the bottom of the view
func (v *RunDetailsView) renderStatusBar() string {
	// Create formatter for consistent formatting
	formatter := components.NewStatusFormatter("DETAILS", v.width)

	// Shorter options to fit better (removed logs functionality)
	options := "[h]back [q]dashboard j/k:nav y:copy Y:all r:refresh ?:help Q:quit"

	// Add URL opening hint if current selection has a URL
	if v.hasCurrentSelectionURL() {
		options = "o:url [h]back [q]dashboard j/k:nav y:copy Y:all r:refresh ?:help Q:quit"
	}

	// Format left content consistently
	leftContent := formatter.FormatViewName()

	// Create status line using formatter
	statusLine := formatter.StandardStatusLine(leftContent, "", options)
	return statusLine.
		SetLoading(v.loading).
		Render()
}

// hasCurrentSelectionURL checks if the current selection contains a URL
func (v *RunDetailsView) hasCurrentSelectionURL() bool {
	if v.navigationMode && v.selectedRow >= 0 && v.selectedRow < len(v.fieldValues) {
		fieldValue := v.fieldValues[v.selectedRow]
		return utils.IsURL(fieldValue)
	}
	// Fallback to current line
	currentLine := v.getCurrentLine()
	return utils.IsURL(currentLine)
}

// renderContentWithCursor renders the content with a visible row selector
func (v *RunDetailsView) renderContentWithCursor() []string {
	if v.showLogs {
		// For logs view, just return the viewport content as-is
		return strings.Split(v.viewport.View(), "\n")
	}

	// Get all content lines
	allLines := strings.Split(v.fullContent, "\n")
	if len(allLines) == 0 {
		return []string{}
	}

	// Calculate viewport bounds
	viewportHeight := v.viewport.Height
	if viewportHeight <= 0 {
		viewportHeight = v.height - 6 // Fallback calculation
	}

	// Get the current viewport offset
	viewportOffset := v.viewport.YOffset

	// Determine which lines are visible
	visibleLines := []string{}
	contentWidth := v.viewport.Width
	if contentWidth <= 0 {
		contentWidth = v.width - 6 // Account for border (2) + padding (4)
	}

	for i := viewportOffset; i < len(allLines) && i < viewportOffset+viewportHeight; i++ {
		line := allLines[i]

		// Truncate line if too long
		lineRunes := []rune(line)
		if len(lineRunes) > contentWidth {
			line = string(lineRunes[:contentWidth-3]) + "..."
		}

		// Check if this line should be highlighted
		shouldHighlight := false
		if v.navigationMode && v.selectedRow >= 0 && v.selectedRow < len(v.fieldRanges) {
			fieldRange := v.fieldRanges[v.selectedRow]
			if i >= fieldRange[0] && i <= fieldRange[1] {
				shouldHighlight = true
			}
		}

		if shouldHighlight {
			// Apply highlight style using full available width
			highlightedLine := applyHighlightStyle(line, contentWidth, v.clipboardManager.ShouldHighlight())
			visibleLines = append(visibleLines, highlightedLine)
		} else {
			// Non-selected lines - use full available width
			styledLine := lipgloss.NewStyle().
				MaxWidth(contentWidth).
				Inline(true).
				Render(line)
			visibleLines = append(visibleLines, styledLine)
		}
	}

	return visibleLines
}

// applyHighlightStyle applies highlighting to a line
func applyHighlightStyle(line string, width int, shouldBlink bool) string {
	if shouldBlink {
		// Bright green flash for copy feedback
		highlightStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("82")). // Bright green
			Foreground(lipgloss.Color("0")).  // Black text
			Bold(true).
			Width(width).
			MaxWidth(width).
			Inline(true)
		return highlightStyle.Render(line)
	}

	// Normal focused highlight (matching dashboard style)
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("255")).
		Width(width).
		MaxWidth(width).
		Inline(true)
	return highlightStyle.Render(line)
}

// createHighlightedContent creates content with the selected field highlighted
func (v *RunDetailsView) createHighlightedContent(lines []string) string {
	if v.selectedRow < 0 || v.selectedRow >= len(v.fieldRanges) {
		return v.fullContent
	}

	// Get the range of lines for the selected field
	fieldRange := v.fieldRanges[v.selectedRow]
	startLine := fieldRange[0]
	endLine := fieldRange[1]

	var result strings.Builder
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("238")).
		Foreground(lipgloss.Color("15"))

	for i, line := range lines {
		if i >= startLine && i <= endLine {
			// Highlight all lines in the field range
			result.WriteString(highlightStyle.Render(line))
		} else {
			result.WriteString(line)
		}
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// updateStatusHistory updates the status history display
func (v *RunDetailsView) updateStatusHistory(status string, isPolling bool) {
	timestamp := time.Now().Format("15:04:05")
	var entry string

	if isPolling {
		// Show polling indicator
		entry = fmt.Sprintf("[%s] 🔄 %s", timestamp, status)
	} else {
		// Regular status update - only add if different from last non-polling status
		if len(v.statusHistory) > 0 {
			// Find last non-polling status
			for i := len(v.statusHistory) - 1; i >= 0; i-- {
				if !strings.Contains(v.statusHistory[i], "🔄") && strings.Contains(v.statusHistory[i], status) {
					// Same status as before, don't add duplicate
					return
				}
			}
		}
		statusIcon := styles.GetStatusIcon(status)
		entry = fmt.Sprintf("[%s] %s %s", timestamp, statusIcon, status)
	}

	v.statusHistory = append(v.statusHistory, entry)

	// Keep history size reasonable
	if len(v.statusHistory) > 50 {
		v.statusHistory = v.statusHistory[len(v.statusHistory)-50:]
	}
}

// formatDurationDetails formats a duration into a human-readable string
func formatDurationDetails(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// Message types for clipboard and status feedback
// Message types are defined in dashboard_messages.go and list.go
