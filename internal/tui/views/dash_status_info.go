package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// initializeStatusInfoFields initializes the selectable fields for the status info overlay
func (d *DashboardView) initializeStatusInfoFields() {
	d.statusInfoFields = []string{}
	d.statusInfoFieldLines = []int{}
	d.statusInfoKeys = []string{}
	d.statusInfoSelectedRow = 0

	lineNum := 0

	// User Info fields
	if d.userInfo != nil {
		if d.userInfo.Name != "" {
			d.statusInfoKeys = append(d.statusInfoKeys, "Name:")
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.Name)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}
		if d.userInfo.Email != "" {
			d.statusInfoKeys = append(d.statusInfoKeys, "Email:")
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.Email)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}
		if d.userInfo.GithubUsername != "" {
			d.statusInfoKeys = append(d.statusInfoKeys, "GitHub:")
			d.statusInfoFields = append(d.statusInfoFields, d.userInfo.GithubUsername)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++
		}

		// Plan info
		lineNum++ // Skip one line for Plan section
		tierDisplay := strings.Title(strings.ToLower(d.userInfo.Tier))
		if tierDisplay == "" {
			tierDisplay = "Basic"
		}
		d.statusInfoKeys = append(d.statusInfoKeys, "Account Tier:")
		d.statusInfoFields = append(d.statusInfoFields, tierDisplay)
		d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
		lineNum++

		// Usage info based on plan type
		upperTier := strings.ToUpper(d.userInfo.Tier)
		if upperTier == "FREE" || upperTier == "BASIC" {
			// Show runs remaining for usage-based plans
			var runsRemaining string
			if d.userInfo.TotalRuns > 0 {
				remaining := d.userInfo.RemainingRuns
				if remaining < 0 {
					remaining = 0
				}
				runsRemaining = fmt.Sprintf("%d / %d", remaining, d.userInfo.TotalRuns)
			} else {
				runsRemaining = "Unknown"
			}
			d.statusInfoKeys = append(d.statusInfoKeys, "Runs Remaining:")
			d.statusInfoFields = append(d.statusInfoFields, runsRemaining)
			d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
			lineNum++

			// Also show percentage usage if we have the data
			if d.userInfo.TotalRuns > 0 {
				usedRuns := d.userInfo.TotalRuns - d.userInfo.RemainingRuns
				percentage := float64(usedRuns) / float64(d.userInfo.TotalRuns) * 100

				var usageValue string
				if percentage >= 90 {
					usageValue = fmt.Sprintf("%.1f%% ⚠️", percentage)
				} else if percentage >= 75 {
					usageValue = fmt.Sprintf("%.1f%% ⚡", percentage)
				} else {
					usageValue = fmt.Sprintf("%.1f%% ✅", percentage)
				}

				d.statusInfoKeys = append(d.statusInfoKeys, "Usage:")
				d.statusInfoFields = append(d.statusInfoFields, usageValue)
				d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
				lineNum++
			}
		} else if upperTier == "PRO" {
			// Show percentage for PRO plans
			if d.userInfo.TotalRuns > 0 {
				usedRuns := d.userInfo.TotalRuns - d.userInfo.RemainingRuns
				percentage := float64(usedRuns) / float64(d.userInfo.TotalRuns) * 100
				d.statusInfoKeys = append(d.statusInfoKeys, "Usage:")
				d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%.1f%%", percentage))
				d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
				lineNum++
			} else {
				d.statusInfoKeys = append(d.statusInfoKeys, "Usage:")
				d.statusInfoFields = append(d.statusInfoFields, "Unlimited")
				d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
				lineNum++
			}
		}
	}

	// System info
	lineNum++ // Skip one line for System section
	d.statusInfoKeys = append(d.statusInfoKeys, "Repositories:")
	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%d", len(d.repositories)))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++
	d.statusInfoKeys = append(d.statusInfoKeys, "Total Runs:")
	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("%d", len(d.allRuns)))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++

	// Run status breakdown
	var running, completed, failed int
	for _, run := range d.allRuns {
		switch run.Status {
		case "RUNNING", "PENDING":
			running++
		case "DONE":
			completed++
		case "FAILED", "CANCELLED":
			failed++
		}
	}
	d.statusInfoKeys = append(d.statusInfoKeys, "Run Status:")
	d.statusInfoFields = append(d.statusInfoFields, fmt.Sprintf("🔄 %d  ✅ %d  ❌ %d", running, completed, failed))
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++

	// Last refresh time if available
	if !d.lastDataRefresh.IsZero() {
		refreshText := fmt.Sprintf("%s ago", time.Since(d.lastDataRefresh).Truncate(time.Second))
		d.statusInfoKeys = append(d.statusInfoKeys, "Last Refresh:")
		d.statusInfoFields = append(d.statusInfoFields, refreshText)
		d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
		lineNum++
	}

	// API connection info
	lineNum++ // Skip one line for API section
	d.statusInfoKeys = append(d.statusInfoKeys, "API Endpoint:")
	d.statusInfoFields = append(d.statusInfoFields, d.client.GetAPIEndpoint())
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)
	lineNum++
	d.statusInfoKeys = append(d.statusInfoKeys, "Status:")
	d.statusInfoFields = append(d.statusInfoFields, "Connected ✅")
	d.statusInfoFieldLines = append(d.statusInfoFieldLines, lineNum)

	// Ensure we have at least one field selected
	if len(d.statusInfoFields) > 0 {
		d.statusInfoSelectedRow = 0
	}
}

// handleStatusInfoNavigation handles navigation within the status info overlay
func (d *DashboardView) handleStatusInfoNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyDown, tea.KeyRunes:
		if msg.String() == "j" {
			if d.statusInfoSelectedRow < len(d.statusInfoFields)-1 {
				d.statusInfoSelectedRow++
				// Reset horizontal scroll when moving to a new row
				d.statusInfoKeyOffset = 0
				d.statusInfoValueOffset = 0
			}
		} else if msg.String() == "k" {
			if d.statusInfoSelectedRow > 0 {
				d.statusInfoSelectedRow--
				// Reset horizontal scroll when moving to a new row
				d.statusInfoKeyOffset = 0
				d.statusInfoValueOffset = 0
			}
		} else if msg.String() == "g" {
			d.statusInfoSelectedRow = 0
		} else if msg.String() == "G" {
			if len(d.statusInfoFields) > 0 {
				d.statusInfoSelectedRow = len(d.statusInfoFields) - 1
				// Reset horizontal scroll
				d.statusInfoKeyOffset = 0
				d.statusInfoValueOffset = 0
			}
		} else if msg.String() == "s" {
			// Exit status info overlay with 's'
			d.showStatusInfo = false
			return d, nil
		} else if msg.String() == "y" {
			// Copy current field to clipboard
			if d.statusInfoSelectedRow >= 0 && d.statusInfoSelectedRow < len(d.statusInfoFields) {
				var textToCopy string
				if d.statusInfoFocusColumn == 0 && d.statusInfoSelectedRow < len(d.statusInfoKeys) {
					// Copy the key (without the colon)
					textToCopy = strings.TrimSuffix(d.statusInfoKeys[d.statusInfoSelectedRow], ":")
				} else {
					// Copy the value
					textToCopy = d.statusInfoFields[d.statusInfoSelectedRow]
				}

				if err := d.copyToClipboard(textToCopy); err == nil {
					// Show success message temporarily
					d.copiedMessage = fmt.Sprintf("Copied: %s", textToCopy)
					if len(d.copiedMessage) > 50 {
						d.copiedMessage = d.copiedMessage[:47] + "..."
					}
					d.copiedMessageTime = time.Now()

					// Start the blink animation using clipboard manager
					cmd := d.copyToClipboard(textToCopy)
					if cmd != nil {
						return d, tea.Batch(
							cmd,
							d.startMessageClearTimer(2*time.Second),
						)
					}
					return d, d.startMessageClearTimer(2*time.Second)
				}
			}
		}
	case tea.KeyUp:
		if d.statusInfoSelectedRow > 0 {
			d.statusInfoSelectedRow--
			// Reset horizontal scroll when moving to a new row
			d.statusInfoKeyOffset = 0
			d.statusInfoValueOffset = 0
		}
	case tea.KeyLeft:
		if d.statusInfoFocusColumn == 1 {
			// Move from value column to key column
			d.statusInfoFocusColumn = 0
		} else {
			// Scroll key column left
			if d.statusInfoKeyOffset > 0 {
				d.statusInfoKeyOffset--
			}
		}
	case tea.KeyRight:
		if d.statusInfoFocusColumn == 0 {
			// Move from key column to value column
			d.statusInfoFocusColumn = 1
		} else {
			// Scroll value column right
			if d.statusInfoSelectedRow >= 0 && d.statusInfoSelectedRow < len(d.statusInfoFields) {
				value := d.statusInfoFields[d.statusInfoSelectedRow]
				valueMaxWidth := 40 // Available width for value column

				// Debug logging
				debug.LogToFilef("DEBUG: StatusInfo scroll check - Row %d, Value len=%d, MaxWidth=%d, Offset=%d\n",
					d.statusInfoSelectedRow, len(value), valueMaxWidth, d.statusInfoValueOffset)

				// Only scroll if there's more content to show
				if len(value) > d.statusInfoValueOffset+valueMaxWidth {
					d.statusInfoValueOffset++
					debug.LogToFilef("DEBUG: Scrolling value to offset %d\n", d.statusInfoValueOffset)
				}
			}
		}
	case tea.KeyEsc:
		// Exit status info overlay
		d.showStatusInfo = false
		return d, nil
	}

	return d, nil
}

// handleHelpNavigation handles keyboard navigation in the help overlay
func (d *DashboardView) handleHelpNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle special keys for closing help
	switch msg.String() {
	case "?", "q", "b", "escape":
		// Close the help overlay
		d.showDocs = false
		return d, nil
	case "Q":
		// Force quit
		_ = d.cache.SaveToDisk()
		d.cache.Stop()
		return d, tea.Quit
	}

	// Pass other keys to the help view
	updatedHelp, helpCmd := d.helpView.Update(msg)
	d.helpView = updatedHelp
	return d, helpCmd
}

// renderHelp renders the help overlay using the scrollable help view
func (d *DashboardView) renderHelp() string {
	// Set the size for the help view
	d.helpView.SetSize(d.width, d.height)
	// Return the rendered help view
	return d.helpView.View()
}

// renderDocsOld renders the documentation overlay - DEPRECATED (kept for reference)
//
//nolint:unused
func (d *DashboardView) renderDocsOld() string {
	// Calculate box dimensions - leave room for statusline at bottom
	boxWidth := d.width - 4   // Leave 2 chars margin on each side
	boxHeight := d.height - 3 // Leave room for statusline at bottom

	// Ensure minimum dimensions
	if boxWidth < 60 {
		boxWidth = 60
	}

	// Box style with rounded border
	boxStyle := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	// Title bar (inside the box)
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("63")).
		Width(boxWidth-2). // Account for border
		Align(lipgloss.Center).
		Padding(0, 1)

	// Get current page title
	pageTitles := []string{
		"Basic Navigation",
		"Fuzzy Search (FZF)",
		"View Controls",
		"Clipboard Operations",
		"Create Run Form",
		"Dashboard Layout",
		"Tips & Tricks",
		"Quick Reference",
	}

	title := titleStyle.Render(fmt.Sprintf("Documentation - %s", pageTitles[d.docsCurrentPage]))

	// Content styles
	contentStyle := lipgloss.NewStyle().
		Width(boxWidth-2). // Account for border
		Padding(1, 2)

	// Define documentation pages with proper truncation
	pages := d.getDocsPages()
	currentPage := pages[d.docsCurrentPage]

	// Ensure selected row is within bounds
	if d.docsSelectedRow >= len(currentPage) {
		d.docsSelectedRow = len(currentPage) - 1
	}
	if d.docsSelectedRow < 0 {
		d.docsSelectedRow = 0
	}

	// Render content lines with selection highlighting
	var contentLines []string
	maxContentWidth := boxWidth - 6 // Account for border (2) + padding (4)

	for i, row := range currentPage {
		// Truncate long lines to prevent layout issues
		truncatedRow := d.truncateString(row, maxContentWidth)

		if i == d.docsSelectedRow {
			// Highlight selected row
			highlightStyle := lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255")).
				Width(maxContentWidth)
			contentLines = append(contentLines, highlightStyle.Render(truncatedRow))
		} else {
			// Normal row
			normalStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Width(maxContentWidth)
			contentLines = append(contentLines, normalStyle.Render(truncatedRow))
		}
	}

	// Join content lines
	content := contentStyle.Render(strings.Join(contentLines, "\n"))

	// Page indicator at bottom
	indicatorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(boxWidth - 2).
		Align(lipgloss.Center).
		MarginTop(1)

	// Page dots
	var dots []string
	for i := 0; i < len(pageTitles); i++ {
		if i == d.docsCurrentPage {
			dots = append(dots, "●")
		} else {
			dots = append(dots, "○")
		}
	}

	pageIndicator := indicatorStyle.Render(
		fmt.Sprintf("Page %d/%d  %s  (1-8: jump)",
			d.docsCurrentPage+1,
			len(pageTitles),
			strings.Join(dots, " ")))

	// Calculate remaining height for spacing
	innerHeight := boxHeight - 2 // Account for border
	titleHeight := lipgloss.Height(title)
	contentHeight := lipgloss.Height(content)
	indicatorHeight := lipgloss.Height(pageIndicator)
	usedHeight := titleHeight + contentHeight + indicatorHeight
	remainingHeight := innerHeight - usedHeight

	spacing := ""
	if remainingHeight > 0 {
		spacing = strings.Repeat("\n", remainingHeight)
	}

	// Join everything inside the box
	innerContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		spacing,
		pageIndicator,
	)

	// Wrap in the box
	boxedContent := boxStyle.Render(innerContent)

	// Center the box on screen (leaving room for statusline)
	centeredBox := lipgloss.Place(d.width, d.height-1, lipgloss.Center, lipgloss.Center, boxedContent)

	// Create the statusline
	shortHelp := "[h/l]pages [j/k]navigate [y]copy [1-8]jump [?/q/b/ESC]back [Q]uit"

	// Show copy message if active
	statusText := shortHelp
	if d.copiedMessage != "" && time.Since(d.copiedMessageTime) < 2*time.Second {
		statusText = d.copiedMessage
	}

	statusLine := d.statusLine.
		SetWidth(d.width).
		SetLeft("[DOCS]").
		SetRight(fmt.Sprintf("Page %d/%d", d.docsCurrentPage+1, len(pageTitles))).
		SetHelp(statusText).
		Render()

	// Join the centered box and statusline
	return lipgloss.JoinVertical(lipgloss.Left, centeredBox, statusLine)
}

// getDocsPages returns the documentation content for each page - DEPRECATED (kept for reference)
//
//nolint:unused
func (d *DashboardView) getDocsPages() [][]string {
	return [][]string{
		// Page 1: Basic Navigation
		{
			"↑/↓, j/k     Move up/down in current column",
			"←/→, h/l     Move between columns",
			"Tab          Cycle through columns",
			"Enter        Select item and move to next column",
			"Backspace    Move to previous column",
			"g            Jump to first item",
			"G            Jump to last item",
			"gg           Jump to top (vim-style double tap)",
			"Ctrl+u       Page up",
			"Ctrl+d       Page down",
		},
		// Page 2: Fuzzy Search (FZF)
		{
			"f            Activate FZF mode on current column",
			"Type         Filter items in real-time",
			"↑/↓          Navigate filtered items",
			"Ctrl+j/k     Alternative navigation in FZF",
			"Enter        Select item and proceed",
			"ESC          Cancel FZF mode",
			"",
			"In Create View:",
			"Ctrl+F       FZF for repository (insert mode)",
			"f            FZF for repository (normal mode)",
		},
		// Page 3: View Controls
		{
			"n            Create new run",
			"b            Bulk runs (multiple at once)",
			"s            Show status/user info overlay",
			"r            Refresh data",
			"o            Open URL (when available)",
			"?            Toggle help/documentation",
			"q            Go back/quit (context-aware)",
			"Q            Force quit from anywhere",
			"ESC, b       Alternative back navigation",
		},
		// Page 4: Clipboard Operations
		{
			"y            Copy current selection to clipboard",
			"Y            Copy all content (details view)",
			"",
			"Visual Feedback:",
			"Green flash  Successful copy animation",
			"Status msg   Shows what was copied",
			"",
			"Tip: All selectable fields support copying",
		},
		// Page 5: Create Run Form
		{
			"Normal Mode:",
			"i, Enter     Enter insert mode",
			"j/k          Navigate fields",
			"ESC (2x)     Return to dashboard",
			"",
			"Insert Mode:",
			"Tab/Shift+Tab Navigate between fields",
			"ESC          Switch to normal mode",
			"Ctrl+S       Submit run",
			"Ctrl+L       Clear all fields",
			"Ctrl+X       Clear current field",
			"Ctrl+F       Repository fuzzy search",
		},
		// Page 6: Dashboard Layout
		{
			"Left Column  Repositories with active runs",
			"Middle       Runs for selected repository",
			"Right        Details for selected run",
			"",
			"Status Icons:",
			"🟢           Success",
			"🔵           Running",
			"🟡           Pending",
			"🔴           Failed",
			"⚪           Unknown",
		},
		// Page 7: Tips & Tricks
		{
			"Quick Find   Use 'f' instead of scrolling",
			"Fast Nav     Enter drills down, Backspace goes up",
			"Context      'q' behavior changes by view",
			"Memory       Recently used repos saved",
			"Smart Icons  📁 current, 🔄 history, ✏️ edited",
			"",
			"Pro Tip: Chain 'f' + Enter for quick access",
		},
		// Page 8: Quick Reference
		{
			"Navigation   j/k h/l Tab Enter Backspace",
			"Search       f (fuzzy) / (search)",
			"Actions      n (new) r (refresh) s (status)",
			"Clipboard    y (copy) Y (copy all)",
			"View Control ? (help) q (back) Q (quit)",
			"",
			"Vim Commands gg G Ctrl+u Ctrl+d",
			"Form Submit  Ctrl+S",
		},
	}
}

// renderStatusInfo renders the status/user info overlay
func (d *DashboardView) renderStatusInfo() string {
	if !d.showStatusInfo {
		return ""
	}

	// Use the status info fields that were prepared in initializeStatusInfoFields
	if len(d.statusInfoFields) == 0 {
		return "Status info not available"
	}

	// Create a simple list view for now
	var content strings.Builder
	content.WriteString("User Info\n")
	content.WriteString("=========\n\n")

	for i, field := range d.statusInfoFields {
		if i < len(d.statusInfoKeys) {
			content.WriteString(fmt.Sprintf("%s %s\n", d.statusInfoKeys[i], field))
		}
	}

	return content.String()
}
