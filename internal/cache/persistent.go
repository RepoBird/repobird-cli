package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// PersistentCache handles file-based caching for terminal status runs
type PersistentCache struct {
	mu       sync.RWMutex
	cacheDir string
	userID   *int // Optional user ID for user-specific caching
}

// RepositoryHistory tracks repositories used in runs
type RepositoryHistory struct {
	Repositories []string  `json:"repositories"`
	LastUsed     time.Time `json:"lastUsed"`
	Version      int       `json:"version"`
}

// CachedRun wraps a RunResponse with metadata
type CachedRun struct {
	Run      *models.RunResponse `json:"run"`
	CachedAt time.Time           `json:"cachedAt"`
	Version  int                 `json:"version"` // For future schema changes
}

const (
	cacheVersion         = 1
	repoHistoryVersion   = 1
	appName              = "repobird"
	repoHistoryFile      = "repository_history.json"
	maxRepositoryHistory = 50 // Keep last 50 repositories
)

// NewPersistentCache creates a new persistent cache instance
func NewPersistentCache() (*PersistentCache, error) {
	return NewPersistentCacheForUser(nil)
}

// NewPersistentCacheForUser creates a new persistent cache instance for a specific user
func NewPersistentCacheForUser(userID *int) (*PersistentCache, error) {
	dir, err := getCacheDirForUser(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	return &PersistentCache{
		cacheDir: dir,
		userID:   userID,
	}, nil
}

// getCacheDir returns the appropriate cache directory for the platform (backward compatibility)
func getCacheDir() (string, error) {
	return getCacheDirForUser(nil)
}

// getCacheDirForUser returns the appropriate cache directory for a specific user
func getCacheDirForUser(userID *int) (string, error) {
	// Use os.UserCacheDir for cross-platform compatibility
	baseDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	var cacheDir string
	if userID != nil {
		// Special handling for debug/test mode (negative user IDs)
		if *userID < 0 {
			cacheDir = filepath.Join(baseDir, appName, "debug", fmt.Sprintf("user-%d", *userID), "runs")
			debug.LogToFilef("DEBUG: Using debug cache directory: %s (userID=%d)\n", cacheDir, *userID)
		} else {
			// User-specific cache directory for real users
			cacheDir = filepath.Join(baseDir, appName, "users", fmt.Sprintf("user-%d", *userID), "runs")
			debug.LogToFilef("DEBUG: Using user-specific cache directory: %s (userID=%d)\n", cacheDir, *userID)
		}
	} else {
		// Fallback to shared cache directory for backward compatibility
		cacheDir = filepath.Join(baseDir, appName, "shared", "runs")
		debug.LogToFilef("DEBUG: Using shared cache directory: %s (no userID)\n", cacheDir)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return cacheDir, nil
}

// getCacheFilePath returns the file path for a specific run ID
func (pc *PersistentCache) getCacheFilePath(runID string) string {
	// Use simple naming: runID.json
	// For better organization with many files, could use subdirectories based on ID prefix
	return filepath.Join(pc.cacheDir, fmt.Sprintf("%s.json", runID))
}

// SaveRun saves a terminal status run to persistent cache
func (pc *PersistentCache) SaveRun(run *models.RunResponse) error {
	// Only cache terminal status runs
	if !isTerminalStatus(run.Status) {
		return nil
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	cached := CachedRun{
		Run:      run,
		CachedAt: time.Now(),
		Version:  cacheVersion,
	}

	filePath := pc.getCacheFilePath(run.GetIDString())
	return writeJSONAtomic(filePath, cached)
}

// LoadRun loads a cached run by ID
func (pc *PersistentCache) LoadRun(runID string) (*models.RunResponse, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	filePath := pc.getCacheFilePath(runID)

	var cached CachedRun
	if err := readJSON(filePath, &cached); err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Cache miss
		}
		// If file is corrupted, remove it
		_ = os.Remove(filePath)
		return nil, nil
	}

	// Check version for future compatibility
	if cached.Version != cacheVersion {
		// Handle version mismatch if needed in future
		os.Remove(filePath)
		return nil, nil
	}

	return cached.Run, nil
}

// LoadAllTerminalRuns loads all cached terminal runs
func (pc *PersistentCache) LoadAllTerminalRuns() (map[string]*models.RunResponse, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	runs := make(map[string]*models.RunResponse)

	// Read all .json files in cache directory
	files, err := os.ReadDir(pc.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return runs, nil // Empty cache
		}
		return nil, err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(pc.cacheDir, file.Name())
		var cached CachedRun

		if err := readJSON(filePath, &cached); err != nil {
			// Remove corrupted file
			_ = os.Remove(filePath)
			continue
		}

		// Check version
		if cached.Version != cacheVersion {
			_ = os.Remove(filePath)
			continue
		}

		// Only include if it's still a terminal status
		if cached.Run != nil && isTerminalStatus(cached.Run.Status) {
			runID := cached.Run.GetIDString()
			if runID != "" {
				runs[runID] = cached.Run
			}
		}
	}

	return runs, nil
}

// DeleteRun removes a run from the cache
func (pc *PersistentCache) DeleteRun(runID string) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	filePath := pc.getCacheFilePath(runID)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CleanOldCache removes cache files older than specified duration
func (pc *PersistentCache) CleanOldCache(maxAge time.Duration) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	files, err := os.ReadDir(pc.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cutoff := time.Now().Add(-maxAge)

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(pc.cacheDir, file.Name())
		info, err := file.Info()
		if err != nil {
			continue
		}

		// Remove files older than cutoff
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filePath)
		}
	}

	return nil
}

// writeJSONAtomic writes JSON data atomically using temp file + rename
func writeJSONAtomic(filePath string, data interface{}) error {
	// Write to temp file first
	tempPath := filePath + ".tmp"

	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty print for debugging

	if err := encoder.Encode(data); err != nil {
		_ = file.Close()
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// readJSON reads and decodes JSON from a file
func readJSON(filePath string, dst interface{}) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	return nil
}

// getRepositoryHistoryPath returns the path to repository history file
func (pc *PersistentCache) getRepositoryHistoryPath() string {
	// Repository history is stored in the parent directory of runs
	parentDir := filepath.Dir(pc.cacheDir)
	return filepath.Join(parentDir, repoHistoryFile)
}

// AddRepository adds a repository to the history, moving it to front if already exists
func (pc *PersistentCache) AddRepository(repo string) error {
	if repo == "" {
		return nil
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	history, err := pc.loadRepositoryHistory()
	if err != nil {
		// If load fails, start with empty history
		history = &RepositoryHistory{
			Repositories: []string{},
			Version:      repoHistoryVersion,
		}
	}

	// Remove repo if it already exists to avoid duplicates
	for i, existing := range history.Repositories {
		if existing == repo {
			history.Repositories = append(history.Repositories[:i], history.Repositories[i+1:]...)
			break
		}
	}

	// Add to front
	history.Repositories = append([]string{repo}, history.Repositories...)

	// Limit history size
	if len(history.Repositories) > maxRepositoryHistory {
		history.Repositories = history.Repositories[:maxRepositoryHistory]
	}

	history.LastUsed = time.Now()

	// Save back to file
	return writeJSONAtomic(pc.getRepositoryHistoryPath(), history)
}

// GetRepositoryHistory returns the repository history, most recent first
func (pc *PersistentCache) GetRepositoryHistory() ([]string, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	history, err := pc.loadRepositoryHistory()
	if err != nil {
		return []string{}, nil // Return empty list if no history found
	}

	return history.Repositories, nil
}

// GetMostRecentRepository returns the most recently used repository
func (pc *PersistentCache) GetMostRecentRepository() (string, error) {
	repos, err := pc.GetRepositoryHistory()
	if err != nil || len(repos) == 0 {
		return "", err
	}
	return repos[0], nil
}

// loadRepositoryHistory loads repository history from file
func (pc *PersistentCache) loadRepositoryHistory() (*RepositoryHistory, error) {
	filePath := pc.getRepositoryHistoryPath()

	var history RepositoryHistory
	if err := readJSON(filePath, &history); err != nil {
		if os.IsNotExist(err) {
			return &RepositoryHistory{
				Repositories: []string{},
				Version:      repoHistoryVersion,
			}, nil
		}
		return nil, err
	}

	// Handle version compatibility
	if history.Version != repoHistoryVersion {
		// For now, just reset on version mismatch
		return &RepositoryHistory{
			Repositories: []string{},
			Version:      repoHistoryVersion,
		}, nil
	}

	return &history, nil
}
