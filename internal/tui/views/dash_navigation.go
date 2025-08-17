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
func (d *DashboardView) handleMillerColumnsNavigation(msg tea.KeyMsg) tea.Cmd {
	// Cancel any pending 'gg' command if another key is pressed
	if d.waitingForG {
		// Cancel if it's not the second 'g' or if it's any non-rune key
		if msg.Type != tea.KeyRunes || string(msg.Runes) != "g" {
			d.waitingForG = false
		}
		// Continue processing the current key normally
	}

	switch {
	case key.Matches(msg, d.keys.Up) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "k"):
		switch d.focusedColumn {
		case 0: // Repository column
			if d.selectedRepoIdx > 0 {
				d.selectedRepoIdx--
			} else if len(d.repositories) > 0 {
				// Wrap to last item
				d.selectedRepoIdx = len(d.repositories) - 1
			}
			if len(d.repositories) > 0 {
				d.selectedRepo = &d.repositories[d.selectedRepoIdx]
				debug.LogToFilef("\n[NAV UP] Moving to repo[%d]: '%s'\n", d.selectedRepoIdx, d.selectedRepo.Name)
				return d.selectRepository(d.selectedRepo)
			}
		case 1: // Runs column
			if d.selectedRunIdx > 0 {
				d.selectedRunIdx--
			} else if len(d.filteredRuns) > 0 {
				// Wrap to last item
				d.selectedRunIdx = len(d.filteredRuns) - 1
			}
			if len(d.filteredRuns) > d.selectedRunIdx {
				d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
				d.updateDetailLines()
			}
		case 2: // Details column
			if d.selectedDetailLine > 0 {
				// Try to find previous non-empty line
				newIdx := d.findNextNonEmptyLine(d.selectedDetailLine, -1)
				if newIdx != d.selectedDetailLine {
					d.selectedDetailLine = newIdx
				} else {
					// If no non-empty line found, just move up one
					d.selectedDetailLine--
				}
			} else if len(d.detailLines) > 0 {
				// Wrap to last item
				d.selectedDetailLine = len(d.detailLines) - 1
			}
		}

	case key.Matches(msg, d.keys.Down) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "j"):
		switch d.focusedColumn {
		case 0: // Repository column
			if d.selectedRepoIdx < len(d.repositories)-1 {
				d.selectedRepoIdx++
			} else if len(d.repositories) > 0 {
				// Wrap to first item
				d.selectedRepoIdx = 0
			}
			if len(d.repositories) > 0 {
				d.selectedRepo = &d.repositories[d.selectedRepoIdx]
				debug.LogToFilef("\n[NAV DOWN] Moving to repo[%d]: '%s'\n", d.selectedRepoIdx, d.selectedRepo.Name)
				return d.selectRepository(d.selectedRepo)
			}
		case 1: // Runs column
			if d.selectedRunIdx < len(d.filteredRuns)-1 {
				d.selectedRunIdx++
			} else if len(d.filteredRuns) > 0 {
				// Wrap to first item
				d.selectedRunIdx = 0
			}
			if len(d.filteredRuns) > d.selectedRunIdx {
				d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
				d.updateDetailLines()
			}
		case 2: // Details column
			if d.selectedDetailLine < len(d.detailLines)-1 {
				// Try to find next non-empty line
				newIdx := d.findNextNonEmptyLine(d.selectedDetailLine, 1)
				if newIdx != d.selectedDetailLine {
					d.selectedDetailLine = newIdx
				} else {
					// If no non-empty line found, just move down one
					d.selectedDetailLine++
				}
			} else if len(d.detailLines) > 0 {
				// Wrap to first item
				d.selectedDetailLine = 0
			}
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "J":
		// Capital J - scroll half page down
		switch d.focusedColumn {
		case 0: // Repository column
			if len(d.repositories) > 0 {
				// Calculate half page worth of items (assuming ~20 visible items per page)
				halfPage := 10
				newIdx := d.selectedRepoIdx + halfPage
				if newIdx >= len(d.repositories) {
					newIdx = len(d.repositories) - 1
				}
				d.selectedRepoIdx = newIdx
				d.selectedRepo = &d.repositories[d.selectedRepoIdx]
				debug.LogToFilef("\n[NAV HALF-PAGE-DOWN] Moving to repo[%d]: '%s'\n", d.selectedRepoIdx, d.selectedRepo.Name)
				return d.selectRepository(d.selectedRepo)
			}
		case 1: // Runs column
			if len(d.filteredRuns) > 0 {
				// Calculate half page worth of items
				halfPage := 10
				newIdx := d.selectedRunIdx + halfPage
				if newIdx >= len(d.filteredRuns) {
					newIdx = len(d.filteredRuns) - 1
				}
				d.selectedRunIdx = newIdx
				d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
				d.updateDetailLines()
			}
		case 2: // Details column
			if len(d.detailLines) > 0 {
				// Calculate half page worth of items
				halfPage := 10
				newIdx := d.selectedDetailLine + halfPage
				if newIdx >= len(d.detailLines) {
					newIdx = len(d.detailLines) - 1
				}
				d.selectedDetailLine = newIdx
			}
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "K":
		// Capital K - scroll half page up
		switch d.focusedColumn {
		case 0: // Repository column
			if len(d.repositories) > 0 {
				// Calculate half page worth of items (assuming ~20 visible items per page)
				halfPage := 10
				newIdx := d.selectedRepoIdx - halfPage
				if newIdx < 0 {
					newIdx = 0
				}
				d.selectedRepoIdx = newIdx
				d.selectedRepo = &d.repositories[d.selectedRepoIdx]
				debug.LogToFilef("\n[NAV HALF-PAGE-UP] Moving to repo[%d]: '%s'\n", d.selectedRepoIdx, d.selectedRepo.Name)
				return d.selectRepository(d.selectedRepo)
			}
		case 1: // Runs column
			if len(d.filteredRuns) > 0 {
				// Calculate half page worth of items
				halfPage := 10
				newIdx := d.selectedRunIdx - halfPage
				if newIdx < 0 {
					newIdx = 0
				}
				d.selectedRunIdx = newIdx
				d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
				d.updateDetailLines()
			}
		case 2: // Details column
			if len(d.detailLines) > 0 {
				// Calculate half page worth of items
				halfPage := 10
				newIdx := d.selectedDetailLine - halfPage
				if newIdx < 0 {
					newIdx = 0
				}
				d.selectedDetailLine = newIdx
			}
		}

	case key.Matches(msg, d.keys.Tab):
		// Tab cycles through columns
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

	case key.Matches(msg, d.keys.Enter):
		// Enter moves focus right and selects first item
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
		}

	case msg.Type == tea.KeyBackspace:
		// Backspace moves focus left
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

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "y":
		// Copy current row/line in any column
		var textToCopy string

		switch d.focusedColumn {
		case 0: // Repository column
			if d.selectedRepoIdx < len(d.repositories) {
				repo := d.repositories[d.selectedRepoIdx]
				textToCopy = repo.Name
			}
		case 1: // Runs column
			if d.selectedRunIdx < len(d.filteredRuns) {
				run := d.filteredRuns[d.selectedRunIdx]
				textToCopy = fmt.Sprintf("%s - %s", run.GetIDString(), run.Title)
			}
		case 2: // Details column
			if d.selectedDetailLine < len(d.detailLinesOriginal) {
				// Use original untruncated text for copying
				textToCopy = d.detailLinesOriginal[d.selectedDetailLine]
			}
		}

		if textToCopy != "" {
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
		}

	case msg.Type == tea.KeyRunes && string(msg.Runes) == "o":
		// Open URL in browser if current selection contains a URL
		var urlText string

		switch d.focusedColumn {
		case 0: // Repository column - handle repository URLs
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
		case 1: // Runs column - could check for PR URLs in run data
			if d.selectedRunIdx < len(d.filteredRuns) {
				run := d.filteredRuns[d.selectedRunIdx]
				if run.PrURL != nil && *run.PrURL != "" {
					urlText = *run.PrURL
				}
			}
		case 2: // Details column - check if selected line contains a URL or is an ID field
			if d.selectedDetailLine < len(d.detailLinesOriginal) {
				lineText := d.detailLinesOriginal[d.selectedDetailLine]
				if utils.IsURL(lineText) {
					urlText = utils.ExtractURL(lineText)
				} else if d.selectedDetailLine == 0 && d.selectedRunData != nil {
					// First line is the ID field, generate RepoBird URL
					runID := d.selectedRunData.GetIDString()
					if utils.IsNonEmptyNumber(runID) {
						urlText = utils.GenerateRepoBirdURL(runID)
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
		}

		if urlText != "" {
			if err := utils.OpenURL(urlText); err == nil {
				d.statusLine.SetTemporaryMessageWithType("🌐 Opened URL in browser", components.MessageSuccess, 1*time.Second)
			} else {
				d.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("✗ Failed to open URL: %v", err), components.MessageError, 1*time.Second)
			}
			return d.startMessageClearTimer(1 * time.Second)
		}

	case key.Matches(msg, d.keys.Right) || (msg.Type == tea.KeyRunes && (string(msg.Runes) == "l" || string(msg.Runes) == "L")):
		// Move focus to the right (l/L keys or right arrow)
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

	case key.Matches(msg, d.keys.Left) || (msg.Type == tea.KeyRunes && string(msg.Runes) == "h"):
		// Move focus to the left
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

	switch column {
	case 0:
		selectedIdx = d.selectedRepoIdx
		viewport = &d.repoViewport
	case 1:
		selectedIdx = d.selectedRunIdx
		viewport = &d.runsViewport
	case 2:
		selectedIdx = d.selectedDetailLine
		viewport = &d.detailsViewport
	default:
		return
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
}

// Additional navigation helper methods

// moveToFirstItem moves selection to first item in current column
func (d *DashboardView) moveToFirstItem() {
	switch d.focusedColumn {
	case 0: // Repository column
		if len(d.repositories) > 0 {
			d.selectedRepoIdx = 0
			d.selectedRepo = &d.repositories[0]
		}
	case 1: // Runs column
		if len(d.filteredRuns) > 0 {
			d.selectedRunIdx = 0
			d.selectedRunData = d.filteredRuns[0]
			d.updateDetailLines()
		}
	case 2: // Details column
		d.restoreOrInitDetailSelection()
	}
}

// moveToLastItem moves selection to last item in current column
func (d *DashboardView) moveToLastItem() {
	switch d.focusedColumn {
	case 0: // Repository column
		if len(d.repositories) > 0 {
			d.selectedRepoIdx = len(d.repositories) - 1
			d.selectedRepo = &d.repositories[d.selectedRepoIdx]
		}
	case 1: // Runs column
		if len(d.filteredRuns) > 0 {
			d.selectedRunIdx = len(d.filteredRuns) - 1
			d.selectedRunData = d.filteredRuns[d.selectedRunIdx]
			d.updateDetailLines()
		}
	case 2: // Details column
		if len(d.detailLines) > 0 {
			d.selectedDetailLine = len(d.detailLines) - 1
		}
	}
}

// handleColumnNavigation handles left/right column navigation
func (d *DashboardView) handleColumnNavigation(direction int) {
	if direction > 0 && d.focusedColumn < 2 {
		// Move right
		d.focusedColumn++
		d.ensureValidSelection()
	} else if direction < 0 && d.focusedColumn > 0 {
		// Save detail line selection if leaving details column
		if d.focusedColumn == 2 && d.selectedRunData != nil {
			runID := d.selectedRunData.GetIDString()
			if runID != "" {
				d.detailLineMemory[runID] = d.selectedDetailLine
			}
		}
		// Move left
		d.focusedColumn--
	}
}

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

// ensureValidSelection ensures there's a valid selection when entering a column
func (d *DashboardView) ensureValidSelection() {
	switch d.focusedColumn {
	case 1: // Runs column
		if len(d.filteredRuns) > 0 && d.selectedRunData == nil {
			d.selectedRunIdx = 0
			d.selectedRunData = d.filteredRuns[0]
			d.updateDetailLines()
		}
	case 2: // Details column
		d.restoreOrInitDetailSelection()
	}
}
