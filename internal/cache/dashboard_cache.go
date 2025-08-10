package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

// DashboardCache manages hierarchical caching for dashboard data
type DashboardCache struct {
	mu          sync.RWMutex
	cacheDir    string
	maxAge      time.Duration
	initialized bool
	userID      *int // Optional user ID for user-specific caching
}

// RepositoryCache contains repository overview data
type RepositoryCache struct {
	Repositories []models.Repository `json:"repositories"`
	CachedAt     time.Time           `json:"cached_at"`
	TTL          time.Duration       `json:"ttl"`
}

// RepoDataCache contains detailed data for a specific repository
type RepoDataCache struct {
	Repository string                         `json:"repository"`
	Runs       []*models.RunResponse          `json:"runs"`
	RunDetails map[string]*models.RunResponse `json:"run_details"`
	CachedAt   time.Time                      `json:"cached_at"`
	TTL        time.Duration                  `json:"ttl"`
}

var dashboardCache *DashboardCache

// InitializeDashboardCache sets up the dashboard cache system
func InitializeDashboardCache() error {
	return InitializeDashboardCacheForUser(nil)
}

// InitializeDashboardCacheForUser sets up the dashboard cache system for a specific user
func InitializeDashboardCacheForUser(userID *int) error {
	if dashboardCache != nil && dashboardCache.initialized {
		return nil
	}

	cacheDir, err := getDashboardCacheDirForUser(userID)
	if err != nil {
		return err
	}

	dashboardCache = &DashboardCache{
		cacheDir: cacheDir,
		maxAge:   5 * time.Minute, // Cache for 5 minutes
		userID:   userID,
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(dashboardCache.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create dashboard cache directory: %w", err)
	}

	dashboardCache.initialized = true
	return nil
}

// GetRepositoryOverview returns cached repository overview data
func GetRepositoryOverview() ([]models.Repository, bool, error) {
	if err := InitializeDashboardCache(); err != nil {
		return nil, false, err
	}

	dashboardCache.mu.RLock()
	defer dashboardCache.mu.RUnlock()

	repoFile := filepath.Join(dashboardCache.cacheDir, "repos.json")

	// Check if file exists
	if _, err := os.Stat(repoFile); os.IsNotExist(err) {
		return nil, false, nil
	}

	// Read and parse the file
	data, err := os.ReadFile(repoFile)
	if err != nil {
		return nil, false, err
	}

	var repoCache RepositoryCache
	if err := json.Unmarshal(data, &repoCache); err != nil {
		return nil, false, err
	}

	// Check if cache is still valid
	if time.Since(repoCache.CachedAt) > dashboardCache.maxAge {
		return nil, false, nil // Cache expired
	}

	return repoCache.Repositories, true, nil
}

// SetRepositoryOverview caches repository overview data
func SetRepositoryOverview(repositories []models.Repository) error {
	if err := InitializeDashboardCache(); err != nil {
		return err
	}

	dashboardCache.mu.Lock()
	defer dashboardCache.mu.Unlock()

	repoCache := RepositoryCache{
		Repositories: repositories,
		CachedAt:     time.Now(),
		TTL:          dashboardCache.maxAge,
	}

	data, err := json.MarshalIndent(repoCache, "", "  ")
	if err != nil {
		return err
	}

	repoFile := filepath.Join(dashboardCache.cacheDir, "repos.json")
	return os.WriteFile(repoFile, data, 0644)
}

// GetRepositoryData returns cached data for a specific repository
func GetRepositoryData(repoName string) ([]*models.RunResponse, map[string]*models.RunResponse, bool, error) {
	if err := InitializeDashboardCache(); err != nil {
		return nil, nil, false, err
	}

	dashboardCache.mu.RLock()
	defer dashboardCache.mu.RUnlock()

	// Sanitize repo name for filename
	safeRepoName := sanitizeFilename(repoName)
	repoFile := filepath.Join(dashboardCache.cacheDir, fmt.Sprintf("repo_%s.json", safeRepoName))

	// Check if file exists
	if _, err := os.Stat(repoFile); os.IsNotExist(err) {
		return nil, nil, false, nil
	}

	// Read and parse the file
	data, err := os.ReadFile(repoFile)
	if err != nil {
		return nil, nil, false, err
	}

	var repoData RepoDataCache
	if err := json.Unmarshal(data, &repoData); err != nil {
		return nil, nil, false, err
	}

	// Check if cache is still valid
	if time.Since(repoData.CachedAt) > dashboardCache.maxAge {
		return nil, nil, false, nil // Cache expired
	}

	return repoData.Runs, repoData.RunDetails, true, nil
}

// SetRepositoryData caches data for a specific repository
func SetRepositoryData(repoName string, runs []*models.RunResponse, details map[string]*models.RunResponse) error {
	if err := InitializeDashboardCache(); err != nil {
		return err
	}

	dashboardCache.mu.Lock()
	defer dashboardCache.mu.Unlock()

	repoData := RepoDataCache{
		Repository: repoName,
		Runs:       runs,
		RunDetails: details,
		CachedAt:   time.Now(),
		TTL:        dashboardCache.maxAge,
	}

	data, err := json.MarshalIndent(repoData, "", "  ")
	if err != nil {
		return err
	}

	// Sanitize repo name for filename
	safeRepoName := sanitizeFilename(repoName)
	repoFile := filepath.Join(dashboardCache.cacheDir, fmt.Sprintf("repo_%s.json", safeRepoName))
	return os.WriteFile(repoFile, data, 0644)
}

// BuildRepositoryOverviewFromRuns builds repository overview from run data
func BuildRepositoryOverviewFromRuns(runs []*models.RunResponse) []models.Repository {
	repoMap := make(map[string]*models.Repository)
	repoIDNameMap := make(map[int]string) // Track repo ID to name mapping

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

	// Convert map to slice
	repositories := make([]models.Repository, 0, len(repoMap))
	for _, repo := range repoMap {
		repositories = append(repositories, *repo)
	}

	return repositories
}

// FilterRunsByRepository filters runs for a specific repository
func FilterRunsByRepository(runs []*models.RunResponse, repoName string) []*models.RunResponse {
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

// GetAllCachedRepositoryData returns all cached repository data for dashboard initialization
func GetAllCachedRepositoryData() (map[string][]*models.RunResponse, error) {
	if err := InitializeDashboardCache(); err != nil {
		return nil, err
	}

	dashboardCache.mu.RLock()
	defer dashboardCache.mu.RUnlock()

	result := make(map[string][]*models.RunResponse)

	// Read all repo_*.json files
	files, err := filepath.Glob(filepath.Join(dashboardCache.cacheDir, "repo_*.json"))
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files that can't be read
		}

		var repoData RepoDataCache
		if err := json.Unmarshal(data, &repoData); err != nil {
			continue // Skip files that can't be parsed
		}

		// Check if cache is still valid
		if time.Since(repoData.CachedAt) <= dashboardCache.maxAge {
			result[repoData.Repository] = repoData.Runs
		}
	}

	return result, nil
}

// InvalidateRepositoryCache removes cached data for a specific repository
func InvalidateRepositoryCache(repoName string) error {
	if err := InitializeDashboardCache(); err != nil {
		return err
	}

	dashboardCache.mu.Lock()
	defer dashboardCache.mu.Unlock()

	safeRepoName := sanitizeFilename(repoName)
	repoFile := filepath.Join(dashboardCache.cacheDir, fmt.Sprintf("repo_%s.json", safeRepoName))

	if _, err := os.Stat(repoFile); err == nil {
		return os.Remove(repoFile)
	}

	return nil // File doesn't exist, nothing to do
}

// InvalidateAllCache removes all dashboard cache files
func InvalidateAllDashboardCache() error {
	if err := InitializeDashboardCache(); err != nil {
		return err
	}

	dashboardCache.mu.Lock()
	defer dashboardCache.mu.Unlock()

	return os.RemoveAll(dashboardCache.cacheDir)
}

// InitializeDashboardForUser reinitializes the dashboard cache for a specific user
func InitializeDashboardForUser(userID *int) error {
	// Clear current dashboard cache first
	if dashboardCache != nil {
		dashboardCache.mu.Lock()
		dashboardCache.initialized = false
		dashboardCache.mu.Unlock()
	}
	dashboardCache = nil

	// Initialize with user-specific cache
	return InitializeDashboardCacheForUser(userID)
}

// sanitizeFilename makes a string safe for use as a filename
func sanitizeFilename(name string) string {
	// Replace common problematic characters
	safe := name
	safe = filepath.Base(safe) // Remove any path components

	// Replace slashes and other problematic characters
	replacements := map[string]string{
		"/":  "_",
		"\\": "_",
		":":  "_",
		"*":  "_",
		"?":  "_",
		"\"": "_",
		"<":  "_",
		">":  "_",
		"|":  "_",
		" ":  "_",
	}

	for old, new := range replacements {
		safe = strings.ReplaceAll(safe, old, new)
	}

	return safe
}

// getDashboardCacheDirForUser returns the dashboard cache directory path for a specific user
func getDashboardCacheDirForUser(userID *int) (string, error) {
	// Use os.UserCacheDir for cross-platform compatibility (same as persistent cache)
	baseDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	var cacheDir string
	if userID != nil {
		// User-specific cache directory following the same pattern as persistent cache
		cacheDir = filepath.Join(baseDir, "repobird", "users", fmt.Sprintf("user-%d", *userID), "dashboard")
	} else {
		// Fallback to shared cache directory for backward compatibility
		cacheDir = filepath.Join(baseDir, "repobird", "shared", "dashboard")
	}

	return cacheDir, nil
}
