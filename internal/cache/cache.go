package cache

import (
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

// Global cache for run list and details to persist across view transitions
type GlobalCache struct {
	mu sync.RWMutex
	
	// List cache
	runs       []models.RunResponse
	cached     bool
	cachedAt   time.Time
	
	// Details cache
	details    map[string]*models.RunResponse
	
	// UI state
	selectedIndex int
}

var globalCache = &GlobalCache{
	details: make(map[string]*models.RunResponse),
}

// GetCachedList returns the cached run list if it's still valid (< 30 seconds old)
func GetCachedList() (runs []models.RunResponse, cached bool, cachedAt time.Time, details map[string]*models.RunResponse, selectedIndex int) {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()
	
	if globalCache.cached && time.Since(globalCache.cachedAt) < 30*time.Second {
		// Return copies to avoid concurrent modification
		runsCopy := make([]models.RunResponse, len(globalCache.runs))
		copy(runsCopy, globalCache.runs)
		
		detailsCopy := make(map[string]*models.RunResponse)
		for k, v := range globalCache.details {
			if v != nil {
				detailsCopy[k] = v
			}
		}
		
		return runsCopy, true, globalCache.cachedAt, detailsCopy, globalCache.selectedIndex
	}
	
	return nil, false, time.Time{}, make(map[string]*models.RunResponse), 0
}

// SetCachedList updates the global run list cache
func SetCachedList(runs []models.RunResponse, details map[string]*models.RunResponse) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	
	globalCache.runs = make([]models.RunResponse, len(runs))
	copy(globalCache.runs, runs)
	globalCache.cached = true
	globalCache.cachedAt = time.Now()
	
	// Merge the existing details with new ones
	if globalCache.details == nil {
		globalCache.details = make(map[string]*models.RunResponse)
	}
	
	if details != nil {
		for k, v := range details {
			if v != nil {
				globalCache.details[k] = v
			}
		}
	}
}

// AddCachedDetail adds a single run detail to the cache
func AddCachedDetail(runID string, run *models.RunResponse) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	
	if globalCache.details == nil {
		globalCache.details = make(map[string]*models.RunResponse)
	}
	
	if run != nil {
		globalCache.details[runID] = run
	}
}

// SetSelectedIndex updates the selected index in the cache
func SetSelectedIndex(index int) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	
	globalCache.selectedIndex = index
}

// ClearCache clears all cached data (useful for testing)
func ClearCache() {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()
	
	globalCache.runs = nil
	globalCache.cached = false
	globalCache.cachedAt = time.Time{}
	globalCache.details = make(map[string]*models.RunResponse)
	globalCache.selectedIndex = 0
}