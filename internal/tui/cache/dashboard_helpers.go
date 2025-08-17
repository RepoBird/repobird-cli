// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"fmt"
	"sort"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

// BuildRepositoryOverviewFromRuns creates repository overview from runs
func (c *SimpleCache) BuildRepositoryOverviewFromRuns(runs []*models.RunResponse) []models.Repository {
	repoMap := make(map[string]*models.Repository)
	repoIDNameMap := make(map[int]string)
	repoLatestRunCreation := make(map[string]time.Time) // Track latest run creation time per repo

	// First pass: extract unique repository names and create repository objects
	for _, run := range runs {
		if run == nil {
			continue
		}
		repoName := run.GetRepositoryName()
		if repoName == "" {
			continue
		}

		// Track the repo ID to name mapping if we have both
		if run.RepoID > 0 {
			repoIDNameMap[run.RepoID] = repoName
		}

		repo, exists := repoMap[repoName]
		if !exists {
			repo = &models.Repository{
				Name:         repoName,
				RunCounts:    models.RunStats{},
				LastActivity: run.UpdatedAt,
			}
			repoMap[repoName] = repo
		}

		// Track the latest run creation time for this repository
		if latestTime, exists := repoLatestRunCreation[repoName]; !exists || run.CreatedAt.After(latestTime) {
			repoLatestRunCreation[repoName] = run.CreatedAt
		}
	}

	// Second pass: update statistics, including runs that only have repo ID
	for _, run := range runs {
		if run == nil {
			continue
		}
		var repo *models.Repository
		var repoName string

		// Try to find repo by name first
		repoName = run.GetRepositoryName()
		if repoName != "" {
			repo = repoMap[repoName]
		} else if run.RepoID > 0 {
			// Try to find by repo ID mapping
			if mappedName, exists := repoIDNameMap[run.RepoID]; exists {
				repoName = mappedName
				repo = repoMap[mappedName]
			}
		}

		if repo == nil {
			continue
		}

		// Update statistics
		switch run.Status {
		case models.StatusDone:
			repo.RunCounts.Completed++
		case models.StatusProcessing, models.StatusInitializing:
			repo.RunCounts.Running++
		case models.StatusQueued:
			repo.RunCounts.Running++ // Count queued as running for now
		case models.StatusFailed:
			repo.RunCounts.Failed++
		}
		repo.RunCounts.Total++

		// Update last activity
		if run.UpdatedAt.After(repo.LastActivity) {
			repo.LastActivity = run.UpdatedAt
		}

		// Update latest run creation time if newer
		if repoName != "" {
			if latestTime, exists := repoLatestRunCreation[repoName]; !exists || run.CreatedAt.After(latestTime) {
				repoLatestRunCreation[repoName] = run.CreatedAt
			}
		}

		// Note: API repository info would be stored here if available
		// Currently RunResponse doesn't have RepositoryObj field
	}

	// Convert map to slice
	repositories := make([]models.Repository, 0, len(repoMap))
	for _, repo := range repoMap {
		repositories = append(repositories, *repo)
	}

	// Sort repositories by most recent run creation time
	// Repos with runs sorted by latest run creation, repos without runs at the bottom
	sort.Slice(repositories, func(i, j int) bool {
		iTime, iHasRuns := repoLatestRunCreation[repositories[i].Name]
		jTime, jHasRuns := repoLatestRunCreation[repositories[j].Name]

		// If both have runs, sort by most recent run creation
		if iHasRuns && jHasRuns {
			return iTime.After(jTime)
		}

		// Repos with runs come before repos without runs
		if iHasRuns && !jHasRuns {
			return true
		}
		if !iHasRuns && jHasRuns {
			return false
		}

		// If neither has runs, sort alphabetically by name
		return repositories[i].Name < repositories[j].Name
	})

	return repositories
}

// GetRepositoryOverview retrieves cached repository overview
func (c *SimpleCache) GetRepositoryOverview() ([]models.Repository, bool) {
	// No lock needed - GetDashboardCache is thread-safe
	dashData, exists := c.GetDashboardCache()
	if !exists || dashData == nil {
		return nil, false
	}

	// Build repositories from cached runs if not present
	if len(dashData.Runs) > 0 {
		runs := make([]*models.RunResponse, len(dashData.Runs))
		for i := range dashData.Runs {
			runs[i] = &dashData.Runs[i]
		}
		repos := c.BuildRepositoryOverviewFromRuns(runs)
		return repos, true
	}

	return nil, false
}

// SetRepositoryOverview stores repository overview in cache
func (c *SimpleCache) SetRepositoryOverview(repos []models.Repository) {
	// Get existing dashboard data or create new
	dashData, _ := c.GetDashboardCache()
	if dashData == nil {
		dashData = &DashboardData{
			LastUpdated: time.Now(),
		}
	}

	// Note: We're not storing repos directly in DashboardData
	// They are derived from runs, so we just update the timestamp
	dashData.LastUpdated = time.Now()
	c.SetDashboardCache(dashData)
}

// GetFormData retrieves saved form data
func (c *SimpleCache) GetFormData() *FormData {
	// No lock needed - session cache handles thread safety
	if data, found := c.hybrid.session.GetFormData("formData"); found {
		if formData, ok := data.(*FormData); ok {
			return formData
		}
	}
	return nil
}

// SetFormData saves form data
func (c *SimpleCache) SetFormData(data *FormData) {
	// No lock needed - session cache handles thread safety
	_ = c.hybrid.session.SetFormData("formData", data)
}

// ClearFormData clears saved form data
func (c *SimpleCache) ClearFormData() {
	// No lock needed - session cache handles thread safety
	c.hybrid.session.cache.Delete("form:formData")
}

// FormData represents the saved form state
type FormData struct {
	Title          string
	Repository     string
	Source         string
	Target         string
	Issue          string
	Prompt         string
	Context        string
	RunType        string
	Fields         map[string]string // Additional form fields
	ShowContext    bool
	LastLoadedFile string
}

// GetRepositoryHistory returns repository history
func (c *SimpleCache) GetRepositoryHistory() ([]string, error) {
	// No lock needed - GetDashboardCache is thread-safe
	dashData, exists := c.GetDashboardCache()
	if exists && dashData != nil && dashData.RepositoryList != nil {
		return dashData.RepositoryList, nil
	}

	return []string{}, nil
}

// AddRepositoryToHistory adds a repository to the history
func (c *SimpleCache) AddRepositoryToHistory(repo string) {
	// Get existing data without lock
	dashData, _ := c.GetDashboardCache()
	if dashData == nil {
		dashData = &DashboardData{
			LastUpdated: time.Now(),
		}
	}

	// Add to history if not already present
	found := false
	for _, r := range dashData.RepositoryList {
		if r == repo {
			found = true
			break
		}
	}

	if !found {
		dashData.RepositoryList = append([]string{repo}, dashData.RepositoryList...)
		// Keep only last 20 repositories
		if len(dashData.RepositoryList) > 20 {
			dashData.RepositoryList = dashData.RepositoryList[:20]
		}
	}

	c.SetDashboardCache(dashData)
}

// GetCachedList returns cached runs with details
func (c *SimpleCache) GetCachedList() ([]models.RunResponse, bool, map[string]*models.RunResponse) {
	runs := c.GetRuns()
	if len(runs) == 0 {
		return nil, false, nil
	}

	// Build details map from individual cached runs
	details := make(map[string]*models.RunResponse)
	for _, run := range runs {
		runCopy := run
		details[run.ID] = &runCopy
	}

	return runs, true, details
}

// SetCachedList stores runs in cache
func (c *SimpleCache) SetCachedList(runs []models.RunResponse, details map[string]*models.RunResponse) {
	c.SetRuns(runs)

	// Also cache individual run details
	for _, run := range runs {
		c.SetRun(run)
	}
}

// SetRepositoryData caches data for a specific repository
func (c *SimpleCache) SetRepositoryData(repoName string, runs []*models.RunResponse, details map[string]*models.RunResponse) {
	// Prepare data without lock
	key := fmt.Sprintf("repo:%s", repoName)
	data := &RepositoryData{
		Name:        repoName,
		Runs:        runs,
		Details:     details,
		LastUpdated: time.Now(),
	}
	// Call session cache without holding lock - it handles its own thread safety
	_ = c.hybrid.session.SetFormData(key, data)
}

// GetRepositoryData retrieves cached data for a specific repository
func (c *SimpleCache) GetRepositoryData(repoName string) (*RepositoryData, bool) {
	// No lock needed - session cache handles thread safety
	key := fmt.Sprintf("repo:%s", repoName)
	if item, found := c.hybrid.session.GetFormData(key); found {
		if data, ok := item.(*RepositoryData); ok {
			return data, true
		}
	}
	return nil, false
}

// RepositoryData holds cached data for a specific repository
type RepositoryData struct {
	Name        string
	Runs        []*models.RunResponse
	Details     map[string]*models.RunResponse
	LastUpdated time.Time
}

// InitializeCacheForUser is a no-op for the embedded cache (kept for compatibility)
func (c *SimpleCache) InitializeCacheForUser(userID *int) {
	// No-op: Each TUI view has its own cache instance
}

// InitializeDashboardForUser is a no-op for the embedded cache (kept for compatibility)
func (c *SimpleCache) InitializeDashboardForUser(userID *int) {
	// No-op: Each TUI view has its own cache instance
}

// GetFileHashCache returns a map for file hash caching
func (c *SimpleCache) GetFileHashCache() map[string]string {
	// No lock needed - this is a read-only operation
	// Build a map from individual file hashes
	// For simplicity, we'll return an empty map and let the caller manage it
	// In a real implementation, we might want to track this differently
	return make(map[string]string)
}

// SetSelectedIndex stores the selected index (for list view)
func (c *SimpleCache) SetSelectedIndex(idx int) {
	// No lock needed - session cache handles thread safety
	_ = c.hybrid.session.SetFormData("selectedIndex", idx)
}

// GetSelectedIndex retrieves the selected index
func (c *SimpleCache) GetSelectedIndex() int {
	// No lock needed - session cache handles thread safety
	if item, found := c.hybrid.session.GetFormData("selectedIndex"); found {
		if idx, ok := item.(int); ok {
			return idx
		}
	}
	return 0
}
