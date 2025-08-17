// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"fmt"
	"sort"
	"sync"

	"github.com/repobird/repobird-cli/internal/models"
)

// HybridCache combines permanent disk storage and session memory caching
type HybridCache struct {
	permanent *PermanentCache
	session   *SessionCache
	userID    string
	mu        sync.RWMutex
}

// NewHybridCache creates a new hybrid cache for the specified user
func NewHybridCache(userID string) (*HybridCache, error) {
	if userID == "" {
		userID = "anonymous"
	}

	permanent, err := NewPermanentCache(userID)
	if err != nil {
		// If permanent cache fails, continue with session-only
		permanent = nil
	}

	session := NewSessionCache()

	return &HybridCache{
		permanent: permanent,
		session:   session,
		userID:    userID,
	}, nil
}

// GetRun checks both caches intelligently based on run state
func (h *HybridCache) GetRun(id string) (*models.RunResponse, bool) {
	// Check permanent cache first for terminal runs
	if h.permanent != nil {
		if run, found := h.permanent.GetRun(id); found {
			return run, true
		}
	}

	// Check session cache for active runs
	if run, found := h.session.GetRun(id); found {
		return run, true
	}

	return nil, false
}

// SetRun routes to appropriate cache based on status and age
func (h *HybridCache) SetRun(run models.RunResponse) error {
	// Make routing decision once - no state changes after initial decision
	if shouldPermanentlyCache(run) {
		// Direct to permanent, no session interaction
		if h.permanent != nil {
			// Optionally remove from session cache if it exists there
			// This is safe as it's a separate operation
			go func() {
				_ = h.session.InvalidateRun(run.ID)
			}()
			return h.permanent.SetRun(run)
		}
	}

	// Direct to session, no permanent interaction
	return h.session.SetRun(run)
}

// GetRuns returns merged results from both caches
func (h *HybridCache) GetRuns() ([]models.RunResponse, bool) {
	// No mutex needed - child caches handle their own locking
	runMap := make(map[string]models.RunResponse)

	// Get from caches in parallel to avoid sequential blocking
	var wg sync.WaitGroup
	var mapMu sync.Mutex

	// Permanent cache goroutine
	if h.permanent != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if permanentRuns, found := h.permanent.GetAllRuns(); found {
				mapMu.Lock()
				for _, run := range permanentRuns {
					runMap[run.ID] = run
				}
				mapMu.Unlock()
			}
		}()
	}

	// Session cache goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if sessionRuns, found := h.session.GetRuns(); found {
			mapMu.Lock()
			for _, run := range sessionRuns {
				// Session cache has priority for active runs
				runMap[run.ID] = run
			}
			mapMu.Unlock()
		}
	}()

	wg.Wait()

	// Convert map to slice and sort by creation time (newest first)
	runs := make([]models.RunResponse, 0, len(runMap))
	for _, run := range runMap {
		runs = append(runs, run)
	}

	// Sort runs by creation time (newest first)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].CreatedAt.After(runs[j].CreatedAt)
	})

	return runs, len(runs) > 0
}

// SetRuns stores multiple runs, routing each to the appropriate cache
func (h *HybridCache) SetRuns(runs []models.RunResponse) error {
	// Separate runs by cache destination with single decision
	var sessionRuns []models.RunResponse
	var permanentRuns []models.RunResponse

	for _, run := range runs {
		if shouldPermanentlyCache(run) {
			permanentRuns = append(permanentRuns, run)
		} else {
			sessionRuns = append(sessionRuns, run)
		}
	}

	// Store in parallel to avoid sequential blocking
	var wg sync.WaitGroup
	var permErr, sessErr error

	// Store permanent runs in background
	if h.permanent != nil && len(permanentRuns) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, run := range permanentRuns {
				if err := h.permanent.SetRun(run); err != nil {
					// Log error but continue
					permErr = err
				}
			}
		}()
	}

	// Store session runs in background
	if len(sessionRuns) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessErr = h.session.SetRuns(sessionRuns)
		}()
	}

	wg.Wait()

	// Return session error if any (higher priority)
	if sessErr != nil {
		return sessErr
	}
	return permErr
}

// InvalidateRun removes a run from both caches
func (h *HybridCache) InvalidateRun(id string) error {
	// Remove from session cache
	_ = h.session.InvalidateRun(id)

	// Remove from permanent cache
	if h.permanent != nil {
		_ = h.permanent.InvalidateRun(id)
	}

	return nil
}

// InvalidateActiveRuns only clears non-terminal runs from session cache
func (h *HybridCache) InvalidateActiveRuns() error {
	return h.session.InvalidateActiveRuns()
}

// GetRepository retrieves cached repository data
func (h *HybridCache) GetRepository(name string) (*models.Repository, bool) {
	// Repositories are not cached in this implementation yet
	// This is a placeholder for future implementation
	return nil, false
}

// SetRepository caches repository data
func (h *HybridCache) SetRepository(repo models.Repository) error {
	// Placeholder for future implementation
	return nil
}

// GetRepositoryList retrieves cached repository list
func (h *HybridCache) GetRepositoryList() ([]string, bool) {
	if h.permanent != nil {
		return h.permanent.GetRepositoryList()
	}
	return nil, false
}

// SetRepositoryList caches repository list
func (h *HybridCache) SetRepositoryList(repos []string) error {
	if h.permanent != nil {
		return h.permanent.SetRepositoryList(repos)
	}
	return nil
}

// GetUserInfo retrieves cached user info (checks permanent first)
func (h *HybridCache) GetUserInfo() (*models.UserInfo, bool) {
	if h.permanent != nil {
		return h.permanent.GetUserInfo()
	}
	return nil, false
}

// SetUserInfo caches user info to permanent storage
func (h *HybridCache) SetUserInfo(info *models.UserInfo) error {
	if h.permanent != nil {
		return h.permanent.SetUserInfo(info)
	}
	return fmt.Errorf("permanent cache not available")
}

// InvalidateUserInfo removes user info from cache
func (h *HybridCache) InvalidateUserInfo() error {
	// For permanent cache, we would need to implement this
	// For now, just clear the file
	if h.permanent != nil {
		// This could be implemented in PermanentCache
		// For now, we'll just overwrite with nil on next set
	}
	return nil
}

// GetAuthCache retrieves cached authentication info with timestamp
func (h *HybridCache) GetAuthCache() (*AuthCache, bool) {
	if h.permanent != nil {
		return h.permanent.GetAuthCache()
	}
	return nil, false
}

// SetAuthCache stores authentication info with timestamp
func (h *HybridCache) SetAuthCache(userInfo *models.UserInfo) error {
	if h.permanent != nil {
		return h.permanent.SetAuthCache(userInfo)
	}
	return fmt.Errorf("permanent cache not available")
}

// IsAuthCacheValid checks if cached authentication is still valid
func (h *HybridCache) IsAuthCacheValid() bool {
	if h.permanent != nil {
		return h.permanent.IsAuthCacheValid()
	}
	return false
}

// GetLastUsedRepository retrieves the last repository used for trigger runs
func (h *HybridCache) GetLastUsedRepository() (string, bool) {
	if h.permanent != nil {
		return h.permanent.GetLastUsedRepository()
	}
	return "", false
}

// SetLastUsedRepository stores the last repository used for trigger runs
func (h *HybridCache) SetLastUsedRepository(repository string) error {
	if h.permanent != nil {
		return h.permanent.SetLastUsedRepository(repository)
	}
	return nil
}

// GetFileHash retrieves cached file hash from permanent storage
func (h *HybridCache) GetFileHash(path string) (string, bool) {
	if h.permanent != nil {
		return h.permanent.GetFileHash(path)
	}
	return "", false
}

// SetFileHash caches file hash to permanent storage
func (h *HybridCache) SetFileHash(path string, hash string) error {
	if h.permanent != nil {
		return h.permanent.SetFileHash(path, hash)
	}
	return nil
}

// GetAllFileHashes returns all cached file hashes
func (h *HybridCache) GetAllFileHashes() map[string]string {
	if h.permanent != nil {
		return h.permanent.GetAllFileHashes()
	}
	return make(map[string]string)
}

// GetDashboardData retrieves cached dashboard data from session
func (h *HybridCache) GetDashboardData() (*DashboardData, bool) {
	return h.session.GetDashboardData()
}

// SetDashboardData caches dashboard data in session
func (h *HybridCache) SetDashboardData(data *DashboardData) error {
	return h.session.SetDashboardData(data)
}

// Clear removes all cached data
func (h *HybridCache) Clear() error {
	// Clear session cache
	_ = h.session.Clear()

	// Clear permanent cache
	if h.permanent != nil {
		_ = h.permanent.Clear()
	}

	return nil
}

// Close releases resources
func (h *HybridCache) Close() error {
	// Close session cache
	_ = h.session.Close()

	// Close permanent cache (no-op currently)
	if h.permanent != nil {
		_ = h.permanent.Close()
	}

	return nil
}

// GetStats returns cache statistics
func (h *HybridCache) GetStats() CacheStats {
	stats := CacheStats{}

	// Count permanent runs
	if h.permanent != nil {
		if runs, found := h.permanent.GetAllRuns(); found {
			stats.PermanentRuns = len(runs)
		}
	}

	// Count active runs in session
	if runs, found := h.session.GetRuns(); found {
		stats.ActiveRuns = len(runs)
	}

	// Count repositories
	if h.permanent != nil {
		if repos, found := h.permanent.GetRepositoryList(); found {
			stats.Repositories = len(repos)
		}
	}

	// Note: Disk and memory usage calculation would require more sophisticated tracking

	return stats
}

// CacheStats holds cache statistics
type CacheStats struct {
	PermanentRuns    int
	ActiveRuns       int
	Repositories     int
	DiskUsageBytes   int64
	MemoryUsageBytes int64
	HitRate          float64
}
