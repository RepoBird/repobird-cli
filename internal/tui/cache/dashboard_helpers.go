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

	// First pass: extract unique repository names and create repository objects
	for _, run := range runs {
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
	}

	// Second pass: update statistics, including runs that only have repo ID
	for _, run := range runs {
		var repo *models.Repository

		// Try to find repo by name first
		repoName := run.GetRepositoryName()
		if repoName != "" {
			repo = repoMap[repoName]
		} else if run.RepoID > 0 {
			// Try to find by repo ID mapping
			if mappedName, exists := repoIDNameMap[run.RepoID]; exists {
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

		// Note: API repository info would be stored here if available
		// Currently RunResponse doesn't have RepositoryObj field
	}

	// Convert map to slice and sort by last activity
	repositories := make([]models.Repository, 0, len(repoMap))
	for _, repo := range repoMap {
		repositories = append(repositories, *repo)
	}

	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].LastActivity.After(repositories[j].LastActivity)
	})

	return repositories
}

// GetRepositoryOverview retrieves cached repository overview
func (c *SimpleCache) GetRepositoryOverview() ([]models.Repository, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.RLock()
	defer c.mu.RUnlock()

	if data, found := c.hybrid.session.GetFormData("formData"); found {
		if formData, ok := data.(*FormData); ok {
			return formData
		}
	}
	return nil
}

// SetFormData saves form data
func (c *SimpleCache) SetFormData(data *FormData) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.hybrid.session.SetFormData("formData", data)
}

// FormData represents the saved form state
type FormData struct {
	Title      string
	Repository string
	Source     string
	Target     string
	Issue      string
	Prompt     string
	Context    string
	RunType    string
}

// GetRepositoryHistory returns repository history
func (c *SimpleCache) GetRepositoryHistory() ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dashData, exists := c.GetDashboardCache()
	if exists && dashData != nil && dashData.RepositoryList != nil {
		return dashData.RepositoryList, nil
	}

	return []string{}, nil
}

// AddRepositoryToHistory adds a repository to the history
func (c *SimpleCache) AddRepositoryToHistory(repo string) {
	c.mu.Lock()
	defer c.mu.Unlock()

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
	if runs == nil {
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
	c.mu.Lock()
	defer c.mu.Unlock()

	key := fmt.Sprintf("repo:%s", repoName)
	data := &RepositoryData{
		Name:        repoName,
		Runs:        runs,
		Details:     details,
		LastUpdated: time.Now(),
	}
	_ = c.hybrid.session.SetFormData(key, data)
}

// GetRepositoryData retrieves cached data for a specific repository
func (c *SimpleCache) GetRepositoryData(repoName string) (*RepositoryData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

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
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Build a map from individual file hashes
	// For simplicity, we'll return an empty map and let the caller manage it
	// In a real implementation, we might want to track this differently
	return make(map[string]string)
}

// SetSelectedIndex stores the selected index (for list view)
func (c *SimpleCache) SetSelectedIndex(idx int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	_ = c.hybrid.session.SetFormData("selectedIndex", idx)
}

// GetSelectedIndex retrieves the selected index
func (c *SimpleCache) GetSelectedIndex() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if item, found := c.hybrid.session.GetFormData("selectedIndex"); found {
		if idx, ok := item.(int); ok {
			return idx
		}
	}
	return 0
}
