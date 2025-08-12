package cache

import (
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

// CacheStrategy defines the main cache interface
type CacheStrategy interface {
	RunCache
	RepositoryCache
	UserCache
	FileCache
	Clear() error
	Close() error
}

// RunCache handles run caching
type RunCache interface {
	// Single run operations
	GetRun(id string) (*models.RunResponse, bool)
	SetRun(run models.RunResponse) error
	
	// Bulk operations
	GetRuns() ([]models.RunResponse, bool)
	SetRuns(runs []models.RunResponse) error
	
	// Invalidation
	InvalidateRun(id string) error
	InvalidateActiveRuns() error
}

// RepositoryCache handles repository data
type RepositoryCache interface {
	GetRepository(name string) (*models.Repository, bool)
	SetRepository(repo models.Repository) error
	GetRepositoryList() ([]string, bool)
	SetRepositoryList(repos []string) error
}

// UserCache handles user information
type UserCache interface {
	GetUserInfo() (*models.UserInfo, bool)
	SetUserInfo(info *models.UserInfo) error
	InvalidateUserInfo() error
}

// FileCache handles file hashes
type FileCache interface {
	GetFileHash(path string) (string, bool)
	SetFileHash(path string, hash string) error
	GetAllFileHashes() map[string]string
}

// StorageLayer defines where data is stored
type StorageLayer int

const (
	MemoryLayer StorageLayer = iota
	DiskLayer
	HybridLayer // Both memory and disk
)

// CachePolicy defines caching behavior
type CachePolicy struct {
	Layer      StorageLayer
	TTL        time.Duration
	Persistent bool
}

// DataPolicies defines policies for different data types
var DataPolicies = map[string]CachePolicy{
	"terminal_runs": {
		Layer:      DiskLayer,
		TTL:        0, // Never expires
		Persistent: true,
	},
	"active_runs": {
		Layer:      MemoryLayer,
		TTL:        5 * time.Minute,
		Persistent: false,
	},
	"repositories": {
		Layer:      DiskLayer,
		TTL:        0,
		Persistent: true,
	},
	"user_info": {
		Layer:      DiskLayer,
		TTL:        0, // Store permanently on disk
		Persistent: true,
	},
	"file_hashes": {
		Layer:      DiskLayer,
		TTL:        0,
		Persistent: true,
	},
	"form_data": {
		Layer:      MemoryLayer,
		TTL:        30 * time.Minute,
		Persistent: false,
	},
}