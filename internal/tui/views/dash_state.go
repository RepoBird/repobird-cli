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
