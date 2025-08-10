package models

import (
	"sort"
	"strings"
	"time"
)

// RepositoryFilter represents filters that can be applied to repository lists
type RepositoryFilter struct {
	NamePattern   string
	Languages     []string
	MinStars      int
	MaxStars      int
	HasActivity   bool
	ActivitySince *time.Time
	StatusFilter  string // "all", "active", "inactive", "has_failures"
}

// RepositorySortBy represents different ways to sort repositories
type RepositorySortBy int

const (
	SortByName RepositorySortBy = iota
	SortByActivity
	SortByStars
	SortByRunCount
	SortByStatus
)

// RepositoryAggregator provides methods for working with collections of repositories
type RepositoryAggregator struct {
	repositories []Repository
	runs         []*RunResponse
}

// NewRepositoryAggregator creates a new repository aggregator
func NewRepositoryAggregator(runs []*RunResponse) *RepositoryAggregator {
	return &RepositoryAggregator{
		runs: runs,
	}
}

// ExtractRepositories extracts unique repositories from a list of runs
func (ra *RepositoryAggregator) ExtractRepositories() []Repository {
	repoMap := make(map[string]*Repository)

	// Extract unique repository names and create repository objects
	for _, run := range ra.runs {
		if run.Repository == "" {
			continue
		}

		repo, exists := repoMap[run.Repository]
		if !exists {
			repo = &Repository{
				Name:         run.Repository,
				RunCounts:    RunStats{},
				LastActivity: run.UpdatedAt,
			}
			repoMap[run.Repository] = repo
		}

		// Update last activity if this run is more recent
		if run.UpdatedAt.After(repo.LastActivity) {
			repo.LastActivity = run.UpdatedAt
		}

		// Update run counts
		repo.RunCounts.Total++
		switch run.Status {
		case StatusQueued, StatusInitializing, StatusProcessing, StatusPostProcess:
			repo.RunCounts.Running++
		case StatusDone:
			repo.RunCounts.Completed++
		case StatusFailed:
			repo.RunCounts.Failed++
		}
	}

	// Convert map to slice
	repositories := make([]Repository, 0, len(repoMap))
	for _, repo := range repoMap {
		repositories = append(repositories, *repo)
	}

	return repositories
}

// FilterRepositories applies filters to a list of repositories
func FilterRepositories(repos []Repository, filter *RepositoryFilter) []Repository {
	if filter == nil {
		return repos
	}

	filtered := make([]Repository, 0)

	for _, repo := range repos {
		if !matchesFilter(&repo, filter) {
			continue
		}
		filtered = append(filtered, repo)
	}

	return filtered
}

// matchesFilter checks if a repository matches the given filter
func matchesFilter(repo *Repository, filter *RepositoryFilter) bool {
	// Name pattern filter
	if filter.NamePattern != "" {
		if !strings.Contains(strings.ToLower(repo.Name), strings.ToLower(filter.NamePattern)) {
			return false
		}
	}

	// Stars filter
	if filter.MinStars > 0 && repo.Stars < filter.MinStars {
		return false
	}
	if filter.MaxStars > 0 && repo.Stars > filter.MaxStars {
		return false
	}

	// Activity filter
	if filter.HasActivity && repo.LastActivity.IsZero() {
		return false
	}
	if filter.ActivitySince != nil && repo.LastActivity.Before(*filter.ActivitySince) {
		return false
	}

	// Language filter
	if len(filter.Languages) > 0 {
		hasLanguage := false
		for _, filterLang := range filter.Languages {
			for _, repoLang := range repo.Languages {
				if strings.EqualFold(filterLang, repoLang) {
					hasLanguage = true
					break
				}
			}
			if hasLanguage {
				break
			}
		}
		if !hasLanguage {
			return false
		}
	}

	// Status filter
	switch filter.StatusFilter {
	case "active":
		if repo.RunCounts.Running == 0 {
			return false
		}
	case "inactive":
		if repo.RunCounts.Total > 0 {
			return false
		}
	case "has_failures":
		if repo.RunCounts.Failed == 0 {
			return false
		}
	}

	return true
}

// SortRepositories sorts a list of repositories by the specified criteria
func SortRepositories(repos []Repository, sortBy RepositorySortBy, ascending bool) []Repository {
	sorted := make([]Repository, len(repos))
	copy(sorted, repos)

	sort.Slice(sorted, func(i, j int) bool {
		var less bool
		switch sortBy {
		case SortByName:
			less = strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
		case SortByActivity:
			less = sorted[i].LastActivity.After(sorted[j].LastActivity) // More recent first
		case SortByStars:
			less = sorted[i].Stars > sorted[j].Stars // More stars first
		case SortByRunCount:
			less = sorted[i].RunCounts.Total > sorted[j].RunCounts.Total // More runs first
		case SortByStatus:
			less = compareRepositoryStatus(&sorted[i], &sorted[j])
		default:
			less = strings.ToLower(sorted[i].Name) < strings.ToLower(sorted[j].Name)
		}

		if ascending {
			return less
		}
		return !less
	})

	return sorted
}

// compareRepositoryStatus compares repositories by their status priority
// Order: Running > Failed > Completed > No runs
func compareRepositoryStatus(a, b *Repository) bool {
	scoreA := getRepositoryStatusScore(a)
	scoreB := getRepositoryStatusScore(b)
	return scoreA > scoreB
}

// getRepositoryStatusScore assigns a priority score based on run status
func getRepositoryStatusScore(repo *Repository) int {
	if repo.RunCounts.Running > 0 {
		return 4 // Highest priority for running
	}
	if repo.RunCounts.Failed > 0 {
		return 3 // High priority for failed
	}
	if repo.RunCounts.Completed > 0 {
		return 2 // Medium priority for completed
	}
	return 1 // Lowest priority for no runs
}

// GetRepositoryDisplayName returns a formatted display name for the repository
func GetRepositoryDisplayName(repo *Repository, maxLength int) string {
	name := repo.GetDisplayName()
	if maxLength > 0 && len(name) > maxLength {
		if maxLength > 3 {
			return name[:maxLength-3] + "..."
		}
		return name[:maxLength]
	}
	return name
}

// GetRepositoryStatusSummary returns a human-readable status summary
func GetRepositoryStatusSummary(repo *Repository) string {
	if repo.RunCounts.Total == 0 {
		return "No runs"
	}

	if repo.RunCounts.Running > 0 {
		return "Running"
	}

	if repo.RunCounts.Failed > 0 {
		return "Has failures"
	}

	if repo.RunCounts.Completed > 0 {
		return "Completed"
	}

	return "Unknown"
}

// GetRepositoryRunsSummary returns a summary of runs for display
func GetRepositoryRunsSummary(repo *Repository) string {
	if repo.RunCounts.Total == 0 {
		return "0 runs"
	}

	parts := []string{}
	if repo.RunCounts.Running > 0 {
		parts = append(parts, "running")
	}
	if repo.RunCounts.Completed > 0 {
		parts = append(parts, "completed")
	}
	if repo.RunCounts.Failed > 0 {
		parts = append(parts, "failed")
	}

	if len(parts) == 0 {
		return "0 runs"
	}

	return strings.Join(parts, ", ")
}
