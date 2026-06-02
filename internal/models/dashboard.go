// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import (
	"strings"
	"time"
)

// LayoutType represents the different dashboard layout options
type LayoutType int

const (
	// LayoutTripleColumn shows repositories, runs, and details in three columns (Miller Columns style)
	LayoutTripleColumn LayoutType = iota
	// LayoutAllRuns shows all runs in a chronological timeline view
	LayoutAllRuns
	// LayoutRepositoriesOnly shows only the repositories list
	LayoutRepositoriesOnly
)

// String returns a string representation of the layout type
func (l LayoutType) String() string {
	switch l {
	case LayoutTripleColumn:
		return "triple_column"
	case LayoutAllRuns:
		return "all_runs"
	case LayoutRepositoriesOnly:
		return "repositories_only"
	default:
		return "unknown"
	}
}

// Repository represents a repository with metadata for dashboard display
type Repository struct {
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Stars        int       `json:"stars,omitempty"`
	LastActivity time.Time `json:"last_activity,omitempty"`
	RunCounts    RunStats  `json:"run_counts"`
	Languages    []string  `json:"languages,omitempty"`
	Size         string    `json:"size,omitempty"`
	Contributors int       `json:"contributors,omitempty"`
}

// RunStats contains aggregated statistics about runs for a repository
type RunStats struct {
	Running   int `json:"running"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Total     int `json:"total"`
}

// GetDisplayName returns the repository name for display purposes
func (r *Repository) GetDisplayName() string {
	if r.Name != "" {
		return r.Name
	}
	return "Unknown Repository"
}

// APIRepository represents a repository from the API
type APIRepository struct {
	ID                    int     `json:"id"`
	Name                  string  `json:"name"`
	RepoName              string  `json:"repoName"`
	RepoOwner             string  `json:"repoOwner"`
	RepoURL               string  `json:"repoUrl"`
	DefaultBranch         string  `json:"defaultBranch"`
	DefaultBaseBranch     *string `json:"defaultBaseBranch"`
	DefaultPRTargetBranch *string `json:"defaultPrTargetBranch"`
	DefaultOutputBranch   *string `json:"defaultOutputBranch"`
	IsEnabled             bool    `json:"isEnabled"`
	GitHubInstallationID  *int    `json:"githubInstallationId"`
}

// FullName returns the owner/repo name when the API provides owner and repo parts.
func (r APIRepository) FullName() string {
	if r.RepoOwner != "" && r.RepoName != "" {
		return r.RepoOwner + "/" + r.RepoName
	}
	if strings.Contains(r.Name, "/") {
		return r.Name
	}
	if r.RepoName != "" {
		return r.RepoName
	}
	return r.Name
}

// RepositoryDefaultsUpdate describes branch-default changes for a repository.
type RepositoryDefaultsUpdate struct {
	DefaultBaseBranch        *string
	DefaultPRTargetBranch    *string
	DefaultOutputBranch      *string
	ClearDefaultBaseBranch   bool
	ClearDefaultPRTarget     bool
	ClearDefaultOutputBranch bool
}

// Payload converts default updates into the sparse JSON payload expected by the API.
func (u RepositoryDefaultsUpdate) Payload() map[string]interface{} {
	payload := make(map[string]interface{})
	if u.ClearDefaultBaseBranch {
		payload["defaultBaseBranch"] = nil
	} else if u.DefaultBaseBranch != nil {
		payload["defaultBaseBranch"] = *u.DefaultBaseBranch
	}
	if u.ClearDefaultPRTarget {
		payload["defaultPrTargetBranch"] = nil
	} else if u.DefaultPRTargetBranch != nil {
		payload["defaultPrTargetBranch"] = *u.DefaultPRTargetBranch
	}
	if u.ClearDefaultOutputBranch {
		payload["defaultOutputBranch"] = nil
	} else if u.DefaultOutputBranch != nil {
		payload["defaultOutputBranch"] = *u.DefaultOutputBranch
	}
	return payload
}

// RepositoryListResponse represents the API response for repository list
type RepositoryListResponse struct {
	Data     []APIRepository `json:"data"`
	Metadata struct {
		CurrentPage int `json:"currentPage"`
		Total       int `json:"total"`
		TotalPages  int `json:"totalPages"`
	} `json:"metadata"`
}
