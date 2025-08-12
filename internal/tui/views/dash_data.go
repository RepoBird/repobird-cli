package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// syncFileHashes syncs file hashes from the API on startup
func (d *DashboardView) syncFileHashes() tea.Cmd {
	return func() tea.Msg {
		// Create file hash cache instance
		// File hash cache is now embedded in SimpleCache
		// No need to sync separately - cache handles this
		debug.LogToFile("DEBUG: Using embedded cache for file hashes\n")

		// Return a proper message instead of nil
		return syncFileHashesMsg{}
	}
}

// loadUserInfo loads user information from the API
func (d *DashboardView) loadUserInfo() tea.Cmd {
	return func() tea.Msg {
		// First check if we have cached user info
		if cachedInfo := d.cache.GetUserInfo(); cachedInfo != nil {
			return dashboardUserInfoLoadedMsg{
				userInfo: cachedInfo,
				error:    nil,
			}
		}

		// Fetch from API if not cached
		userInfo, err := d.client.GetUserInfo()
		if err == nil && userInfo != nil {
			// Cache the user info
			d.cache.SetUserInfo(userInfo)
		}
		return dashboardUserInfoLoadedMsg{
			userInfo: userInfo,
			error:    err,
		}
	}
}

// loadDashboardData loads data from cache or API
func (d *DashboardView) loadDashboardData() tea.Cmd {
	return func() tea.Msg {
		debug.LogToFilef("\n[LOAD DASHBOARD DATA] Starting...\n")

		// First try to load from run cache which should always have data
		runs, cached, detailsCache := d.cache.GetCachedList()
		debug.LogToFilef("  Cache check: cached=%v, runs=%d, details=%d\n", cached, len(runs), len(detailsCache))

		if cached && len(runs) > 0 {
			// Validate that cached data is not test data
			isValidCache := true
			for _, run := range runs {
				// Skip test data (runs with "test-" prefix or empty repository)
				if strings.HasPrefix(run.ID, "test-") || run.Repository == "" {
					isValidCache = false
					debug.LogToFilef("DEBUG: Skipping invalid cached run: ID=%s, Repository=%s\n", run.ID, run.Repository)
					break
				}
			}

			if isValidCache {
				// Convert to pointer slice
				allRuns := make([]*models.RunResponse, len(runs))
				for i, run := range runs {
					allRuns[i] = &run
				}

				// Try to get cached repository overview
				repositories, repoCached := d.cache.GetRepositoryOverview()
				if !repoCached || len(repositories) == 0 {
					// Build repositories from runs if not cached
					repositories = d.cache.BuildRepositoryOverviewFromRuns(allRuns)
					d.cache.SetRepositoryOverview(repositories)
				}

				return dashboardDataLoadedMsg{
					repositories: repositories,
					allRuns:      allRuns,
					detailsCache: detailsCache,
					error:        nil,
				}
			} else {
				// Clear invalid cache and continue to API fetch
				d.cache.Clear()
				debug.LogToFilef("DEBUG: Cleared invalid cache data, fetching from API\n")
			}
		}

		// No cache, fetch from API
		debug.LogToFilef("  No valid cache, fetching from API...\n")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Store API repositories for ID mapping
		d.apiRepositories = make(map[int]models.APIRepository)

		// First, try to get repositories from API
		debug.LogToFilef("  Calling ListRepositories API...\n")
		apiRepositories, err := d.client.ListRepositories(ctx)
		if err != nil {
			debug.LogToFilef("  ListRepositories failed: %v\n", err)
			// Fall back to building repos from runs if repository API fails
			return d.loadFromRunsOnly()
		}
		debug.LogToFilef("  ListRepositories succeeded: %d repos\n", len(apiRepositories))

		// Store API repositories by ID for quick lookup
		for _, apiRepo := range apiRepositories {
			d.apiRepositories[apiRepo.ID] = apiRepo
		}

		// Convert API repositories to dashboard models
		repositories := make([]models.Repository, 0, len(apiRepositories))
		for _, apiRepo := range apiRepositories {
			// Construct full repository name
			repoName := apiRepo.Name
			if repoName == "" {
				repoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
			}

			repositories = append(repositories, models.Repository{
				Name:        repoName,
				Description: "",                // API doesn't provide description
				RunCounts:   models.RunStats{}, // Will be populated below
			})
		}

		// Get runs to populate repository statistics
		runs, cached, detailsCache = d.cache.GetCachedList()
		if !cached || len(runs) == 0 {
			// Fetch runs from API using context-aware method with timeout
			debug.LogToFilef("  Calling ListRuns API with context...\n")

			// Create context with 5-second timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Use the context-aware ListRuns method
			listResp, err := d.client.ListRuns(ctx, 1, 1000) // page 1, limit 1000
			if err != nil {
				debug.LogToFilef("  ListRuns failed: %v\n", err)
				// Still return repos even if runs fail
				d.cache.SetRepositoryOverview(repositories)
				return dashboardDataLoadedMsg{
					repositories: repositories,
					allRuns:      []*models.RunResponse{},
					detailsCache: detailsCache,
					error:        nil,
				}
			}

			var runsResp []*models.RunResponse
			if listResp != nil && listResp.Data != nil {
				runsResp = listResp.Data
				debug.LogToFilef("  ListRuns succeeded, got %d runs\n", len(runsResp))
			} else {
				runsResp = []*models.RunResponse{}
				debug.LogToFilef("  ListRuns returned empty response\n")
			}

			// Convert to pointer slice
			allRuns := make([]*models.RunResponse, len(runsResp))
			copy(allRuns, runsResp)

			// Update repository statistics from runs
			debug.LogToFilef("  About to call updateRepositoryStats with %d repos and %d runs\n", len(repositories), len(allRuns))
			repositories = d.updateRepositoryStats(repositories, allRuns)
			debug.LogToFilef("  updateRepositoryStats completed, got %d repos back\n", len(repositories))

			// Skip caching for now to avoid deadlock - TEMPORARY FIX
			debug.LogToFilef("  SKIPPING cache operations (temporary fix for deadlock)\n")

			// Just set overview without individual repo caching
			// d.cache.SetRepositoryOverview(repositories)

			debug.LogToFilef("  Data loaded successfully, returning message\n")
			return dashboardDataLoadedMsg{
				repositories: repositories,
				allRuns:      allRuns,
				detailsCache: detailsCache,
				error:        nil,
			}
		}

		// Use cached run data
		allRuns := make([]*models.RunResponse, len(runs))
		for i, run := range runs {
			allRuns[i] = &run
		}

		// Update repository statistics from cached runs
		repositories = d.updateRepositoryStats(repositories, allRuns)
		d.cache.SetRepositoryOverview(repositories)

		return dashboardDataLoadedMsg{
			repositories: repositories,
			allRuns:      allRuns,
			error:        nil,
		}
	}
}

// loadFromRunsOnly loads dashboard data using only runs (fallback method)
func (d *DashboardView) loadFromRunsOnly() tea.Msg {
	runs, cached, detailsCache := d.cache.GetCachedList()
	if !cached || len(runs) == 0 {
		// Fetch from API using context-aware method
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		listResp, err := d.client.ListRuns(ctx, 1, 1000) // page 1, limit 1000
		if err != nil {
			return dashboardDataLoadedMsg{
				detailsCache: make(map[string]*models.RunResponse),
				error:        err,
			}
		}

		var runsResp []*models.RunResponse
		if listResp != nil && listResp.Data != nil {
			runsResp = listResp.Data
		} else {
			runsResp = []*models.RunResponse{}
		}

		// Convert to pointer slice
		allRuns := make([]*models.RunResponse, len(runsResp))
		copy(allRuns, runsResp)

		// Build repository overview from runs
		repositories := d.cache.BuildRepositoryOverviewFromRuns(allRuns)

		// Cache the data
		d.cache.SetRepositoryOverview(repositories)

		// Cache runs by repository
		for _, repo := range repositories {
			repoRuns := d.filterRunsByRepository(allRuns, repo.Name)
			repoDetails := make(map[string]*models.RunResponse)

			// Add any cached details
			for _, run := range repoRuns {
				if detail, exists := detailsCache[run.GetIDString()]; exists {
					repoDetails[run.GetIDString()] = detail
				}
			}

			d.cache.SetRepositoryData(repo.Name, repoRuns, repoDetails)
		}

		return dashboardDataLoadedMsg{
			repositories: repositories,
			allRuns:      allRuns,
			error:        nil,
		}
	}

	// Use cached run data
	allRuns := make([]*models.RunResponse, len(runs))
	for i, run := range runs {
		allRuns[i] = &run
	}

	// Build repository overview from cached runs
	repositories := d.cache.BuildRepositoryOverviewFromRuns(allRuns)
	d.cache.SetRepositoryOverview(repositories)

	return dashboardDataLoadedMsg{
		repositories: repositories,
		allRuns:      allRuns,
		detailsCache: detailsCache,
		error:        nil,
	}
}

// updateRepositoryStats updates repository statistics from runs
func (d *DashboardView) updateRepositoryStats(repositories []models.Repository, allRuns []*models.RunResponse) []models.Repository {
	debug.LogToFilef("    [updateRepositoryStats] Starting with %d repos and %d runs\n", len(repositories), len(allRuns))
	// Create maps for quick lookup
	repoMap := make(map[string]*models.Repository)
	repoIDMap := make(map[int]*models.Repository) // Map by repo ID

	for i := range repositories {
		repoMap[repositories[i].Name] = &repositories[i]

		// Also map by ID if we have API repositories
		if d.apiRepositories != nil {
			for id, apiRepo := range d.apiRepositories {
				apiRepoName := apiRepo.Name
				if apiRepoName == "" {
					apiRepoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
				}
				if apiRepoName == repositories[i].Name {
					repoIDMap[id] = &repositories[i]
					break
				}
			}
		}
	}

	// Update statistics from runs
	for _, run := range allRuns {
		var repo *models.Repository

		// First try to match by repository name
		repoName := run.GetRepositoryName()
		if repoName != "" {
			repo = repoMap[repoName]
		}

		// If not found and we have a repo ID, try to match by ID
		if repo == nil && run.RepoID > 0 {
			repo = repoIDMap[run.RepoID]
		}

		if repo == nil {
			continue
		}

		// Update last activity if this run is more recent
		if run.UpdatedAt.After(repo.LastActivity) {
			repo.LastActivity = run.UpdatedAt
		}

		// Update run counts
		repo.RunCounts.Total++
		switch run.Status {
		case models.StatusQueued, models.StatusInitializing, models.StatusProcessing, models.StatusPostProcess:
			repo.RunCounts.Running++
		case models.StatusDone:
			repo.RunCounts.Completed++
		case models.StatusFailed:
			repo.RunCounts.Failed++
		}
	}

	debug.LogToFilef("    [updateRepositoryStats] Completed, returning %d repos\n", len(repositories))
	return repositories
}

// filterRunsByRepository filters runs by repository name
func (d *DashboardView) filterRunsByRepository(runs []*models.RunResponse, repoName string) []*models.RunResponse {
	var filtered []*models.RunResponse
	repoIDSet := make(map[int]bool)

	// First pass: collect all runs that match by name and build ID set
	for _, run := range runs {
		runRepoName := run.GetRepositoryName()
		if runRepoName == repoName {
			filtered = append(filtered, run)
			// If this run has a repo ID, track it
			if run.RepoID > 0 {
				repoIDSet[run.RepoID] = true
			}
		}
	}

	// Second pass: also include runs that match by repo ID
	if len(repoIDSet) > 0 {
		for _, run := range runs {
			// Skip if already included
			if run.GetRepositoryName() == repoName {
				continue
			}
			// Include if repo ID matches
			if run.RepoID > 0 && repoIDSet[run.RepoID] {
				filtered = append(filtered, run)
			}
		}
	}

	return filtered
}

// selectRepository loads data for a specific repository
func (d *DashboardView) selectRepository(repo *models.Repository) tea.Cmd {
	if repo == nil {
		return nil
	}

	return func() tea.Msg {
		// Filter runs for this repository
		var filteredRuns []*models.RunResponse

		// First try to match by repository name
		matchCount := 0
		for _, run := range d.allRuns {
			runRepoName := run.GetRepositoryName()
			if runRepoName == repo.Name {
				filteredRuns = append(filteredRuns, run)
				matchCount++
				continue
			}

			// Also try to match by repo ID if we have API repositories
			if run.RepoID > 0 && d.apiRepositories != nil {
				if apiRepo, exists := d.apiRepositories[run.RepoID]; exists {
					apiRepoName := apiRepo.Name
					if apiRepoName == "" {
						apiRepoName = fmt.Sprintf("%s/%s", apiRepo.RepoOwner, apiRepo.RepoName)
					}
					if apiRepoName == repo.Name {
						filteredRuns = append(filteredRuns, run)
						matchCount++
					}
				}
			}
		}

		return dashboardRepositorySelectedMsg{
			repository: repo,
			runs:       filteredRuns,
		}
	}
}
