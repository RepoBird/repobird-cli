package cache

import (
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
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
}

var globalCache = &GlobalCache{
	details:         make(map[string]*models.RunResponse),
	detailsAt:       make(map[string]time.Time),
	terminalDetails: make(map[string]*models.RunResponse),
}

// GetCachedList returns the cached run list if it's still valid (< 30 seconds old)
func GetCachedList() (runs []models.RunResponse, cached bool, cachedAt time.Time, details map[string]*models.RunResponse, selectedIndex int) {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	if globalCache.cached && len(globalCache.runs) > 0 {
		// Always return cached data if available, regardless of age
		// Only auto-refresh on explicit refresh action or poll for active runs
		runsCopy := make([]models.RunResponse, len(globalCache.runs))
		copy(runsCopy, globalCache.runs)

		// Merge terminal (permanent) and active (temporary) details caches
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
	if details != nil {
		for k, v := range details {
			if v != nil {
				if isTerminalStatus(v.Status) {
					// Store terminal runs permanently
					globalCache.terminalDetails[k] = v
				} else {
					// Store active runs temporarily
					globalCache.details[k] = v
					globalCache.detailsAt[k] = now
				}
			}
		}
	}
}

// AddCachedDetail adds a single run detail to the cache
func AddCachedDetail(runID string, run *models.RunResponse) {
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
		} else {
			// Store active runs temporarily
			globalCache.details[runID] = run
			globalCache.detailsAt[runID] = time.Now()
		}
	}
}

// SetSelectedIndex updates the selected index in the cache
func SetSelectedIndex(index int) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.selectedIndex = index
}

// SaveFormData saves the current form state
func SaveFormData(data *FormData) {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.formData = data
}

// GetFormData retrieves the saved form state
func GetFormData() *FormData {
	globalCache.mu.RLock()
	defer globalCache.mu.RUnlock()

	return globalCache.formData
}

// ClearFormData clears the saved form state
func ClearFormData() {
	globalCache.mu.Lock()
	defer globalCache.mu.Unlock()

	globalCache.formData = nil
}

// ClearCache clears all cached data (useful for testing)
func ClearCache() {
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

// ClearActiveCache clears only the temporary cache (keeps terminal runs)
func ClearActiveCache() {
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
