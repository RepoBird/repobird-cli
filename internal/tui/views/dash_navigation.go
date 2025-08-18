// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/utils"
)

// findNextNonEmptyLine finds the next non-empty line starting from current index
func (d *DashboardView) findNextNonEmptyLine(startIdx int, direction int) int {
	if len(d.detailLines) == 0 {
		return startIdx
	}

	idx := startIdx
	for {
		idx += direction

		// Check bounds
		if idx < 0 {
			return startIdx // No non-empty line found upward
		}
		if idx >= len(d.detailLines) {
			return startIdx // No non-empty line found downward
		}

		// Check if line is non-empty
		if !d.isEmptyLine(d.detailLines[idx]) {
			return idx
		}

		// Prevent infinite loop (shouldn't happen but safety check)
		if idx == 0 && direction < 0 {
			return startIdx
		}
		if idx == len(d.detailLines)-1 && direction > 0 {
			return startIdx
		}
	}
}

// handleMillerColumnsNavigation handles navigation in the Miller Columns layout
// handleMillerColumnsNavigation handles navigation keys for Miller columns layout
// This function delegates to smaller, focused handlers to reduce complexity
func (d *DashboardView) handleMillerColumnsNavigation(msg tea.KeyMsg) tea.Cmd {
	// Cancel any pending 'gg' command if another key is pressed
	d.handlePendingGCommand(msg)

	// Map key to action using helper methods
	switch {
	case d.isUpKey(msg):
		return d.handleUpNavigation()
	case d.isDownKey(msg):
		return d.handleDownNavigation()
	case d.isHalfPageDown(msg):
		return d.handleHalfPageDown()
	case d.isHalfPageUp(msg):
		return d.handleHalfPageUp()
	case key.Matches(msg, d.keys.Tab):
		return d.handleTabNavigation()
	case key.Matches(msg, d.keys.Enter):
		return d.handleEnterNavigation()
	case msg.Type == tea.KeyBackspace:
		return d.handleBackspaceNavigation()
	case d.isYankKey(msg):
		return d.handleYankOperation()
	case d.isOpenKey(msg):
		return d.handleOpenOperation()
	case d.isRightKey(msg):
		return d.handleRightNavigation()
	case d.isLeftKey(msg):
		return d.handleLeftNavigation()
	}
	return nil
}

// cycleLayout cycles through available layouts
func (d *DashboardView) cycleLayout() {
	switch d.currentLayout {
	case models.LayoutTripleColumn:
		d.currentLayout = models.LayoutAllRuns
	case models.LayoutAllRuns:
		d.currentLayout = models.LayoutRepositoriesOnly
	case models.LayoutRepositoriesOnly:
		d.currentLayout = models.LayoutTripleColumn
	default:
		d.currentLayout = models.LayoutTripleColumn
	}
}

// scrollToSelected ensures the selected item is visible in the viewport
func (d *DashboardView) scrollToSelected(column int) {
	var selectedIdx int
	var viewport *viewport.Model
	var totalItems int

	switch column {
	case 0:
		selectedIdx = d.selectedRepoIdx
		viewport = &d.repoViewport
		totalItems = len(d.repositories)
	case 1:
		selectedIdx = d.selectedRunIdx
		viewport = &d.runsViewport
		totalItems = len(d.filteredRuns)
	case 2:
		selectedIdx = d.selectedDetailLine
		viewport = &d.detailsViewport
		totalItems = len(d.detailLines)
	default:
		return
	}

	// CRITICAL FIX: Validate bounds before scrolling to prevent panic
	// If selected index is beyond content, reset to safe position
	if totalItems == 0 {
		viewport.YOffset = 0
		return
	}
	
	// Ensure selected index is within bounds
	if selectedIdx >= totalItems {
		selectedIdx = totalItems - 1
	}
	if selectedIdx < 0 {
		selectedIdx = 0
	}

	// Calculate maximum safe YOffset based on content
	maxYOffset := totalItems - 1
	if maxYOffset < 0 {
		maxYOffset = 0
	}

	// Calculate if we need to scroll
	visibleStart := viewport.YOffset
	visibleEnd := viewport.YOffset + viewport.Height - 1

	if selectedIdx < visibleStart {
		// Scroll up to show selected item
		viewport.YOffset = selectedIdx
	} else if selectedIdx > visibleEnd {
		// Scroll down to show selected item
		viewport.YOffset = selectedIdx - viewport.Height + 1
	}
	
	// CRITICAL: Ensure YOffset doesn't exceed content bounds
	if viewport.YOffset > maxYOffset {
		debug.LogToFilef("🚨 SCROLL FIX: Clamping YOffset from %d to %d (max for %d items) 🚨\n", 
			viewport.YOffset, maxYOffset, totalItems)
		viewport.YOffset = maxYOffset
	}
	if viewport.YOffset < 0 {
		viewport.YOffset = 0
	}
}

// Additional navigation helper methods

// restoreOrInitDetailSelection restores saved detail line selection or initializes to first non-empty line
func (d *DashboardView) restoreOrInitDetailSelection() {
	// Try to restore saved selection for this run
	restored := false
	if d.selectedRunData != nil {
		runID := d.selectedRunData.GetIDString()
		if runID != "" {
			if savedLine, exists := d.detailLineMemory[runID]; exists && savedLine >= 0 && savedLine < len(d.detailLines) {
				d.selectedDetailLine = savedLine
				restored = true
			}
		}
	}

	// If not restored, initialize to first non-empty line
	if !restored && len(d.detailLines) > 0 {
		d.selectedDetailLine = 0
		// Skip empty lines at the beginning
		if d.isEmptyLine(d.detailLines[0]) {
			newIdx := d.findNextNonEmptyLine(0, 1)
			if newIdx >= 0 && newIdx < len(d.detailLines) {
				d.selectedDetailLine = newIdx
			}
		}
	}
}

func (d *DashboardView) isUpKey(msg tea.KeyMsg) bool {
	return key.Matches(msg, d.keys.Up) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "k")
}

func (d *DashboardView) isDownKey(msg tea.KeyMsg) bool {
	return key.Matches(msg, d.keys.Down) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "j")
}

func (d *DashboardView) isHalfPageDown(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && string(msg.Runes) == "J"
}

func (d *DashboardView) isHalfPageUp(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && string(msg.Runes) == "K"
}

func (d *DashboardView) isYankKey(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && string(msg.Runes) == "y"
}

func (d *DashboardView) isOpenKey(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && string(msg.Runes) == "o"
}

func (d *DashboardView) isRightKey(msg tea.KeyMsg) bool {
	return key.Matches(msg, d.keys.Right) || 
		(msg.Type == tea.KeyRunes && (string(msg.Runes) == "l" || string(msg.Runes) == "L"))
}

func (d *DashboardView) isLeftKey(msg tea.KeyMsg) bool {
	return key.Matches(msg, d.keys.Left) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "h")
}

// handlePendingGCommand handles the pending 'gg' command state
func (d *DashboardView) handlePendingGCommand(msg tea.KeyMsg) {
	if d.waitingForG {
		// Cancel if it's not the second 'g' or if it's any non-rune key
		if msg.Type != tea.KeyRunes || string(msg.Runes) != "g" {
			d.waitingForG = false
		}
	}
}

// Navigation handlers
func (d *DashboardView) handleUpNavigation() tea.Cmd {
	switch d.focusedColumn {
	case 0:
		return d.navigateRepoUp()
	case 1:
		return d.navigateRunUp()
	case 2:
		return d.navigateDetailUp()
	}
	return nil
}

func (d *DashboardView) handleDownNavigation() tea.Cmd {
	switch d.focusedColumn {
	case 0:
		return d.navigateRepoDown()
	case 1:
		return d.navigateRunDown()
	case 2:
		return d.navigateDetailDown()
	}
	return nil
}

func (d *DashboardView) handleHalfPageDown() tea.Cmd {
	const halfPage = 10
	switch d.focusedColumn {
	case 0:
		return d.navigateRepoHalfPageDown(halfPage)
	case 1:
		return d.navigateRunHalfPageDown(halfPage)
	case 2:
		return d.navigateDetailHalfPageDown(halfPage)
	}
	return nil
}

func (d *DashboardView) handleHalfPageUp() tea.Cmd {
	const halfPage = 10
	switch d.focusedColumn {
	case 0:
		return d.navigateRepoHalfPageUp(halfPage)
	case 1:
		return d.navigateRunHalfPageUp(halfPage)
	case 2:
		return d.navigateDetailHalfPageUp(halfPage)
	}
	return nil
}

// Repository navigation
func (d *DashboardView) navigateRepoUp() tea.Cmd {
	if d.selectedRepoIdx > 0 {
		d.selectedRepoIdx--
	} else if len(d.repositories) > 0 {
		d.selectedRepoIdx = len(d.repositories) - 1 // Wrap to last
	}
	if len(d.repositories) > 0 {
		d.selectedRepo = &d.repositories[d.selectedRepoIdx]
		debug.LogToFilef("\n[NAV UP] Moving to repo[%d]: '%s'\n", d.selectedRepoIdx, d.selectedRepo.Name)
		return d.selectRepository(d.selectedRepo)
	}
	return nil
}

func (d *DashboardView) navigateRepoDown() tea.Cmd {
	if d.selectedRepoIdx < len(d.repositories)-1 {
		d.selectedRepoIdx++
	} else if len(d.repositories) > 0 {
		d.selectedRepoIdx = 0 // Wrap to first
	}
	if len(d.repositories) > 0 {
		d.selectedRepo = &d.repositories[d.selectedRepoIdx]
		debug.LogToFilef("\n[NAV DOWN] Moving to repo[%d]: '%s'\n", d.selectedRepoIdx, d.selectedRepo.Name)
		return d.selectRepository(d.selectedRepo)
	}
	return nil
}

func (d *DashboardView) navigateRepoHalfPageDown(halfPage int) tea.Cmd {
	if len(d.repositories) > 0 {
		newIdx := d.selectedRepoIdx + halfPage
		if newIdx >= len(d.repositories) {
			newIdx = len(d.repositories) - 1
		}
		d.selectedRepoIdx = newIdx
		d.selectedRepo = &d.repositories[d.selectedRepoIdx]
		debug.LogToFilef("\n[NAV HALF-PAGE-DOWN] Moving to repo[%d]: '%s'\n", d.selectedRepoIdx, d.selectedRepo.Name)
		return d.selectRepository(d.selectedRepo)
	}
	return nil
}

func (d *DashboardView) navigateRepoHalfPageUp(halfPage int) tea.Cmd {
	if len(d.repositories) > 0 {
		newIdx := d.selectedRepoIdx - halfPage
		if newIdx < 0 {
			newIdx = 0
		}
		d.selectedRepoIdx = newIdx
		d.selectedRepo = &d.repositories[d.selectedRepoIdx]
		debug.LogToFilef("\n[NAV HALF-PAGE-UP] Moving to repo[%d]: '%s'\n", d.selectedRepoIdx, d.selectedRepo.Name)
		return d.selectRepository(d.selectedRepo)
	}
	return nil
}

// Run navigation
func (d *DashboardView) navigateRunUp() tea.Cmd {
	if d.selectedRunIdx > 0 {
		d.selectedRunIdx--
	} else if len(d.filteredRuns) > 0 {
		d.selectedRunIdx = len(d.filteredRuns) - 1 // Wrap to last
	}
	if len(d.filteredRuns) > d.selectedRunIdx {
		d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
		d.updateDetailLines()
	}
	return nil
}

func (d *DashboardView) navigateRunDown() tea.Cmd {
	if d.selectedRunIdx < len(d.filteredRuns)-1 {
		d.selectedRunIdx++
	} else if len(d.filteredRuns) > 0 {
		d.selectedRunIdx = 0 // Wrap to first
	}
	if len(d.filteredRuns) > d.selectedRunIdx {
		d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
		d.updateDetailLines()
	}
	return nil
}

func (d *DashboardView) navigateRunHalfPageDown(halfPage int) tea.Cmd {
	if len(d.filteredRuns) > 0 {
		newIdx := d.selectedRunIdx + halfPage
		if newIdx >= len(d.filteredRuns) {
			newIdx = len(d.filteredRuns) - 1
		}
		d.selectedRunIdx = newIdx
		d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
		d.updateDetailLines()
	}
	return nil
}

func (d *DashboardView) navigateRunHalfPageUp(halfPage int) tea.Cmd {
	if len(d.filteredRuns) > 0 {
		newIdx := d.selectedRunIdx - halfPage
		if newIdx < 0 {
			newIdx = 0
		}
		d.selectedRunIdx = newIdx
		d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
		d.updateDetailLines()
	}
	return nil
}

// Detail navigation
func (d *DashboardView) navigateDetailUp() tea.Cmd {
	if d.selectedDetailLine > 0 {
		// Try to find previous non-empty line
		newIdx := d.findNextNonEmptyLine(d.selectedDetailLine, -1)
		if newIdx != d.selectedDetailLine {
			d.selectedDetailLine = newIdx
		} else {
			d.selectedDetailLine--
		}
	} else if len(d.detailLines) > 0 {
		d.selectedDetailLine = len(d.detailLines) - 1 // Wrap to last
	}
	return nil
}

func (d *DashboardView) navigateDetailDown() tea.Cmd {
	if d.selectedDetailLine < len(d.detailLines)-1 {
		// Try to find next non-empty line
		newIdx := d.findNextNonEmptyLine(d.selectedDetailLine, 1)
		if newIdx != d.selectedDetailLine {
			d.selectedDetailLine = newIdx
		} else {
			d.selectedDetailLine++
		}
	} else if len(d.detailLines) > 0 {
		d.selectedDetailLine = 0 // Wrap to first
	}
	return nil
}

func (d *DashboardView) navigateDetailHalfPageDown(halfPage int) tea.Cmd {
	if len(d.detailLines) > 0 {
		newIdx := d.selectedDetailLine + halfPage
		if newIdx >= len(d.detailLines) {
			newIdx = len(d.detailLines) - 1
		}
		d.selectedDetailLine = newIdx
	}
	return nil
}

func (d *DashboardView) navigateDetailHalfPageUp(halfPage int) tea.Cmd {
	if len(d.detailLines) > 0 {
		newIdx := d.selectedDetailLine - halfPage
		if newIdx < 0 {
			newIdx = 0
		}
		d.selectedDetailLine = newIdx
	}
	return nil
}

// Tab navigation
func (d *DashboardView) handleTabNavigation() tea.Cmd {
	d.focusedColumn = (d.focusedColumn + 1) % 3
	if d.focusedColumn == 1 && len(d.filteredRuns) > 0 && d.selectedRunData == nil {
		// Moving to runs column, select first run if none selected
		d.selectedRunIdx = 0
		d.selectedRunData = d.filteredRuns[0]
		d.updateDetailLines()
	} else if d.focusedColumn == 2 {
		// Moving to details column, restore or init selection
		d.restoreOrInitDetailSelection()
	}
	return nil
}

// Enter navigation
func (d *DashboardView) handleEnterNavigation() tea.Cmd {
	if d.focusedColumn < 2 {
		d.focusedColumn++
		if d.focusedColumn == 1 && len(d.filteredRuns) > 0 {
			// Moving to runs column, select first run if none selected
			if d.selectedRunData == nil && len(d.filteredRuns) > 0 {
				d.selectedRunIdx = 0
				d.selectedRunData = d.filteredRuns[0]
				d.updateDetailLines()
			}
		} else if d.focusedColumn == 2 {
			// Moving to details column, restore or init selection
			d.restoreOrInitDetailSelection()
		}
	} else if d.focusedColumn == 2 && d.selectedRunData != nil {
		// In details column with a selected run - navigate to details view
		return d.navigateToDetailsView(d.selectedRunData)
	}
	return nil
}

// Backspace navigation
func (d *DashboardView) handleBackspaceNavigation() tea.Cmd {
	if d.focusedColumn > 0 {
		// Save detail line selection if leaving details column
		if d.focusedColumn == 2 && d.selectedRunData != nil {
			runID := d.selectedRunData.GetIDString()
			if runID != "" {
				d.detailLineMemory[runID] = d.selectedDetailLine
			}
		}
		d.focusedColumn--
	}
	return nil
}

// Right navigation
func (d *DashboardView) handleRightNavigation() tea.Cmd {
	if d.focusedColumn < 2 {
		d.focusedColumn++
		// If moving to runs column and no run selected, select first
		if d.focusedColumn == 1 && len(d.filteredRuns) > 0 && d.selectedRunData == nil {
			d.selectedRunIdx = 0
			d.selectedRunData = d.filteredRuns[0]
			d.updateDetailLines()
		} else if d.focusedColumn == 2 {
			// Moving to details column, restore or init selection
			d.restoreOrInitDetailSelection()
		}
	}
	return nil
}

// Left navigation
func (d *DashboardView) handleLeftNavigation() tea.Cmd {
	if d.focusedColumn > 0 {
		// Save detail line selection if leaving details column
		if d.focusedColumn == 2 && d.selectedRunData != nil {
			runID := d.selectedRunData.GetIDString()
			if runID != "" {
				d.detailLineMemory[runID] = d.selectedDetailLine
			}
		}
		d.focusedColumn--
	}
	return nil
}

// Yank operation
func (d *DashboardView) handleYankOperation() tea.Cmd {
	var textToCopy string

	switch d.focusedColumn {
	case 0: // Repository column
		textToCopy = d.getRepoTextToCopy()
	case 1: // Runs column
		textToCopy = d.getRunTextToCopy()
	case 2: // Details column
		textToCopy = d.getDetailTextToCopy()
	}

	if textToCopy != "" {
		return d.performCopyOperation(textToCopy)
	}
	return nil
}

func (d *DashboardView) getRepoTextToCopy() string {
	if d.selectedRepoIdx < len(d.repositories) {
		repo := d.repositories[d.selectedRepoIdx]
		return repo.Name
	}
	return ""
}

func (d *DashboardView) getRunTextToCopy() string {
	if d.selectedRunIdx < len(d.filteredRuns) {
		run := d.filteredRuns[d.selectedRunIdx]
		return fmt.Sprintf("%s - %s", run.GetIDString(), run.Title)
	}
	return ""
}

func (d *DashboardView) getDetailTextToCopy() string {
	if d.selectedDetailLine < len(d.detailLinesOriginal) {
		// Use original untruncated text for copying
		return d.detailLinesOriginal[d.selectedDetailLine]
	}
	return ""
}

func (d *DashboardView) performCopyOperation(textToCopy string) tea.Cmd {
	cmd := d.copyToClipboard(textToCopy)
	// Show what's actually on the clipboard, truncated for display if needed
	displayText := textToCopy
	maxLen := 30
	if len(displayText) > maxLen {
		displayText = displayText[:maxLen-3] + "..."
	}

	if cmd != nil {
		message := fmt.Sprintf("📋 Copied \"%s\"", displayText)
		d.copiedMessage = message // Set for backward compatibility with tests
		d.copiedMessageTime = time.Now()
		d.statusLine.SetTemporaryMessageWithType(message, components.MessageSuccess, 150*time.Millisecond)
		return cmd
	} else {
		d.copiedMessage = "✗ Failed to copy" // Set for backward compatibility with tests
		d.copiedMessageTime = time.Now()
		d.statusLine.SetTemporaryMessageWithType("✗ Failed to copy", components.MessageError, 150*time.Millisecond)
	}
	return nil
}

// Open operation
func (d *DashboardView) handleOpenOperation() tea.Cmd {
	var urlText string

	switch d.focusedColumn {
	case 0: // Repository column
		return d.handleRepoOpen()
	case 1: // Runs column
		urlText = d.getRunURL()
	case 2: // Details column
		return d.handleDetailOpen()
	}

	if urlText != "" {
		return d.openURL(urlText)
	}
	return nil
}

func (d *DashboardView) handleRepoOpen() tea.Cmd {
	if d.selectedRepoIdx < len(d.repositories) {
		repo := d.repositories[d.selectedRepoIdx]
		// Check if we can provide URL options
		apiRepo := d.getAPIRepositoryForRepo(&repo)
		if apiRepo != nil {
			// Show URL selection prompt in status line
			d.showURLSelectionPrompt = true
			d.pendingRepoForURL = &repo
			d.pendingAPIRepoForURL = apiRepo
			return nil
		}
	}
	return nil
}

func (d *DashboardView) getRunURL() string {
	if d.selectedRunIdx < len(d.filteredRuns) {
		run := d.filteredRuns[d.selectedRunIdx]
		if run.PrURL != nil && *run.PrURL != "" {
			return *run.PrURL
		}
	}
	return ""
}

func (d *DashboardView) handleDetailOpen() tea.Cmd {
	if d.selectedDetailLine < len(d.detailLinesOriginal) {
		lineText := d.detailLinesOriginal[d.selectedDetailLine]
		if utils.IsURL(lineText) {
			urlText := utils.ExtractURL(lineText)
			return d.openURL(urlText)
		} else if d.selectedDetailLine == 0 && d.selectedRunData != nil {
			// First line is the ID field, generate RepoBird URL
			runID := d.selectedRunData.GetIDString()
			if utils.IsNonEmptyNumber(runID) {
				urlText := utils.GenerateRepoBirdURL(runID)
				return d.openURL(urlText)
			}
		} else if d.selectedDetailLine == 2 && d.selectedRunData != nil {
			// Repository line - show URL selection prompt
			repoName := d.selectedRunData.GetRepositoryName()
			if repoName != "" {
				repo := d.getRepositoryByName(repoName)
				apiRepo := d.getAPIRepositoryForRepo(repo)
				if apiRepo != nil {
					// Show URL selection prompt in status line
					d.showURLSelectionPrompt = true
					d.pendingRepoForURL = repo
					d.pendingAPIRepoForURL = apiRepo
					return nil
				}
			}
		}
	}
	return nil
}

func (d *DashboardView) openURL(urlText string) tea.Cmd {
	if err := utils.OpenURLWithTimeout(urlText); err == nil {
		d.statusLine.SetTemporaryMessageWithType("🌐 Opened URL in browser", components.MessageSuccess, 1*time.Second)
	} else {
		d.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("✗ Failed to open URL: %v", err), components.MessageError, 1*time.Second)
	}
	return d.startMessageClearTimer(1 * time.Second)
}

// navigateToDetailsView navigates to the details view for a specific run
func (d *DashboardView) navigateToDetailsView(run *models.RunResponse) tea.Cmd {
	if run != nil {
		// Save dashboard state before navigating
		debug.LogToFilef("💾 DASHBOARD: Saving state before navigation - repo=%d, run=%d, detail=%d, column=%d 💾\n",
			d.selectedRepoIdx, d.selectedRunIdx, d.selectedDetailLine, d.focusedColumn)
		d.cache.SetNavigationContext("dashboardState", map[string]interface{}{
			"selectedRepoIdx":    d.selectedRepoIdx,
			"selectedRunIdx":     d.selectedRunIdx,
			"selectedDetailLine": d.selectedDetailLine,
			"focusedColumn":      d.focusedColumn, // This should be 2 (details column)
		})
		
		return func() tea.Msg {
			return messages.NavigateToDetailsMsg{RunData: run}
		}
	}
	return nil
}