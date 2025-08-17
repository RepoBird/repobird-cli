// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/repobird/repobird-cli/internal/models"
)

// CacheData represents the structure of persisted cache data
type CacheData struct {
	Runs          []models.RunResponse `json:"runs"`
	UserInfo      *models.UserInfo     `json:"userInfo"`
	FileHashes    map[string]string    `json:"fileHashes"`
	DashboardData *DashboardData       `json:"dashboardData"`
	SavedAt       time.Time            `json:"savedAt"`
}

// SaveToDisk persists cache to disk (called on quit)
// Note: With the new hybrid cache, most data is already persisted automatically
// This method is kept for backward compatibility
func (c *SimpleCache) SaveToDisk() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// The hybrid cache already persists terminal runs and other data automatically
	// This is now a no-op for compatibility
	return nil
}

// LoadFromDisk restores cache from disk (called on start)
// Note: With the new hybrid cache, data is loaded automatically from the permanent cache
// This method is kept for backward compatibility
func (c *SimpleCache) LoadFromDisk() error {
	// The hybrid cache automatically loads persisted data from disk
	// This is now a no-op for compatibility
	return nil
}

// GetCacheFilePath returns the path where cache is stored
func GetCacheFilePath() string {
	// Respect XDG_CONFIG_HOME environment variable for testing
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = xdg.ConfigHome
	}
	return filepath.Join(configDir, "repobird", "cache.json")
}
