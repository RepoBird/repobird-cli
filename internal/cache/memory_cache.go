// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/domain"
)

// MemoryCache implements domain.CacheService using in-memory storage
type MemoryCache struct {
	mu              sync.RWMutex
	runs            map[string]*domain.Run
	runList         []*domain.Run
	runListCachedAt time.Time
	cacheTTL        time.Duration
}

// NewMemoryCache creates a new memory-based cache service
func NewMemoryCache(cacheTTL time.Duration) domain.CacheService {
	if cacheTTL == 0 {
		cacheTTL = 30 * time.Second
	}
	return &MemoryCache{
		runs:     make(map[string]*domain.Run),
		cacheTTL: cacheTTL,
	}
}

// GetRun retrieves a cached run by ID
func (c *MemoryCache) GetRun(id string) (*domain.Run, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	run, found := c.runs[id]
	return run, found
}

// SetRun caches a run
func (c *MemoryCache) SetRun(id string, run *domain.Run) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.runs[id] = run
}

// GetRunList retrieves the cached run list
func (c *MemoryCache) GetRunList() ([]*domain.Run, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if list cache is still valid
	if time.Since(c.runListCachedAt) > c.cacheTTL {
		return nil, false
	}

	if c.runList == nil {
		return nil, false
	}

	// Return a copy to prevent mutations
	runsCopy := make([]*domain.Run, len(c.runList))
	copy(runsCopy, c.runList)
	return runsCopy, true
}

// SetRunList caches the run list
func (c *MemoryCache) SetRunList(runs []*domain.Run) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Make a copy to prevent external mutations
	c.runList = make([]*domain.Run, len(runs))
	copy(c.runList, runs)
	c.runListCachedAt = time.Now()

	// Also update individual run cache
	for _, run := range runs {
		c.runs[run.ID] = run
	}
}

// InvalidateRun invalidates a specific run or the list cache
func (c *MemoryCache) InvalidateRun(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if id == "list" {
		c.runList = nil
		c.runListCachedAt = time.Time{}
	} else {
		delete(c.runs, id)
	}
}

// Clear clears all cached data
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.runs = make(map[string]*domain.Run)
	c.runList = nil
	c.runListCachedAt = time.Time{}
}
