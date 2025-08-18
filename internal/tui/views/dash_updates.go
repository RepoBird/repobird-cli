// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/utils"
)

// updateDetailLines updates the detail lines for the selected run
func (d *DashboardView) updateDetailLines() {
	// Save current selection before updating if we're in the details column
	if d.focusedColumn == 2 && d.selectedRunData != nil {
		runID := d.selectedRunData.GetIDString()
		if runID != "" {
			d.detailLineMemory[runID] = d.selectedDetailLine
		}
	}

	d.detailLines = []string{}
	d.detailLinesOriginal = []string{}
	d.selectedDetailLine = 0

	if d.selectedRunData == nil {
		return
	}

	run := d.selectedRunData
	// Calculate available width for text (accounting for padding)
	columnWidth := d.width / 3
	if columnWidth < 10 {
		columnWidth = 10
	}
	textWidth := columnWidth - 4 // Account for padding and borders
	if textWidth < 10 {
		textWidth = 10
	}

	// Helper function to truncate text to single line
	truncateLine := func(text string) string {
		if len(text) > textWidth {
			return text[:textWidth-3] + "..."
		}
		return text
	}

	// Helper to add both truncated and original lines
	addLine := func(text string) {
		d.detailLines = append(d.detailLines, truncateLine(text))
		d.detailLinesOriginal = append(d.detailLinesOriginal, text)
	}

	// Add single-line fields (truncated for display, original for copying)
	addLine(fmt.Sprintf("ID: %s", run.GetIDString()))
	addLine(fmt.Sprintf("Status: %s", string(run.Status)))
	addLine(fmt.Sprintf("Repository: %s", run.GetRepositoryName()))

	// Show run type - normalize API values to display values
	if run.RunType != "" {
		displayType := "Run"
		runTypeLower := strings.ToLower(run.RunType)
		if strings.Contains(runTypeLower, "plan") {
			displayType = "Plan"
		}
		addLine(fmt.Sprintf("Type: %s", displayType))
	}

	if run.Source != "" && run.Target != "" {
		addLine(fmt.Sprintf("Branch: %s → %s", run.Source, run.Target))
	}

	addLine(fmt.Sprintf("Created: %s", run.CreatedAt.Format("Jan 2 15:04")))
	addLine(fmt.Sprintf("Updated: %s", run.UpdatedAt.Format("Jan 2 15:04")))

	// Show PR URL if available
	if run.PrURL != nil && *run.PrURL != "" {
		addLine(fmt.Sprintf("PR URL: %s", *run.PrURL))
	}

	// Show trigger source if available
	if run.TriggerSource != nil && *run.TriggerSource != "" {
		addLine(fmt.Sprintf("Trigger: %s", *run.TriggerSource))
	}

	// Title - single line truncated
	if run.Title != "" {
		addLine("")
		addLine("Title:")
		addLine(run.Title)
	}

	// Description - single line truncated
	if run.Description != "" {
		addLine("")
		addLine("Description:")
		addLine(run.Description)
	}

	// Prompt - single line truncated
	if run.Prompt != "" {
		addLine("")
		addLine("Prompt:")
		addLine(run.Prompt)
	}

	// Error - single line truncated
	if run.Error != "" {
		addLine("")
		addLine("Error:")
		addLine(run.Error)
	}

	// Plan field - special handling (can be multi-line if space available)
	// This should be last so it can use remaining space
	if strings.Contains(strings.ToLower(run.RunType), "plan") && run.Status == models.StatusDone && run.Plan != "" {
		addLine("")
		addLine("Plan:")
		// For now, just show first line with ellipsis if there's more
		// The renderDetailsColumn will handle proper multi-line display
		lines := strings.Split(run.Plan, "\n")
		if len(lines) > 0 {
			// Store full plan in original, but truncate for display
			d.detailLinesOriginal[len(d.detailLinesOriginal)-1] = run.Plan // Replace last "Plan:" with full plan
			firstLine := truncateLine(lines[0])
			if len(lines) > 1 {
				firstLine = firstLine + " (...)"
			}
			d.detailLines[len(d.detailLines)-1] = firstLine // Update display version
		}
	}

	// Update the details viewport with new content
	d.updateDetailsViewportContent()

	// Restore saved selection for this run if available
	if runID := run.GetIDString(); runID != "" {
		if savedLine, exists := d.detailLineMemory[runID]; exists {
			// Ensure the saved selection is within bounds
			if savedLine >= 0 && savedLine < len(d.detailLines) {
				d.selectedDetailLine = savedLine
			}
		}
	}
}

// updateAllRunsListData converts runs data for the shared scrollable list component
func (d *DashboardView) updateAllRunsListData() {
	if d.allRunsList == nil {
		return
	}

	var items [][]string
	for _, run := range d.allRuns {
		if run == nil {
			continue
		}

		// Format row data: [ID, Repository, Status, Created]
		row := []string{
			run.ID,
			run.Repository,
			string(run.Status),
			run.CreatedAt.Format("2006-01-02 15:04"),
		}
		items = append(items, row)
	}

	d.allRunsList.SetItems(items)
}

// updateViewportSizes updates the viewport dimensions based on window size
func (d *DashboardView) updateViewportSizes() {
	if d.width == 0 || d.height == 0 {
		return
	}

	// Calculate column widths (accounting for borders)
	totalWidth := d.width - 6 // 3 columns * 2 border chars each
	leftWidth := totalWidth / 3
	centerWidth := totalWidth / 3
	rightWidth := totalWidth - leftWidth - centerWidth

	// Height for viewports (subtract title, borders, status line)
	viewportHeight := d.height - 7 // 2 for title, 2 for borders top/bottom, 1 for column title, 2 for status
	if viewportHeight < 5 {
		viewportHeight = 5
	}

	// Update viewport sizes
	// Width accounts for: border (2) + padding (2) = 4 total
	d.repoViewport.Width = leftWidth - 4
	d.repoViewport.Height = viewportHeight

	d.runsViewport.Width = centerWidth - 4
	d.runsViewport.Height = viewportHeight

	d.detailsViewport.Width = rightWidth - 4
	d.detailsViewport.Height = viewportHeight

	debug.LogToFilef("updateViewportSizes: terminal=%dx%d, cols=%d/%d/%d, viewports=%d/%d/%d\n",
		d.width, d.height, leftWidth, centerWidth, rightWidth,
		d.repoViewport.Width, d.runsViewport.Width, d.detailsViewport.Width)
}

// updateViewportContent updates the content of viewports when data changes
func (d *DashboardView) updateViewportContent() {
	// Update repositories viewport
	d.updateRepoViewportContent()

	// Update runs viewport
	d.updateRunsViewportContent()

	// Update details viewport
	d.updateDetailsViewportContent()
}

// updateRepoViewportContent updates the repository column viewport content
func (d *DashboardView) updateRepoViewportContent() {
	var items []string
	
	// Get filtered items if FZF is active
	var repos []models.Repository
	var filteredIndices []int
	if d.inlineFZF != nil && d.inlineFZF.IsActive() && d.fzfColumn == 0 {
		// Use filtered items from FZF
		filteredItems := d.inlineFZF.GetFilteredItems()
		for _, filteredItem := range filteredItems {
			// Find matching repository
			for i, repo := range d.repositories {
				if repo.Name == filteredItem {
					repos = append(repos, repo)
					filteredIndices = append(filteredIndices, i)
					break
				}
			}
		}
	} else {
		// Use all repositories
		repos = d.repositories
		for i := range d.repositories {
			filteredIndices = append(filteredIndices, i)
		}
	}
	
	for idx, repo := range repos {
		originalIdx := filteredIndices[idx]
		statusIcon := d.getRepositoryStatusIcon(&repo)
		baseItem := fmt.Sprintf("%s %s", statusIcon, repo.Name)

		// Calculate actual available width for text
		maxWidth := d.repoViewport.Width
		if maxWidth <= 0 {
			maxWidth = 30 // Fallback minimum
		}

		// Truncate using rune-safe method BEFORE styling
		item := baseItem
		runes := []rune(baseItem)
		if len(runes) > maxWidth {
			if maxWidth > 3 {
				item = string(runes[:maxWidth-3]) + "..."
			} else {
				item = "..."
			}
		}

		// Highlight selected repository (use filtered index for FZF mode)
		isSelected := false
		if d.inlineFZF != nil && d.inlineFZF.IsActive() && d.fzfColumn == 0 {
			isSelected = idx == d.inlineFZF.GetSelectedIndex()
		} else {
			isSelected = originalIdx == d.selectedRepoIdx
		}
		item = d.applyItemHighlight(item, isSelected, d.focusedColumn == 0, maxWidth)

		items = append(items, item)
	}

	if len(items) == 0 {
		// Show loading or empty state with proper highlighting
		emptyMsg := "No repositories"
		if d.loading {
			emptyMsg = "Loading repositories..."
		}

		// Apply highlighting if this column is focused and selected
		maxWidth := d.repoViewport.Width
		if maxWidth <= 0 {
			maxWidth = 30
		}

		if d.focusedColumn == 0 && d.selectedRepoIdx == 0 {
			// Apply focused highlight for better visibility
			emptyMsg = lipgloss.NewStyle().
				Width(maxWidth).
				MaxWidth(maxWidth).
				Inline(true).
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255")).
				Render(emptyMsg)
		} else {
			emptyMsg = lipgloss.NewStyle().
				Width(maxWidth).
				MaxWidth(maxWidth).
				Inline(true).
				Render(emptyMsg)
		}

		items = []string{emptyMsg}
	}

	content := strings.Join(items, "\n")
	d.repoViewport.SetContent(content)

	// Auto-scroll to keep selected item visible
	d.scrollToSelected(0)
}

// updateRunsViewportContent updates the runs column viewport content
func (d *DashboardView) updateRunsViewportContent() {
	var items []string

	// Calculate width for proper rendering
	maxWidth := d.runsViewport.Width
	if maxWidth <= 0 {
		maxWidth = 40 // Fallback minimum
	}

	if d.selectedRepo == nil {
		// No repository selected - show message with proper highlighting
		msg := "Select a repository"
		if d.loading {
			msg = "Loading runs..."
		}

		// Apply highlighting if this column is focused
		if d.focusedColumn == 1 && d.selectedRunIdx == 0 {
			msg = lipgloss.NewStyle().
				Width(maxWidth).
				MaxWidth(maxWidth).
				Inline(true).
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255")).
				Render(msg)
		} else if d.focusedColumn == 1 {
			// Column is focused but not this item
			msg = lipgloss.NewStyle().
				Width(maxWidth).
				MaxWidth(maxWidth).
				Inline(true).
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("255")).
				Render(msg)
		} else {
			msg = lipgloss.NewStyle().
				Width(maxWidth).
				MaxWidth(maxWidth).
				Inline(true).
				Render(msg)
		}

		items = []string{msg}
	} else {
		// Get filtered items if FZF is active
		var runs []*models.RunResponse
		var filteredIndices []int
		if d.inlineFZF != nil && d.inlineFZF.IsActive() && d.fzfColumn == 1 {
			// Use filtered items from FZF
			filteredItems := d.inlineFZF.GetFilteredItems()
			for _, filteredItem := range filteredItems {
				// Find matching run by parsing the formatted string
				for i, run := range d.filteredRuns {
					runID := run.GetIDString()
					title := run.Title
					if title == "" {
						title = "Untitled"
					}
					expectedItem := fmt.Sprintf("%s - %s", runID, title)
					if expectedItem == filteredItem {
						runs = append(runs, run)
						filteredIndices = append(filteredIndices, i)
						break
					}
				}
			}
		} else {
			// Use all runs
			runs = d.filteredRuns
			for i := range d.filteredRuns {
				filteredIndices = append(filteredIndices, i)
			}
		}
		
		for idx, run := range runs {
			originalIdx := filteredIndices[idx]
			statusIcon := d.getRunStatusIcon(run.Status)
			runID := run.GetIDString()
			title := run.Title
			if title == "" {
				title = "Untitled"
			}

			// Build the item with proper truncation
			// Format: "[icon] [id] - [title]"
			prefix := fmt.Sprintf("%s %s - ", statusIcon, runID)
			prefixRunes := []rune(prefix)
			prefixLen := len(prefixRunes)

			// Calculate remaining space for title
			remainingWidth := maxWidth - prefixLen
			if remainingWidth < 5 {
				// Not enough space, just truncate the whole thing
				item := prefix + title
				runes := []rune(item)
				if len(runes) > maxWidth {
					item = string(runes[:maxWidth-3]) + "..."
				}
				items = append(items, item)
				debug.LogToFilef("Run[%d]: Truncated whole, width=%d\n", originalIdx, maxWidth)
				continue
			}

			// Truncate title to fit
			titleRunes := []rune(title)
			if len(titleRunes) > remainingWidth {
				title = string(titleRunes[:remainingWidth-3]) + "..."
			}

			item := prefix + title

			// Final safety check
			finalRunes := []rune(item)
			if len(finalRunes) > maxWidth {
				item = string(finalRunes[:maxWidth-3]) + "..."
				debug.LogToFilef("Run[%d]: Final safety truncation triggered\n", originalIdx)
			}

			// Highlight selected run (use filtered index for FZF mode)
			isSelected := false
			if d.inlineFZF != nil && d.inlineFZF.IsActive() && d.fzfColumn == 1 {
				isSelected = idx == d.inlineFZF.GetSelectedIndex()
			} else {
				isSelected = originalIdx == d.selectedRunIdx
			}
			item = d.applyItemHighlight(item, isSelected, d.focusedColumn == 1, maxWidth)

			items = append(items, item)
		}

		if len(items) == 0 {
			msg := fmt.Sprintf("No runs for %s", d.selectedRepo.Name)

			// Apply highlighting if this column is focused
			if d.focusedColumn == 1 && d.selectedRunIdx == 0 {
				msg = lipgloss.NewStyle().
					Width(maxWidth).
					MaxWidth(maxWidth).
					Inline(true).
					Background(lipgloss.Color("63")).
					Foreground(lipgloss.Color("255")).
					Render(msg)
			} else if d.focusedColumn == 1 {
				msg = lipgloss.NewStyle().
					Width(maxWidth).
					MaxWidth(maxWidth).
					Inline(true).
					Background(lipgloss.Color("240")).
					Foreground(lipgloss.Color("255")).
					Render(msg)
			} else {
				msg = lipgloss.NewStyle().
					Width(maxWidth).
					MaxWidth(maxWidth).
					Inline(true).
					Render(msg)
			}

			items = []string{msg}
		}
	}

	content := strings.Join(items, "\n")
	d.runsViewport.SetContent(content)

	// Auto-scroll to keep selected item visible
	d.scrollToSelected(1)
}

// updateDetailsViewportContent updates the details column viewport content
func (d *DashboardView) updateDetailsViewportContent() {
	var displayLines []string

	// Calculate available content width
	contentWidth := d.detailsViewport.Width
	if contentWidth <= 0 {
		contentWidth = 30 // Fallback minimum
	}

	if d.selectedRunData == nil {
		msg := "Select a run"
		if d.loading {
			msg = "Loading details..."
		}

		// Apply highlighting if this column is focused
		if d.focusedColumn == 2 && d.selectedDetailLine == 0 {
			msg = lipgloss.NewStyle().
				Width(contentWidth).
				MaxWidth(contentWidth).
				Inline(true).
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255")).
				Render(msg)
		} else if d.focusedColumn == 2 {
			msg = lipgloss.NewStyle().
				Width(contentWidth).
				MaxWidth(contentWidth).
				Inline(true).
				Background(lipgloss.Color("240")).
				Foreground(lipgloss.Color("255")).
				Render(msg)
		} else {
			msg = lipgloss.NewStyle().
				Width(contentWidth).
				MaxWidth(contentWidth).
				Inline(true).
				Render(msg)
		}

		displayLines = []string{msg}
	} else {
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

			// Truncate displayLine to ensure it fits
			displayRunes := []rune(displayLine)
			if len(displayRunes) > contentWidth {
				displayLine = string(displayRunes[:contentWidth-3]) + "..."
			}

			// Apply width constraint using lipgloss to prevent overflow
			styledLine := displayLine

			if d.focusedColumn == 2 && i == d.selectedDetailLine {
				// Custom blinking: toggle between bright and normal colors
				if d.clipboardManager.ShouldHighlight() {
					// Bright green when visible
					styledLine = lipgloss.NewStyle().
						Width(contentWidth). // Use Width to ensure exact width
						MaxWidth(contentWidth).
						Inline(true).
						Background(lipgloss.Color("82")). // Bright green
						Foreground(lipgloss.Color("0")).  // Black text
						Bold(true).
						Render(displayLine)
				} else {
					// Normal highlight when not blinking
					styledLine = lipgloss.NewStyle().
						Width(contentWidth). // Use Width to ensure exact width
						MaxWidth(contentWidth).
						Inline(true).
						Background(lipgloss.Color("63")).
						Foreground(lipgloss.Color("255")).
						Render(displayLine)
				}
			} else {
				// Apply width constraint but no highlight
				styledLine = lipgloss.NewStyle().
					Width(contentWidth). // Use Width to ensure exact width
					MaxWidth(contentWidth).
					Inline(true).
					Render(displayLine)
			}

			displayLines = append(displayLines, styledLine)
		}
	}

	content := strings.Join(displayLines, "\n")
	d.detailsViewport.SetContent(content)

	// Auto-scroll to keep selected item visible
	d.scrollToSelected(2)
}

// Additional update helper methods

