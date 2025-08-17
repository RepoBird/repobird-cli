// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package views

import (
	"fmt"
	"strings"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/utils"
)

// isEmptyLine checks if a line in the details column is empty or just whitespace
func (d *DashboardView) isEmptyLine(line string) bool {
	return strings.TrimSpace(line) == ""
}

// getAPIRepositoryForRepo finds the corresponding APIRepository for a Repository
func (d *DashboardView) getAPIRepositoryForRepo(repo *models.Repository) *models.APIRepository {
	if repo == nil || d.apiRepositories == nil {
		return nil
	}

	// Find matching API repository by name
	for _, apiRepo := range d.apiRepositories {
		apiRepoName := apiRepo.Name
		if apiRepoName == "" {
			apiRepoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
		}
		if apiRepoName == repo.Name {
			return &apiRepo
		}
	}

	return nil
}

// getRepositoryByName finds a Repository object by name
func (d *DashboardView) getRepositoryByName(name string) *models.Repository {
	if name == "" || len(d.repositories) == 0 {
		return nil
	}

	for i := range d.repositories {
		if d.repositories[i].Name == name {
			return &d.repositories[i]
		}
	}

	return nil
}

// hasCurrentSelectionURL checks if the current selection contains a URL or can generate a RepoBird URL
func (d *DashboardView) hasCurrentSelectionURL() bool {
	switch d.focusedColumn {
	case 0: // Repository column - check if we have API repository data with URLs
		if d.selectedRepoIdx < len(d.repositories) {
			repo := d.repositories[d.selectedRepoIdx]
			apiRepo := d.getAPIRepositoryForRepo(&repo)
			return apiRepo != nil && apiRepo.RepoURL != ""
		}
		return false
	case 1: // Runs column - check for PR URL
		if d.selectedRunIdx < len(d.filteredRuns) {
			run := d.filteredRuns[d.selectedRunIdx]
			return run.PrURL != nil && *run.PrURL != ""
		}
	case 2: // Details column - check if selected line contains URL or can generate RepoBird URL
		if d.selectedDetailLine < len(d.detailLinesOriginal) {
			lineText := d.detailLinesOriginal[d.selectedDetailLine]
			if utils.IsURL(lineText) {
				return true
			}
			// Check if this is the ID field (first line) and we can generate a RepoBird URL
			if d.selectedDetailLine == 0 && d.selectedRunData != nil {
				runID := d.selectedRunData.GetIDString()
				return utils.IsNonEmptyNumber(runID)
			}
			// Check if this is the repository line (line 2) and we have repository data
			if d.selectedDetailLine == 2 && d.selectedRunData != nil {
				repoName := d.selectedRunData.GetRepositoryName()
				if repoName != "" {
					// Find the corresponding Repository object and check if it has URLs
					repo := d.getRepositoryByName(repoName)
					if repo != nil {
						apiRepo := d.getAPIRepositoryForRepo(repo)
						return apiRepo != nil && apiRepo.RepoURL != ""
					}
				}
			}
		}
	}
	return false
}

// State validation and index management helper methods

// isValidRepoIndex checks if the given repository index is valid
func (d *DashboardView) isValidRepoIndex(index int) bool {
	return index >= 0 && index < len(d.repositories)
}

// isValidRunIndex checks if the given run index is valid
func (d *DashboardView) isValidRunIndex(index int) bool {
	return index >= 0 && index < len(d.filteredRuns)
}

// isValidDetailLineIndex checks if the given detail line index is valid
func (d *DashboardView) isValidDetailLineIndex(index int) bool {
	return index >= 0 && index < len(d.detailLines)
}

// resetSelectionState resets all selection state to initial values
func (d *DashboardView) resetSelectionState() {
	d.selectedRepoIdx = 0
	d.selectedRunIdx = 0
	d.selectedDetailLine = 0
	d.focusedColumn = 0
	d.selectedRepo = nil
	d.selectedRunData = nil
	d.detailLines = []string{}
	d.detailLinesOriginal = []string{}
}

// setSelectedRepository safely sets the selected repository and updates related state
func (d *DashboardView) setSelectedRepository(index int) bool {
	if !d.isValidRepoIndex(index) {
		return false
	}

	d.selectedRepoIdx = index
	d.selectedRepo = &d.repositories[index]

	// Reset dependent selections
	d.selectedRunIdx = 0
	d.selectedDetailLine = 0
	d.selectedRunData = nil
	d.filteredRuns = nil
	d.detailLines = []string{}
	d.detailLinesOriginal = []string{}

	return true
}

// setSelectedRun safely sets the selected run and updates related state
func (d *DashboardView) setSelectedRun(index int) bool {
	if !d.isValidRunIndex(index) {
		return false
	}

	d.selectedRunIdx = index
	d.selectedRunData = d.filteredRuns[index]

	// Reset dependent selections
	d.selectedDetailLine = 0
	d.detailLines = []string{}
	d.detailLinesOriginal = []string{}

	return true
}

// setSelectedDetailLine safely sets the selected detail line
func (d *DashboardView) setSelectedDetailLine(index int) bool {
	if !d.isValidDetailLineIndex(index) {
		return false
	}

	d.selectedDetailLine = index
	return true
}

// getSelectionState returns the current selection state for saving/restoration
func (d *DashboardView) getSelectionState() (repoIdx, runIdx, detailLineIdx, focusedColumn int) {
	return d.selectedRepoIdx, d.selectedRunIdx, d.selectedDetailLine, d.focusedColumn
}

// restoreSelectionState restores the selection state from saved values
func (d *DashboardView) restoreSelectionState(repoIdx, runIdx, detailLineIdx, focusedColumn int) {
	// Restore repository selection
	if d.isValidRepoIndex(repoIdx) {
		d.selectedRepoIdx = repoIdx
		d.selectedRepo = &d.repositories[repoIdx]
	}

	// Restore run selection
	if d.isValidRunIndex(runIdx) {
		d.selectedRunIdx = runIdx
		d.selectedRunData = d.filteredRuns[runIdx]
	}

	// Restore detail line selection
	if d.isValidDetailLineIndex(detailLineIdx) {
		d.selectedDetailLine = detailLineIdx
	}

	// Restore focused column
	if focusedColumn >= 0 && focusedColumn <= 2 {
		d.focusedColumn = focusedColumn
	}
}

// hasValidData checks if the dashboard has valid data loaded
func (d *DashboardView) hasValidData() bool {
	return len(d.repositories) > 0 || len(d.allRuns) > 0
}

// getCurrentRepository returns the currently selected repository, or nil if none
func (d *DashboardView) getCurrentRepository() *models.Repository {
	if d.isValidRepoIndex(d.selectedRepoIdx) {
		return &d.repositories[d.selectedRepoIdx]
	}
	return nil
}

// getCurrentRun returns the currently selected run, or nil if none
func (d *DashboardView) getCurrentRun() *models.RunResponse {
	if d.isValidRunIndex(d.selectedRunIdx) {
		return d.filteredRuns[d.selectedRunIdx]
	}
	return nil
}

// getCurrentDetailLine returns the currently selected detail line text, or empty string if none
func (d *DashboardView) getCurrentDetailLine() string {
	if d.isValidDetailLineIndex(d.selectedDetailLine) && d.selectedDetailLine < len(d.detailLinesOriginal) {
		return d.detailLinesOriginal[d.selectedDetailLine]
	}
	return ""
}

// canNavigateLeft checks if navigation to the left is possible
func (d *DashboardView) canNavigateLeft() bool {
	return d.focusedColumn > 0
}

// canNavigateRight checks if navigation to the right is possible
func (d *DashboardView) canNavigateRight() bool {
	return d.focusedColumn < 2
}

// canNavigateUp checks if navigation up is possible in the current column
func (d *DashboardView) canNavigateUp() bool {
	switch d.focusedColumn {
	case 0:
		return d.selectedRepoIdx > 0
	case 1:
		return d.selectedRunIdx > 0
	case 2:
		return d.selectedDetailLine > 0
	}
	return false
}

// canNavigateDown checks if navigation down is possible in the current column
func (d *DashboardView) canNavigateDown() bool {
	switch d.focusedColumn {
	case 0:
		return d.selectedRepoIdx < len(d.repositories)-1
	case 1:
		return d.selectedRunIdx < len(d.filteredRuns)-1
	case 2:
		return d.selectedDetailLine < len(d.detailLines)-1
	}
	return false
}
