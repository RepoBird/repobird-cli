package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/utils"
)

// renderTripleColumnLayout renders the main dashboard layout with three columns
func (d *DashboardView) renderTripleColumnLayout() string {
	// Calculate available height for columns
	// We have d.height total, minus:
	// - 2 for title (1 line + spacing)
	// - 1 for statusline
	availableHeight := d.height - 3
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}

	// Column widths - calculate based on terminal width
	// Each box renders 2 pixels wider than its set width, so subtract 6 total (2 per column)
	// to ensure they fit within terminal width
	totalWidth := d.width - 6 // Subtract 6 to account for the 2-pixel expansion per box
	leftWidth := totalWidth / 3
	centerWidth := totalWidth / 3
	rightWidth := totalWidth - leftWidth - centerWidth // Use remaining width

	// Ensure minimum widths
	if leftWidth < 10 {
		leftWidth = 10
	}
	if centerWidth < 10 {
		centerWidth = 10
	}
	if rightWidth < 10 {
		rightWidth = 10
	}

	// Make columns with rounded borders - use full available height
	// The Height() method in lipgloss includes borders in the total height
	columnHeight := availableHeight
	if columnHeight < 3 {
		columnHeight = 3
	}

	// Create column content with titles
	// Account for borders (2 chars for left/right, 2 for top/bottom)
	// Content width should be column width minus borders
	contentWidth1 := leftWidth - 2
	contentWidth2 := centerWidth - 2
	contentWidth3 := rightWidth - 2
	contentHeight := columnHeight - 2

	leftContent := d.renderRepositoriesColumn(contentWidth1, contentHeight)
	centerContent := d.renderRunsColumn(contentWidth2, contentHeight)
	rightContent := d.renderDetailsColumn(contentWidth3, contentHeight)

	// Create styles for columns
	// Width() and Height() in lipgloss include the border in the total dimensions
	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(columnHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))

	centerStyle := lipgloss.NewStyle().
		Width(centerWidth).
		Height(columnHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("33"))

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(columnHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	// Render each column
	leftBox := leftStyle.Render(leftContent)
	centerBox := centerStyle.Render(centerContent)
	rightBox := rightStyle.Render(rightContent)

	// Join columns horizontally - they should already fit the width exactly
	columns := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftBox,
		centerBox,
		rightBox,
	)

	finalWidth := lipgloss.Width(columns)

	// If columns still exceed terminal width (shouldn't happen with correct calculation)
	// Use PlaceHorizontal to constrain them
	if finalWidth > d.width {
		columns = lipgloss.PlaceHorizontal(d.width, lipgloss.Left, columns)
	}

	// Create statusline
	statusline := d.renderStatusLine("DASH")

	// The statusline should be placed at the bottom with proper spacing
	// Place the columns and statusline in the available space
	_ = lipgloss.Height(columns) // columnsHeight not used right now

	// Add notification line if there's a message to show
	var parts []string
	parts = append(parts, columns)

	parts = append(parts, statusline)

	// Use PlaceVertical to position the statusline at the bottom
	// The available height already accounts for title and statusline
	finalLayout := lipgloss.JoinVertical(lipgloss.Left, parts...)

	return finalLayout
}

// renderAllRunsLayout renders the all runs layout
func (d *DashboardView) renderAllRunsLayout() string {
	// Update data in shared scrollable list component
	d.updateAllRunsListData()
	// Use shared scrollable list component
	runListContent := d.allRunsList.View()

	// Create statusline
	statusline := d.renderStatusLine("RUNS")

	// Add notification above status line if there's a message
	var parts []string
	parts = append(parts, runListContent)
	parts = append(parts, statusline)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderRepositoriesLayout renders the repositories-only layout
func (d *DashboardView) renderRepositoriesLayout() string {
	// Render repositories table
	content := "" // d.renderRepositoriesTable() - method being refactored

	// Create statusline
	statusline := d.renderStatusLine("REPOS")

	// Add notification above status line if there's a message
	var parts []string
	parts = append(parts, content)
	parts = append(parts, statusline)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderRepositoriesColumn renders the left column with repositories
func (d *DashboardView) renderRepositoriesColumn(width, height int) string {
	// Create title with underline
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Width(width).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("63"))

	if d.focusedColumn == 0 {
		titleStyle = titleStyle.Foreground(lipgloss.Color("63"))
	} else {
		titleStyle = titleStyle.Foreground(lipgloss.Color("240"))
	}
	titleText := fmt.Sprintf("Repositories [%d]", len(d.repositories))
	title := titleStyle.Render(titleText)

	// Build items list
	var items []string
	for i, repo := range d.repositories {
		statusIcon := d.getRepositoryStatusIcon(&repo)
		item := fmt.Sprintf("%s %s", statusIcon, repo.Name)

		// Truncate if too long
		if len(item) > width-2 {
			item = item[:width-5] + "..."
		}

		// Highlight selected repository
		if i == d.selectedRepoIdx {
			if d.focusedColumn == 0 {
				// Single blink: bright green briefly when clipboard manager is highlighting
				if d.clipboardManager.ShouldHighlight() {
					// Bright green flash
					item = lipgloss.NewStyle().
						Width(width).
						Background(lipgloss.Color("82")). // Bright green
						Foreground(lipgloss.Color("0")).  // Black text
						Bold(true).
						Render(item)
				} else {
					// Normal focused highlight
					item = lipgloss.NewStyle().
						Width(width).
						Background(lipgloss.Color("63")).
						Foreground(lipgloss.Color("255")).
						Render(item)
				}
			} else {
				item = lipgloss.NewStyle().
					Width(width).
					Background(lipgloss.Color("240")).
					Foreground(lipgloss.Color("255")).
					Render(item)
			}
		}

		items = append(items, item)
	}

	if len(items) == 0 {
		items = []string{"No repositories"}
	}

	// Update viewport content if needed
	d.updateRepoViewportContent()

	// Calculate content height (subtract title height)
	contentHeight := height - 2

	// Render viewport content with padding
	contentStyle := lipgloss.NewStyle().
		Width(width).
		Height(contentHeight).
		Padding(0, 1)

	return lipgloss.JoinVertical(lipgloss.Left, title, contentStyle.Render(d.repoViewport.View()))
}

// renderRunsColumn renders the center column with runs for selected repository
func (d *DashboardView) renderRunsColumn(width, height int) string {
	// Create title with underline
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Width(width).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("33"))

	if d.focusedColumn == 1 {
		titleStyle = titleStyle.Foreground(lipgloss.Color("33"))
	} else {
		titleStyle = titleStyle.Foreground(lipgloss.Color("240"))
	}
	titleText := "Runs"
	if d.selectedRepo != nil && len(d.filteredRuns) > 0 {
		titleText = fmt.Sprintf("Runs [%d]", len(d.filteredRuns))
	}
	title := titleStyle.Render(titleText)

	var items []string
	if d.selectedRepo == nil {
		items = []string{"Select a repository"}
	} else {
		for i, run := range d.filteredRuns {
			statusIcon := d.getRunStatusIcon(run.Status)
			displayTitle := run.Title
			if displayTitle == "" {
				displayTitle = "Untitled Run"
			}

			// Truncate based on available width
			maxTitleLen := width - 5 // Account for icon and padding
			if len(displayTitle) > maxTitleLen {
				displayTitle = displayTitle[:maxTitleLen-3] + "..."
			}

			item := fmt.Sprintf("%s %s", statusIcon, displayTitle)

			// Highlight selected run
			if i == d.selectedRunIdx {
				if d.focusedColumn == 1 {
					// Custom blinking: toggle between bright and normal colors
					if d.clipboardManager.ShouldHighlight() {
						// Bright green when visible
						item = lipgloss.NewStyle().
							Width(width).
							Background(lipgloss.Color("82")). // Bright green
							Foreground(lipgloss.Color("0")).  // Black text
							Bold(true).
							Render(item)
					} else {
						// Normal focused highlight (no blinking)
						item = lipgloss.NewStyle().
							Width(width).
							Background(lipgloss.Color("33")).
							Foreground(lipgloss.Color("255")).
							Render(item)
					}
				} else {
					item = lipgloss.NewStyle().
						Width(width).
						Background(lipgloss.Color("240")).
						Foreground(lipgloss.Color("255")).
						Render(item)
				}
			}

			items = append(items, item)
		}

		if len(items) == 0 {
			items = []string{fmt.Sprintf("No runs for %s", d.selectedRepo.Name)}
		}
	}

	// Update viewport content if needed
	d.updateRunsViewportContent()

	// Calculate content height (subtract title height)
	contentHeight := height - 2

	// Render viewport content with padding
	contentStyle := lipgloss.NewStyle().
		Width(width).
		Height(contentHeight).
		Padding(0, 1)

	return lipgloss.JoinVertical(lipgloss.Left, title, contentStyle.Render(d.runsViewport.View()))
}

// renderDetailsColumn renders the right column with run details
func (d *DashboardView) renderDetailsColumn(width, height int) string {
	// Create title with underline
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Width(width).
		Padding(0, 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(lipgloss.Color("240"))

	if d.focusedColumn == 2 {
		titleStyle = titleStyle.Foreground(lipgloss.Color("63"))
	} else {
		titleStyle = titleStyle.Foreground(lipgloss.Color("240"))
	}
	title := titleStyle.Render("Run Details")

	var displayLines []string
	if d.selectedRunData == nil {
		displayLines = []string{"Select a run"}
	} else {
		// Calculate available content width
		contentWidth := width - 2 // Account for padding
		if contentWidth < 5 {
			contentWidth = 5
		}

		// Build lines with selection highlighting and proper width constraints
		for i, line := range d.detailLines {
			// Check if we should show RepoBird URL hint for ID line
			displayLine := line
			if d.focusedColumn == 2 && i == d.selectedDetailLine && i == 0 && d.selectedRunData != nil {
				// This is the ID line and it's selected, add URL hint if possible
				runID := d.selectedRunData.GetIDString()
				if utils.IsNonEmptyNumber(runID) {
					repobirdURL := utils.GenerateRepoBirdURL(runID)
					// Truncate URL to fit within available width, keeping the line readable
					maxURLLen := contentWidth - len(line) - 3 // 3 chars for " - "
					if maxURLLen > 10 {                       // Only show if we have reasonable space
						truncatedURL := repobirdURL
						if len(truncatedURL) > maxURLLen {
							truncatedURL = truncatedURL[:maxURLLen-3] + "..."
						}
						displayLine = line + " - " + truncatedURL
					}
				}
			}

			// Apply width constraint using lipgloss to prevent overflow
			styledLine := lipgloss.NewStyle().
				MaxWidth(contentWidth).
				Inline(true). // Force single line
				Render(displayLine)

			if d.focusedColumn == 2 && i == d.selectedDetailLine {
				// Custom blinking: toggle between bright and normal colors
				if d.clipboardManager.ShouldHighlight() {
					// Bright green when visible
					styledLine = lipgloss.NewStyle().
						MaxWidth(contentWidth).
						Inline(true).
						Background(lipgloss.Color("82")). // Bright green
						Foreground(lipgloss.Color("0")).  // Black text
						Bold(true).
						Render(displayLine)
				} else {
					// Normal focused highlight (no blinking)
					styledLine = lipgloss.NewStyle().
						MaxWidth(contentWidth).
						Inline(true).
						Background(lipgloss.Color("63")).
						Foreground(lipgloss.Color("255")).
						Render(displayLine)
				}
			}
			displayLines = append(displayLines, styledLine)
		}

		// Special handling for plan field if it's the last item
		// Calculate remaining vertical space
		contentHeight := height - 2 // Subtract title height
		usedLines := len(displayLines)
		remainingLines := contentHeight - usedLines

		// If we have a plan field and remaining space, expand it
		if d.selectedRunData != nil &&
			strings.Contains(strings.ToLower(d.selectedRunData.RunType), "plan") &&
			d.selectedRunData.Status == models.StatusDone &&
			d.selectedRunData.Plan != "" &&
			remainingLines > 0 {
			// Find the plan line (should be last)
			for i := len(d.detailLines) - 1; i >= 0; i-- {
				if strings.HasPrefix(d.detailLines[i], "Plan:") || (i > 0 && d.detailLines[i-1] == "Plan:") {
					// Replace the truncated plan with wrapped version
					wrapped := d.wrapTextWithLimit(d.selectedRunData.Plan, contentWidth, remainingLines)
					if len(wrapped) > 0 {
						// Remove the truncated plan line
						if i < len(displayLines) {
							displayLines = displayLines[:i]
						}
						// Add wrapped lines
						for _, wLine := range wrapped {
							styledLine := lipgloss.NewStyle().
								MaxWidth(contentWidth).
								Render(wLine)
							displayLines = append(displayLines, styledLine)
						}
					}
					break
				}
			}
		}
	}

	// Update viewport content if needed
	d.updateDetailsViewportContent()

	// Calculate content height (subtract title height)
	contentHeight := height - 2

	// Render viewport content with padding
	contentStyle := lipgloss.NewStyle().
		Width(width).
		Height(contentHeight).
		Padding(0, 1)

	return lipgloss.JoinVertical(lipgloss.Left, title, contentStyle.Render(d.detailsViewport.View()))
}

// renderStatusLine renders the universal status line
func (d *DashboardView) renderStatusLine(layoutName string) string {
	// Create formatter for consistent formatting
	formatter := components.NewStatusFormatter(layoutName, d.width)

	// Data freshness indicator - removed to clean up status line
	dataInfo := ""
	isLoadingData := d.loading || d.initializing

	// Debug logging for refresh state
	if d.loading && !d.initializing {
		debug.LogToFilef("🔄 STATUS: Rendering statusline during REFRESH - loading=%t initializing=%t 🔄\n", d.loading, d.initializing)
	}

	// Show refresh indicator during refresh (when loading but not initializing)
	if d.loading && !d.initializing {
		dataInfo = "" // Empty but loading spinner will still show
		debug.LogToFilef("🔄 STATUS: Refresh state - dataInfo empty but spinner should animate 🔄\n")
	}

	// Format left content consistently
	leftContent := formatter.FormatViewName()

	// Handle URL selection prompt with yellow background
	if d.showURLSelectionPrompt {
		promptHelp := "Open URL: (o)RepoBird (g)GitHub [ESC]cancel"
		yellowStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("220")).
			Foreground(lipgloss.Color("232")).
			Padding(0, 1)

		return d.statusLine.
			SetWidth(d.width).
			SetLeft(leftContent).
			SetRight(dataInfo).
			SetHelp(promptHelp).
			SetStyle(yellowStyle).
			SetLoading(isLoadingData).
			Render()
	}

	// Compact help text
	shortHelp := "n:new f:fuzzy s:status y:copy ?:docs r:refresh q:quit"

	// Add URL opening hint if current selection has a URL
	if d.hasCurrentSelectionURL() {
		shortHelp = "o:open-url " + shortHelp
	}

	// Use the existing status line instance that receives spinner updates
	// Format help text based on available space (like StandardStatusLine does)
	formattedHelp := formatter.FormatHelp(leftContent, dataInfo, shortHelp)

	return d.statusLine.
		SetWidth(d.width).
		SetLeft(leftContent).
		SetRight(dataInfo).
		SetHelp(formattedHelp).
		SetLoading(isLoadingData).
		ResetStyle().
		Render()
}
