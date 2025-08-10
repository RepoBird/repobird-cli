package models

import "time"

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
	ID                   int    `json:"id"`
	Name                 string `json:"name"`
	RepoName             string `json:"repoName"`
	RepoOwner            string `json:"repoOwner"`
	RepoURL              string `json:"repoUrl"`
	DefaultBranch        string `json:"defaultBranch"`
	IsEnabled            bool   `json:"isEnabled"`
	GitHubInstallationID *int   `json:"githubInstallationId"`
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
