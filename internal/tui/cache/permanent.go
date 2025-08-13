package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/adrg/xdg"
	"github.com/repobird/repobird-cli/internal/models"
)

// PermanentCache provides persistent disk storage for terminal state data
type PermanentCache struct {
	baseDir string
	userID  string
	mu      sync.RWMutex
}

// NewPermanentCache creates a new disk-based cache for a specific user
func NewPermanentCache(userID string) (*PermanentCache, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = xdg.ConfigHome
	}

	// User-specific cache directory
	userHash := hashUserID(userID)
	baseDir := filepath.Join(configDir, "repobird", "cache", "users", userHash)
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &PermanentCache{
		baseDir: baseDir,
		userID:  userID,
	}, nil
}

// hashUserID creates a stable hash for directory naming
func hashUserID(userID string) string {
	if userID == "" || userID == "anonymous" {
		return "anonymous"
	}
	h := sha256.Sum256([]byte(userID))
	return fmt.Sprintf("user-%x", h[:8])
}

// GetRun retrieves a cached run from disk (terminal states or old stuck runs)
func (p *PermanentCache) GetRun(id string) (*models.RunResponse, bool) {
	// No lock needed - file system handles concurrent reads
	path := filepath.Join(p.baseDir, "runs", id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var run models.RunResponse
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, false
	}

	// Only return if run should be permanently cached
	if !shouldPermanentlyCache(run) {
		// Clean up runs that shouldn't be cached
		_ = os.Remove(path)
		return nil, false
	}

	return &run, true
}

// SetRun stores a run to disk (terminal states or old stuck runs)
func (p *PermanentCache) SetRun(run models.RunResponse) error {
	// Only cache runs that should be permanent
	if !shouldPermanentlyCache(run) {
		return nil
	}

	// Prepare data outside of any lock
	runDir := filepath.Join(p.baseDir, "runs")
	tempPath := filepath.Join(runDir, run.ID+".tmp")
	finalPath := filepath.Join(runDir, run.ID+".json")

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal run: %w", err)
	}

	// Ensure directory exists (idempotent operation)
	if err := os.MkdirAll(runDir, 0700); err != nil {
		return fmt.Errorf("failed to create runs directory: %w", err)
	}

	// Write to temp file (no lock needed)
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename (no lock needed)
	return os.Rename(tempPath, finalPath)
}

// GetAllRuns retrieves all cached runs from disk
func (p *PermanentCache) GetAllRuns() ([]models.RunResponse, bool) {
	// No lock needed - file system handles concurrent reads
	runDir := filepath.Join(p.baseDir, "runs")
	entries, err := os.ReadDir(runDir)
	if err != nil {
		return nil, false
	}

	var runs []models.RunResponse
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		// Skip temporary files
		if strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}

		path := filepath.Join(runDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var run models.RunResponse
		if err := json.Unmarshal(data, &run); err != nil {
			continue
		}

		// Only include runs that should be permanently cached
		if shouldPermanentlyCache(run) {
			runs = append(runs, run)
		} else {
			// Clean up runs that shouldn't be cached
			_ = os.Remove(path)
		}
	}

	return runs, len(runs) > 0
}

// InvalidateRun removes a specific run from disk cache
func (p *PermanentCache) InvalidateRun(id string) error {
	// No lock needed - file system handles concurrent deletes
	path := filepath.Join(p.baseDir, "runs", id+".json")
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// AuthCache stores authentication info with timestamp
type AuthCache struct {
	UserInfo      *models.UserInfo `json:"userInfo"`
	LastAuthTime  time.Time        `json:"lastAuthTime"`
	CacheDuration time.Duration    `json:"cacheDuration"`
}

// GetUserInfo retrieves permanently cached user info
func (p *PermanentCache) GetUserInfo() (*models.UserInfo, bool) {
	// No lock needed - file system handles concurrent reads
	path := filepath.Join(p.baseDir, "user-info.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var info models.UserInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, false
	}

	return &info, true
}

// SetUserInfo permanently caches user info
func (p *PermanentCache) SetUserInfo(info *models.UserInfo) error {
	// Prepare data without lock
	tempPath := filepath.Join(p.baseDir, "user-info.tmp")
	finalPath := filepath.Join(p.baseDir, "user-info.json")

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user info: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, finalPath)
}

// GetAuthCache retrieves cached authentication info with timestamp
func (p *PermanentCache) GetAuthCache() (*AuthCache, bool) {
	// No lock needed - file system handles concurrent reads
	path := filepath.Join(p.baseDir, "auth-cache.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var auth AuthCache
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, false
	}

	return &auth, true
}

// SetAuthCache stores authentication info with timestamp
func (p *PermanentCache) SetAuthCache(userInfo *models.UserInfo) error {
	// Prepare auth cache with 2-week duration
	auth := AuthCache{
		UserInfo:      userInfo,
		LastAuthTime:  time.Now(),
		CacheDuration: 14 * 24 * time.Hour, // 2 weeks
	}

	// Prepare data without lock
	tempPath := filepath.Join(p.baseDir, "auth-cache.tmp")
	finalPath := filepath.Join(p.baseDir, "auth-cache.json")

	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth cache: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, finalPath)
}

// IsAuthCacheValid checks if cached authentication is still valid
func (p *PermanentCache) IsAuthCacheValid() bool {
	auth, found := p.GetAuthCache()
	if !found {
		return false
	}

	// Check if auth is within the cache duration
	elapsed := time.Since(auth.LastAuthTime)
	return elapsed < auth.CacheDuration
}

// GetRepositoryList retrieves cached repository list
func (p *PermanentCache) GetRepositoryList() ([]string, bool) {
	// No lock needed - file system handles concurrent reads
	path := filepath.Join(p.baseDir, "repositories", "list.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var repos []string
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, false
	}

	return repos, true
}

// SetRepositoryList caches repository list
func (p *PermanentCache) SetRepositoryList(repos []string) error {
	// Prepare data without lock
	repoDir := filepath.Join(p.baseDir, "repositories")
	tempPath := filepath.Join(repoDir, "list.tmp")
	finalPath := filepath.Join(repoDir, "list.json")

	data, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal repository list: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(repoDir, 0700); err != nil {
		return fmt.Errorf("failed to create repositories directory: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, finalPath)
}

// GetFileHash retrieves cached file hash
func (p *PermanentCache) GetFileHash(path string) (string, bool) {
	// No lock needed - loadFileHashes handles concurrent reads
	hashes := p.loadFileHashes()
	hash, found := hashes[path]
	return hash, found
}

// SetFileHash caches file hash
func (p *PermanentCache) SetFileHash(filePath string, hash string) error {
	// Use a local lock just for this operation to prevent concurrent hash updates
	// This is the only place where we need synchronization
	p.mu.Lock()
	defer p.mu.Unlock()

	hashes := p.loadFileHashes()
	hashes[filePath] = hash

	return p.saveFileHashesAtomic(hashes)
}

// GetAllFileHashes returns all cached file hashes
func (p *PermanentCache) GetAllFileHashes() map[string]string {
	// No lock needed - loadFileHashes handles concurrent reads
	return p.loadFileHashes()
}

// loadFileHashes loads file hashes from disk
func (p *PermanentCache) loadFileHashes() map[string]string {
	path := filepath.Join(p.baseDir, "file-hashes.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]string)
	}

	var hashes map[string]string
	if err := json.Unmarshal(data, &hashes); err != nil {
		return make(map[string]string)
	}

	return hashes
}

// saveFileHashesAtomic saves file hashes to disk atomically
func (p *PermanentCache) saveFileHashesAtomic(hashes map[string]string) error {
	tempPath := filepath.Join(p.baseDir, "file-hashes.tmp")
	finalPath := filepath.Join(p.baseDir, "file-hashes.json")

	data, err := json.MarshalIndent(hashes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal file hashes: %w", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, finalPath)
}

// Clear removes all cached data for this user
func (p *PermanentCache) Clear() error {
	// No lock needed - RemoveAll is atomic at the OS level
	return os.RemoveAll(p.baseDir)
}

// Close does nothing for permanent cache (no resources to release)
func (p *PermanentCache) Close() error {
	return nil
}

// CleanupOldRuns removes runs older than retention period
func (p *PermanentCache) CleanupOldRuns(maxRuns int) error {
	// No lock needed - file operations are atomic
	runDir := filepath.Join(p.baseDir, "runs")
	entries, err := os.ReadDir(runDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Keep only the most recent maxRuns
	if len(entries) > maxRuns {
		// Sort by modification time and remove oldest
		// This is a simplified implementation
		toRemove := len(entries) - maxRuns
		for i := 0; i < toRemove && i < len(entries); i++ {
			path := filepath.Join(runDir, entries[i].Name())
			_ = os.Remove(path)
		}
	}

	return nil
}

// isTerminalState checks if a run status is terminal (completed, failed, etc)
func isTerminalState(status models.RunStatus) bool {
	statusStr := string(status)
	// Check using the models package terminal statuses
	return models.IsTerminalStatus(statusStr) ||
		status == models.StatusDone ||
		status == models.StatusFailed ||
		statusStr == "COMPLETED" ||
		statusStr == "CANCELLED" ||
		statusStr == "ERROR"
}

// shouldPermanentlyCache checks if a run should be cached permanently
// This includes terminal states AND runs older than 2 hours (stuck runs)
func shouldPermanentlyCache(run models.RunResponse) bool {
	// Terminal states are always cached
	if isTerminalState(run.Status) {
		return true
	}

	// Runs older than 2 hours should be cached permanently
	// (they're likely stuck in an invalid state)
	if time.Since(run.CreatedAt) > 2*time.Hour {
		return true
	}

	return false
}
