// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/repobird/repobird-cli/internal/models"
)

// SessionCache provides in-memory caching with TTL for active/changing data
type SessionCache struct {
	cache *ttlcache.Cache[string, any]
	// Note: ttlcache is thread-safe, no additional mutex needed
}

// NewSessionCache creates a new memory-based cache for session data
func NewSessionCache() *SessionCache {
	cache := ttlcache.New[string, any](
		ttlcache.WithCapacity[string, any](1000),
	)

	go cache.Start()

	return &SessionCache{
		cache: cache,
	}
}

// GetRun retrieves a cached run from memory (only active states)
func (s *SessionCache) GetRun(id string) (*models.RunResponse, bool) {
	// ttlcache is thread-safe, no mutex needed
	item := s.cache.Get("run:" + id)
	if item == nil {
		return nil, false
	}

	run, ok := item.Value().(models.RunResponse)
	if !ok {
		return nil, false
	}

	// Only return active runs from session cache
	// (terminal or old runs should be in permanent cache)
	if shouldPermanentlyCache(run) {
		// Remove from session cache
		s.cache.Delete("run:" + id)
		return nil, false
	}

	return &run, true
}

// SetRun stores a run in memory (only active, recent states)
func (s *SessionCache) SetRun(run models.RunResponse) error {
	// ttlcache is thread-safe, no mutex needed
	// Only cache active, recent runs
	// (terminal or old runs should go to permanent cache)
	if shouldPermanentlyCache(run) {
		// Remove from session cache
		s.cache.Delete("run:" + run.ID)
		return nil
	}

	s.cache.Set("run:"+run.ID, run, 5*time.Minute)
	return nil
}

// GetRuns retrieves all cached runs from memory
func (s *SessionCache) GetRuns() ([]models.RunResponse, bool) {
	// ttlcache is thread-safe, no mutex needed
	// Check if we have a cached run list
	item := s.cache.Get("runs:all")
	if item != nil {
		if runs, ok := item.Value().([]models.RunResponse); ok {
			// Filter to only return active, recent runs
			activeRuns := make([]models.RunResponse, 0)
			for _, run := range runs {
				if !shouldPermanentlyCache(run) {
					activeRuns = append(activeRuns, run)
				}
			}
			return activeRuns, len(activeRuns) > 0
		}
	}

	// Alternatively, collect individual cached runs
	runs := make([]models.RunResponse, 0)
	items := s.cache.Items()
	for key, item := range items {
		if len(key) > 4 && key[:4] == "run:" {
			if run, ok := item.Value().(models.RunResponse); ok {
				if !shouldPermanentlyCache(run) {
					runs = append(runs, run)
				}
			}
		}
	}

	return runs, len(runs) > 0
}

// SetRuns stores multiple runs in memory
func (s *SessionCache) SetRuns(runs []models.RunResponse) error {
	// ttlcache is thread-safe, no mutex needed
	// Cache the full list for quick retrieval
	s.cache.Set("runs:all", runs, 5*time.Minute)

	// Also cache individual active, recent runs
	for _, run := range runs {
		if !shouldPermanentlyCache(run) {
			s.cache.Set("run:"+run.ID, run, 5*time.Minute)
		}
	}

	return nil
}

// InvalidateRun removes a specific run from memory cache
func (s *SessionCache) InvalidateRun(id string) error {
	// ttlcache is thread-safe, no mutex needed
	s.cache.Delete("run:" + id)
	// Also invalidate the runs list to force refresh
	s.cache.Delete("runs:all")
	return nil
}

// InvalidateActiveRuns clears all active runs from memory
func (s *SessionCache) InvalidateActiveRuns() error {
	// ttlcache is thread-safe, no mutex needed
	// Remove the cached runs list
	s.cache.Delete("runs:all")

	// Remove individual run entries
	items := s.cache.Items()
	for key := range items {
		if len(key) > 4 && key[:4] == "run:" {
			s.cache.Delete(key)
		}
	}

	return nil
}

// GetFormData retrieves cached form data (for UI state)
func (s *SessionCache) GetFormData(key string) (any, bool) {
	// ttlcache is thread-safe, no mutex needed
	item := s.cache.Get("form:" + key)
	if item == nil {
		return nil, false
	}

	return item.Value(), true
}

// SetFormData caches form data with longer TTL
func (s *SessionCache) SetFormData(key string, data any) error {
	// ttlcache is thread-safe, no mutex needed
	s.cache.Set("form:"+key, data, 30*time.Minute)
	return nil
}

// GetDashboardData retrieves cached dashboard data
func (s *SessionCache) GetDashboardData() (*DashboardData, bool) {
	// ttlcache is thread-safe, no mutex needed
	item := s.cache.Get("dashboard")
	if item == nil {
		return nil, false
	}

	data, ok := item.Value().(*DashboardData)
	return data, ok
}

// SetDashboardData caches dashboard data
func (s *SessionCache) SetDashboardData(data *DashboardData) error {
	// ttlcache is thread-safe, no mutex needed
	s.cache.Set("dashboard", data, 5*time.Minute)
	return nil
}

// Clear removes all cached items from memory
func (s *SessionCache) Clear() error {
	// ttlcache is thread-safe, no mutex needed
	s.cache.DeleteAll()
	return nil
}

// Close stops the cache's background goroutines
func (s *SessionCache) Close() error {
	s.cache.Stop()
	return nil
}
