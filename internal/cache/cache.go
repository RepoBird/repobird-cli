// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// isTerminalStatus returns true if the run status never changes
func isTerminalStatus(status models.RunStatus) bool {
	return status == models.StatusDone || status == models.StatusFailed
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

// Global cache for run list and details to persist across view transitions
type GlobalCache struct {
	mu sync.RWMutex

	// List cache
	runs     []models.RunResponse
	cached   bool
	cachedAt time.Time

	// Details cache - temporary cache for active runs
	details   map[string]*models.RunResponse
	detailsAt map[string]time.Time

	// Persistent cache for terminal status runs (DONE/FAILED) - never expires
	terminalDetails map[string]*models.RunResponse

	// UI state
	selectedIndex int

	// Form persistence
	formData *FormData

	// Persistent file cache
	persistentCache *PersistentCache

	// File hash cache for duplicate detection
	fileHashCache *FileHashCache

	// In-memory repository history for testing
	repoHistory []string

	// User info cache
	userInfo     *models.UserInfo
	userInfoTime time.Time
}

var globalCache *GlobalCache
var globalCacheOnce sync.Once

// DEPRECATED: Global cache is being phased out in favor of embedded cache in TUI views
// Keeping the variable for backward compatibility with non-TUI code

// ensureGlobalCache ensures the global cache is initialized (for backward compatibility)
func ensureGlobalCache() {
	globalCacheOnce.Do(func() {
		initializeCache()
	})
}

func initializeCache() {
	initializeCacheForUser(nil)
}

// initializeCacheForUser initializes cache for a specific user
func initializeCacheForUser(userID *int) {
	pc, err := NewPersistentCacheForUser(userID)
	if err != nil {
		// Fall back to memory-only cache if persistent cache fails
		pc = nil
	}

	globalCache = &GlobalCache{
		details:         make(map[string]*models.RunResponse),
		detailsAt:       make(map[string]time.Time),
		terminalDetails: make(map[string]*models.RunResponse),
		persistentCache: pc,
		fileHashCache:   NewFileHashCacheForUser(userID),
	}

	// Load persisted terminal runs on startup
	if pc != nil {
		if terminalRuns, err := pc.LoadAllTerminalRuns(); err == nil {
			globalCache.terminalDetails = terminalRuns
			debug.LogToFilef("DEBUG: Loaded %d terminal runs from persistent cache\n", len(terminalRuns))
		} else {
			debug.LogToFilef("DEBUG: Failed to load terminal runs from persistent cache: %v\n", err)
		}
		// Clean up old cache entries (older than 30 days)
		go func() {
			_ = pc.CleanOldCache(30 * 24 * time.Hour)
		}()
	}
}

// GetCachedList returns the cached run list if it's still valid (< 30 seconds old)
func GetCachedList() (
	runs []models.RunResponse,
	cached bool,
	cachedAt time.Time,
	details map[string]*models.RunResponse,
	selectedIndex int,
) {
	ensureGlobalCache()
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	// Always build details cache from terminal and active runs
	detailsCopy := make(map[string]*models.RunResponse)

	// First add terminal runs (these never expire)
	for k, v := range globalCache.terminalDetails {
		if v != nil {
			detailsCopy[k] = v
		}
	}

	// Then add active runs (with 30-second expiry)
	now := time.Now()
	for k, v := range globalCache.details {
		if v != nil {
			// Check if this active run detail is still fresh
			if cachedAt, exists := globalCache.detailsAt[k]; exists && now.Sub(cachedAt) < 30*time.Second {
				detailsCopy[k] = v
			}
		}
	}

	if globalCache.cached && len(globalCache.runs) > 0 {
		// Return cached runs + details
		runsCopy := make([]models.RunResponse, len(globalCache.runs))
		copy(runsCopy, globalCache.runs)

		debug.LogToFilef("DEBUG: GetCachedList returning %d runs, %d cached details (%d terminal + %d active)\n",
			len(runsCopy), len(detailsCopy), len(globalCache.terminalDetails), len(globalCache.details))
		return runsCopy, true, globalCache.cachedAt, detailsCopy, globalCache.selectedIndex
	}

	// No cached runs, but still return available details cache
	debug.LogToFilef("DEBUG: GetCachedList - no cached runs but returning %d details (cached=%t, runs=%d, terminalDetails=%d)\n",
		len(detailsCopy), globalCache.cached, len(globalCache.runs), len(globalCache.terminalDetails))
	return nil, false, time.Time{}, detailsCopy, 0
}

// SetCachedList updates the global run list cache
func SetCachedList(runs []models.RunResponse, details map[string]*models.RunResponse) {
	ensureGlobalCache()
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.runs = make([]models.RunResponse, len(runs))
	copy(globalCache.runs, runs)
	globalCache.cached = true
	globalCache.cachedAt = time.Now()

	// Initialize maps if needed
	if globalCache.details == nil {
		globalCache.details = make(map[string]*models.RunResponse)
	}
	if globalCache.detailsAt == nil {
		globalCache.detailsAt = make(map[string]time.Time)
	}
	if globalCache.terminalDetails == nil {
		globalCache.terminalDetails = make(map[string]*models.RunResponse)
	}

	// Merge the existing details with new ones, separating terminal vs active
	now := time.Now()
	for k, v := range details {
			if v != nil {
				if isTerminalStatus(v.Status) {
					// Store terminal runs permanently
					globalCache.terminalDetails[k] = v
					// Also persist to disk if available
					if globalCache.persistentCache != nil {
						go func(run *models.RunResponse) {
							_ = globalCache.persistentCache.SaveRun(run)
						}(v)
					}
				} else {
					// Store active runs temporarily
					globalCache.details[k] = v
					globalCache.detailsAt[k] = now
				}
			}
		}
	}

// AddCachedDetail adds a single run detail to the cache
func AddCachedDetail(runID string, run *models.RunResponse) {
	ensureGlobalCache()
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	if run != nil {
		// Initialize maps if needed
		if globalCache.details == nil {
			globalCache.details = make(map[string]*models.RunResponse)
		}
		if globalCache.detailsAt == nil {
			globalCache.detailsAt = make(map[string]time.Time)
		}
		if globalCache.terminalDetails == nil {
			globalCache.terminalDetails = make(map[string]*models.RunResponse)
		}

		if isTerminalStatus(run.Status) {
			// Store terminal runs permanently
			globalCache.terminalDetails[runID] = run
			// Remove from active cache if it was there
			delete(globalCache.details, runID)
			delete(globalCache.detailsAt, runID)
			// Also persist to disk if available
			if globalCache.persistentCache != nil {
				go func(r *models.RunResponse) {
					_ = globalCache.persistentCache.SaveRun(r)
				}(run)
			}
		} else {
			// Store active runs temporarily
			globalCache.details[runID] = run
			globalCache.detailsAt[runID] = time.Now()
		}
	}
}

// SetSelectedIndex updates the selected index in the cache
func SetSelectedIndex(index int) {
	ensureGlobalCache()
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.selectedIndex = index
}

// SaveFormData saves the current form state
func SaveFormData(data *FormData) {
	ensureGlobalCache()
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.formData = data
}

// GetFormData retrieves the saved form state
func GetFormData() *FormData {
	ensureGlobalCache()
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	return globalCache.formData
}

// ClearFormData clears the saved form state
func ClearFormData() {
	ensureGlobalCache()
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.formData = nil
}

// ClearCache clears all cached data (useful for testing)
func ClearCache() {
	ensureGlobalCache()
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.runs = nil
	globalCache.cached = false
	globalCache.cachedAt = time.Time{}
	globalCache.details = make(map[string]*models.RunResponse)
	globalCache.detailsAt = make(map[string]time.Time)
	globalCache.terminalDetails = make(map[string]*models.RunResponse)
	globalCache.selectedIndex = 0
	globalCache.formData = nil
}

// InitializeCacheForUser reinitializes the cache for a specific user
func InitializeCacheForUser(userID *int) {
	ensureGlobalCache()
	// Clear current cache first (but preserve user info, form data, and terminal details if same user)
	var savedUserInfo *models.UserInfo
	var savedUserInfoTime time.Time
	var savedFormData *FormData
	var savedTerminalDetails map[string]*models.RunResponse

	if globalCache != nil {
		globalCache.mu.RLock()

		if globalCache.userInfo != nil && userID != nil && globalCache.userInfo.ID == *userID {
			// Save user info, form data, and terminal details if it's the same user
			savedUserInfo = globalCache.userInfo
			savedUserInfoTime = globalCache.userInfoTime
			// Only preserve form data for the same user
			savedFormData = globalCache.formData
			// Preserve terminal details for the same user
			if globalCache.terminalDetails != nil {
				savedTerminalDetails = make(map[string]*models.RunResponse)
				for k, v := range globalCache.terminalDetails {
					savedTerminalDetails[k] = v
				}
			}
		}
		globalCache.mu.RUnlock()
	}

	ClearCache()
	// Initialize with user-specific cache
	initializeCacheForUser(userID)

	// Restore user info, form data, and terminal details
	if savedUserInfo != nil || savedFormData != nil || savedTerminalDetails != nil {
		globalCache.mu.Lock()
		if savedUserInfo != nil {
			globalCache.userInfo = savedUserInfo
			globalCache.userInfoTime = savedUserInfoTime
		}
		if savedFormData != nil {
			globalCache.formData = savedFormData
		}
		if savedTerminalDetails != nil {
			// Restore terminal details cache for same user
			if globalCache.terminalDetails == nil {
				globalCache.terminalDetails = make(map[string]*models.RunResponse)
			}
			for k, v := range savedTerminalDetails {
				globalCache.terminalDetails[k] = v
			}
			debug.LogToFilef("DEBUG: Restored %d terminal details from memory for same user\n", len(savedTerminalDetails))
		}
		globalCache.mu.Unlock()
	}
}

// ClearActiveCache clears only the temporary cache (keeps terminal runs)
func ClearActiveCache() {
	ensureGlobalCache()
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.runs = nil
	globalCache.cached = false
	globalCache.cachedAt = time.Time{}
	globalCache.details = make(map[string]*models.RunResponse)
	globalCache.detailsAt = make(map[string]time.Time)
	// Keep terminalDetails - these never expire
	globalCache.selectedIndex = 0
}

// AddRepositoryToHistory adds a repository to persistent history
func AddRepositoryToHistory(repo string) error {
	ensureGlobalCache()
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	if repo == "" {
		return nil
	}

	// Try persistent cache first
	if globalCache.persistentCache != nil {
		err := globalCache.persistentCache.AddRepository(repo)
		if err == nil {
			return nil
		}
		// Fall through to in-memory if persistent fails
	}

	// Use in-memory storage as fallback
	if globalCache.repoHistory == nil {
		globalCache.repoHistory = []string{}
	}

	// Remove repo if it already exists to avoid duplicates
	for i, existing := range globalCache.repoHistory {
		if existing == repo {
			globalCache.repoHistory = append(globalCache.repoHistory[:i], globalCache.repoHistory[i+1:]...)
			break
		}
	}

	// Add to front
	globalCache.repoHistory = append([]string{repo}, globalCache.repoHistory...)

	// Limit history size
	const maxRepositoryHistory = 50
	if len(globalCache.repoHistory) > maxRepositoryHistory {
		globalCache.repoHistory = globalCache.repoHistory[:maxRepositoryHistory]
	}

	return nil
}

// GetRepositoryHistory returns the repository history, most recent first
func GetRepositoryHistory() ([]string, error) {
	ensureGlobalCache()
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	// Try persistent cache first
	if globalCache.persistentCache != nil {
		history, err := globalCache.persistentCache.GetRepositoryHistory()
		if err == nil && len(history) > 0 {
			return history, nil
		}
		// Fall through to in-memory if persistent fails or is empty
	}

	// Use in-memory storage as fallback
	if globalCache.repoHistory == nil {
		return []string{}, nil
	}

	// Return a copy to prevent external modification
	history := make([]string, len(globalCache.repoHistory))
	copy(history, globalCache.repoHistory)
	return history, nil
}

// GetMostRecentRepository returns the most recently used repository
func GetMostRecentRepository() (string, error) {
	repos, err := GetRepositoryHistory()
	if err != nil || len(repos) == 0 {
		return "", err
	}
	return repos[0], nil
}

// GetCachedUserInfo returns cached user info if available and not expired
func GetCachedUserInfo() *models.UserInfo {
	ensureGlobalCache()
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	// Cache user info for 5 minutes
	if globalCache.userInfo != nil && time.Since(globalCache.userInfoTime) < 5*time.Minute {
		return globalCache.userInfo
	}
	return nil
}

// SetCachedUserInfo stores user info in cache
func SetCachedUserInfo(userInfo *models.UserInfo) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.userInfo = userInfo
	globalCache.userInfoTime = time.Now()
}

// GetFileHashCache returns the global file hash cache instance
func GetFileHashCache() *FileHashCache {
	ensureGlobalCache()
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()
	return globalCache.fileHashCache
}
