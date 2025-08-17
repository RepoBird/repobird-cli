// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"strings"

	"github.com/repobird/repobird-cli/internal/tui/components"
)

// activateFZFMode activates FZF mode for the current column
func (d *DashboardView) activateFZFMode() {
	var items []string

	switch d.focusedColumn {
	case 0: // Repository column
		items = make([]string, len(d.repositories))
		for i, repo := range d.repositories {
			statusIcon := d.getRepositoryStatusIcon(&repo)
			items[i] = fmt.Sprintf("%s %s", statusIcon, repo.Name)
		}
		d.fzfColumn = 0

	case 1: // Runs column
		if len(d.filteredRuns) > 0 {
			items = make([]string, len(d.filteredRuns))
			for i, run := range d.filteredRuns {
				statusIcon := d.getRunStatusIcon(run.Status)
				title := run.Title
				if title == "" {
					title = "Untitled"
				}
				items[i] = fmt.Sprintf("%s %s", statusIcon, title)
			}
			d.fzfColumn = 1
		}

	case 2: // Details column
		if len(d.detailLines) > 0 {
			items = d.detailLines
			d.fzfColumn = 2
		}
	}

	if len(items) > 0 {
		// Calculate appropriate width for FZF
		columnWidth := d.width / 3
		if d.focusedColumn == 2 {
			columnWidth = d.width - (2 * (d.width / 3))
		}

		d.fzfMode = components.NewFZFMode(items, columnWidth, 15)
		d.fzfMode.Activate()
	}
}

// renderWithFZFOverlay renders the dashboard with FZF dropdown overlay
func (d *DashboardView) renderWithFZFOverlay(baseView string) string {
	if d.fzfMode == nil || !d.fzfMode.IsActive() {
		return baseView
	}

	// Split base view into lines
	baseLines := strings.Split(baseView, "\n")

	// Calculate position for FZF dropdown based on focused column and selected item
	columnWidth := d.width / 3
	var xOffset int
	var yOffset int

	switch d.fzfColumn {
	case 0: // Repository column
		xOffset = 2
		yOffset = 3 + d.selectedRepoIdx // Position at selected repository
	case 1: // Runs column
		xOffset = columnWidth + 2
		yOffset = 3 + d.selectedRunIdx // Position at selected run
	case 2: // Details column
		xOffset = (2 * columnWidth) + 2
		yOffset = 3 + d.selectedDetailLine // Position at selected detail line
	}

	// Ensure yOffset is within bounds
	if yOffset < 3 {
		yOffset = 3
	}
	if yOffset > len(baseLines)-15 {
		yOffset = len(baseLines) - 15
	}

	// Create FZF dropdown view
	fzfView := d.fzfMode.View()
	fzfLines := strings.Split(fzfView, "\n")

	// Create a new view with the FZF dropdown overlaid
	result := make([]string, len(baseLines))
	copy(result, baseLines)

	// Insert FZF dropdown at the calculated position
	for i, fzfLine := range fzfLines {
		lineIdx := yOffset + i
		if lineIdx >= 0 && lineIdx < len(result) {
			// Create the overlay line by combining base content and FZF dropdown
			if xOffset < len(result[lineIdx]) {
				// Preserve part of the base line before the dropdown
				basePart := ""
				if xOffset > 0 {
					minLen := xOffset
					if len(result[lineIdx]) < minLen {
						minLen = len(result[lineIdx])
					}
					basePart = result[lineIdx][:minLen]
				}
				// Add the FZF line
				result[lineIdx] = basePart + fzfLine
			} else {
				// Line is shorter than offset, pad and add FZF
				padding := strings.Repeat(" ", xOffset-len(result[lineIdx]))
				result[lineIdx] = result[lineIdx] + padding + fzfLine
			}
		}
	}

	return strings.Join(result, "\n")
}
