package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/styles"
	"github.com/repobird/repobird-cli/internal/utils"
)

// renderHeader renders the header of the details view
func (v *RunDetailsView) renderHeader() string {
	statusIcon := styles.GetStatusIcon(string(v.run.Status))
	statusStyle := styles.GetStatusStyle(string(v.run.Status))
	status := statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, v.run.Status))

	idStr := v.run.GetIDString()
	if len(idStr) > 8 {
		idStr = idStr[:8]
	}
	title := fmt.Sprintf("Run #%s", idStr)
	if v.run.Title != "" {
		title += " - " + v.run.Title
	}

	// Truncate title if too long for terminal width
	if v.width > 25 && len(title) > v.width-20 {
		maxLen := v.width - 23
		if maxLen > 0 && maxLen < len(title) {
			title = title[:maxLen] + "..."
		}
	}

	header := styles.TitleStyle.MaxWidth(v.width).Render(title)

	if models.IsActiveStatus(string(v.run.Status)) {
		if v.pollingStatus {
			// Show active polling indicator
			pollingIndicator := styles.ProcessingStyle.Render(" [Fetching... " + v.spinner.View() + "]")
			header += pollingIndicator
		} else {
			// Show passive polling indicator
			pollingIndicator := styles.ProcessingStyle.Render(" [Monitoring ⟳]")
			header += pollingIndicator
		}
	}

	rightAlign := lipgloss.NewStyle().Align(lipgloss.Right).Width(v.width - lipgloss.Width(header))
	header += rightAlign.Render("Status: " + status)

	return header
}

// renderStatusBar renders the status bar at the bottom of the view
func (v *RunDetailsView) renderStatusBar() string {
	// Shorter options to fit better (removed logs functionality)
	options := "q:back j/k:nav y:copy Y:all r:refresh ?:help Q:quit"

	// Add URL opening hint if current selection has a URL
	if v.hasCurrentSelectionURL() {
		options = "o:url q:back j/k:nav y:copy Y:all r:refresh ?:help Q:quit"
	}

	// Determine if we're loading
	isLoadingData := v.loading

	// Set right content based on loading state
	rightContent := ""
	// Don't show any text when loading, just the spinner

	// Use unified status line system
	return v.statusLine.
		SetWidth(v.width).
		SetLeft("[DETAILS]").
		SetRight(rightContent).
		SetHelp(options).
		SetLoading(isLoadingData).
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

	// Debug: Log viewport rendering details
	debug.LogToFilef("DEBUG: renderContentWithCursor - viewportHeight=%d, contentWidth=%d, offset=%d\n", 
		viewportHeight, contentWidth, viewportOffset)

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
			highlightedLine := applyHighlightStyle(line, contentWidth, v.yankBlink, v.yankBlinkTime)
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
func applyHighlightStyle(line string, width int, yankBlink bool, yankBlinkTime time.Time) string {
	if yankBlink && !yankBlinkTime.IsZero() && time.Since(yankBlinkTime) < 250*time.Millisecond {
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
