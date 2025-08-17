// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
)

// truncateString truncates a string to the specified width with ellipsis
func (d *DashboardView) truncateString(s string, maxWidth int) string {
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

// formatTimeAgo formats time in a human-readable way
func (d *DashboardView) formatTimeAgo(t time.Time) string {
	if t.IsZero() {
		return "Never"
	}

	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "now"
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
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

// renderRepositoriesTable renders a table of repositories with real data
func (d *DashboardView) renderRepositoriesTable() string {
	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	header := fmt.Sprintf("%-25s %-8s %-8s %-10s %-8s %-15s",
		"Repository", "Total", "Running", "Completed", "Failed", "Last Activity")

	var rows []string
	rows = append(rows, headerStyle.Render(header))
	rows = append(rows, strings.Repeat("-", d.width-4))

	for _, repo := range d.repositories {
		statusIcon := d.getRepositoryStatusIcon(&repo)
		repoName := fmt.Sprintf("%s %s", statusIcon, repo.Name)
		lastActivity := d.formatTimeAgo(repo.LastActivity)

		row := fmt.Sprintf("%-25s %-8d %-8d %-10d %-8d %-15s",
			repoName,
			repo.RunCounts.Total,
			repo.RunCounts.Running,
			repo.RunCounts.Completed,
			repo.RunCounts.Failed,
			lastActivity)

		rows = append(rows, row)
	}

	if len(d.repositories) == 0 {
		rows = append(rows, "No repositories found")
	}

	return strings.Join(rows, "\n")
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
