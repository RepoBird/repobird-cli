# Task: Fix Cache Architecture Chaos

## Problem Summary
The current cache implementation mixes different data types with inconsistent TTL strategies, no clear boundaries, and inefficient storage patterns. After our recent refactoring to embed caches in views (eliminating globals), we now need to implement proper cache layering with appropriate storage strategies for different data types.

## Current Issues

### 1. Mixed Responsibilities
- SimpleCache treats all data the same (5-minute TTL)
- No distinction between immutable and changing data
- Terminal runs (DONE/FAILED) expire unnecessarily
- Active runs cached the same as completed ones

### 2. Inefficient Storage
- Everything in memory with TTL expiration
- Completed runs refetched from API repeatedly
- No permanent storage for immutable data
- Poor offline support

### 3. Missing Abstractions
- No interface for different cache strategies
- Direct cache implementation coupling
- No clear invalidation policies
- Mixed concerns in single cache instance

## Proposed Solution: Layered Cache Architecture

### Architecture Overview
```
┌─────────────────────────────────────────────────────────────┐
│                   HybridCache (Facade)                      │
├─────────────────────────────────────────────────────────────┤
│  Permanent Layer (Per User) │  Session Layer (Per User)    │
│  ~/.config/repobird/        │  (TTL Memory)                │
│  users/{user-id}/           │                              │
│  ├── Terminal Runs          │  ├── Active Runs (5min)     │
│  ├── Repositories           │  └── Form Data (30min)      │
│  ├── User Info              │                              │
│  ├── File Hashes            │                              │
│  └── Run History            │                              │
└─────────────────────────────────────────────────────────────┘
```

### Storage Strategy by Data Type

| Data Type | Storage | TTL | Invalidation |
|-----------|---------|-----|--------------|
| Terminal Runs (DONE/FAILED) | Disk | Never | Manual only |
| Active Runs (RUNNING/PENDING) | Memory | 5 min | On status change |
| Repository Metadata | Disk | Never | On repo update |
| User Info | Disk | Never | On user switch |
| Run Lists (Terminal) | Disk | Never | Manual only |
| Run Lists (Active) | Memory | 5 min | On refresh |
| File Hashes | Disk | Never | On file change |
| Form Data | Memory | 30 min | On submit |

## Implementation Plan

### Phase 1: Define Cache Interfaces (Day 1)

#### 1.1 Create Cache Strategy Interfaces
```go
// internal/tui/cache/interfaces.go
package cache

import (
    "context"
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
```

#### 1.2 Define Storage Layers
```go
// internal/tui/cache/layers.go
package cache

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
        Layer:      MemoryLayer,
        TTL:        10 * time.Minute,
        Persistent: false,
    },
    "run_lists": {
        Layer:      HybridLayer,
        TTL:        30 * time.Minute,
        Persistent: true,
    },
}
```

### Phase 2: Implement Storage Backends (Day 2-3)

#### 2.1 Permanent Disk Cache

**Directory Structure:**
```
~/.config/repobird/
├── users/
│   ├── user-7a8b9c1d/    # User 1 (hashed ID)
│   │   ├── runs/
│   │   │   ├── run-123.json
│   │   │   └── run-456.json
│   │   ├── repositories/
│   │   │   └── repos.json
│   │   ├── user-info.json
│   │   └── file-hashes.json
│   └── user-2e3f4a5b/    # User 2 (different user)
│       ├── runs/
│       ├── repositories/
│       └── user-info.json
```

```go
// internal/tui/cache/permanent.go
package cache

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "github.com/adrg/xdg"
)

type PermanentCache struct {
    baseDir string
    userID  string
    mu      sync.RWMutex
}

func NewPermanentCache(userID string) (*PermanentCache, error) {
    configDir := os.Getenv("XDG_CONFIG_HOME")
    if configDir == "" {
        configDir = xdg.ConfigHome
    }
    
    // User-specific cache directory
    userHash := hashUserID(userID)
    baseDir := filepath.Join(configDir, "repobird", "users", userHash)
    if err := os.MkdirAll(baseDir, 0700); err != nil {
        return nil, err
    }
    
    return &PermanentCache{
        baseDir: baseDir,
        userID:  userID,
    }, nil
}

// hashUserID creates a stable hash for directory naming
func hashUserID(userID string) string {
    h := sha256.Sum256([]byte(userID))
    return fmt.Sprintf("user-%x", h[:8])
}

func (p *PermanentCache) GetRun(id string) (*models.RunResponse, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    path := filepath.Join(p.baseDir, "runs", id+".json")
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, false
    }
    
    var run models.RunResponse
    if err := json.Unmarshal(data, &run); err != nil {
        return nil, false
    }
    
    // Only return if run is in terminal state
    if !isTerminalState(run.Status) {
        return nil, false
    }
    
    return &run, true
}

func (p *PermanentCache) SetRun(run models.RunResponse) error {
    // Only cache terminal states
    if !isTerminalState(run.Status) {
        return nil
    }
    
    p.mu.Lock()
    defer p.mu.Unlock()
    
    runDir := filepath.Join(p.baseDir, "runs")
    if err := os.MkdirAll(runDir, 0700); err != nil {
        return err
    }
    
    path := filepath.Join(runDir, run.ID+".json")
    data, err := json.MarshalIndent(run, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(path, data, 0600)
}

func isTerminalState(status models.RunStatus) bool {
    return status == models.StatusDone || 
           status == models.StatusFailed || 
           status == models.StatusCancelled
}

// GetUserInfo retrieves permanently cached user info
func (p *PermanentCache) GetUserInfo() (*models.UserInfo, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
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
    p.mu.Lock()
    defer p.mu.Unlock()
    
    path := filepath.Join(p.baseDir, "user-info.json")
    data, err := json.MarshalIndent(info, "", "  ")
    if err != nil {
        return err
    }
    
    return os.WriteFile(path, data, 0600)
}
```

#### 2.2 Session Memory Cache
```go
// internal/tui/cache/session.go
package cache

import (
    "time"
    "github.com/jellydator/ttlcache/v3"
)

type SessionCache struct {
    cache *ttlcache.Cache[string, any]
}

func NewSessionCache() *SessionCache {
    cache := ttlcache.New[string, any](
        ttlcache.WithCapacity[string, any](1000),
    )
    
    go cache.Start()
    
    return &SessionCache{
        cache: cache,
    }
}

func (s *SessionCache) GetRun(id string) (*models.RunResponse, bool) {
    item := s.cache.Get("run:" + id)
    if item == nil {
        return nil, false
    }
    
    run, ok := item.Value().(models.RunResponse)
    if !ok {
        return nil, false
    }
    
    // Only return active runs from session cache
    if isTerminalState(run.Status) {
        s.cache.Delete("run:" + id)
        return nil, false
    }
    
    return &run, true
}

func (s *SessionCache) SetRun(run models.RunResponse) error {
    // Only cache active states
    if isTerminalState(run.Status) {
        // Remove from session cache if terminal
        s.cache.Delete("run:" + run.ID)
        return nil
    }
    
    s.cache.Set("run:"+run.ID, run, 5*time.Minute)
    return nil
}
```

### Phase 3: Implement Hybrid Cache (Day 4)

#### 3.1 Hybrid Cache Facade
```go
// internal/tui/cache/hybrid.go
package cache

import (
    "sync"
    "github.com/repobird/repobird-cli/internal/models"
)

type HybridCache struct {
    permanent *PermanentCache
    session   *SessionCache
    userID    string
    mu        sync.RWMutex
}

func NewHybridCache(userID string) (*HybridCache, error) {
    if userID == "" {
        return nil, fmt.Errorf("userID is required for cache initialization")
    }
    
    permanent, err := NewPermanentCache(userID)
    if err != nil {
        return nil, err
    }
    
    session := NewSessionCache()
    
    return &HybridCache{
        permanent: permanent,
        session:   session,
        userID:    userID,
    }, nil
}

// GetRun checks both caches intelligently
func (h *HybridCache) GetRun(id string) (*models.RunResponse, bool) {
    // Check permanent cache first for terminal runs
    if run, found := h.permanent.GetRun(id); found {
        return run, true
    }
    
    // Check session cache for active runs
    if run, found := h.session.GetRun(id); found {
        return run, true
    }
    
    return nil, false
}

// SetRun routes to appropriate cache based on status
func (h *HybridCache) SetRun(run models.RunResponse) error {
    if isTerminalState(run.Status) {
        // Move to permanent storage
        h.session.InvalidateRun(run.ID)
        return h.permanent.SetRun(run)
    }
    
    // Keep in session cache
    return h.session.SetRun(run)
}

// GetRuns returns merged results from both caches
func (h *HybridCache) GetRuns() ([]models.RunResponse, bool) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    runs := []models.RunResponse{}
    runMap := make(map[string]models.RunResponse)
    
    // Get permanent runs
    if permanentRuns, found := h.permanent.GetAllRuns(); found {
        for _, run := range permanentRuns {
            runMap[run.ID] = run
        }
    }
    
    // Get session runs (may override with fresher data)
    if sessionRuns, found := h.session.GetRuns(); found {
        for _, run := range sessionRuns {
            runMap[run.ID] = run
        }
    }
    
    // Convert map to slice
    for _, run := range runMap {
        runs = append(runs, run)
    }
    
    return runs, len(runs) > 0
}

// InvalidateActiveRuns only clears non-terminal runs
func (h *HybridCache) InvalidateActiveRuns() error {
    return h.session.InvalidateActiveRuns()
}
```

### Phase 4: Migrate Existing Code (Day 5)

#### 4.1 Update SimpleCache to Use HybridCache
```go
// internal/tui/cache/simple.go
package cache

// SimpleCache now wraps HybridCache for backward compatibility
type SimpleCache struct {
    hybrid *HybridCache
}

func NewSimpleCache() *SimpleCache {
    // Get current user ID from auth context or config
    userID := getCurrentUserID()
    if userID == "" {
        // Fallback to anonymous cache if no user
        userID = "anonymous"
    }
    
    hybrid, err := NewHybridCache(userID)
    if err != nil {
        // Fallback to memory-only if disk cache fails
        hybrid = &HybridCache{
            session: NewSessionCache(),
            userID:  userID,
        }
    }
    
    return &SimpleCache{
        hybrid: hybrid,
    }
}

// getCurrentUserID retrieves the current user ID from context
func getCurrentUserID() string {
    // This would get the user ID from:
    // 1. Current auth context
    // 2. Config file
    // 3. API verification
    // For now, returns empty string if not authenticated
    return services.GetCurrentUserID()
}

// Delegate all operations to hybrid cache
func (c *SimpleCache) GetRun(id string) *models.RunResponse {
    run, _ := c.hybrid.GetRun(id)
    return run
}

func (c *SimpleCache) SetRun(run models.RunResponse) {
    _ = c.hybrid.SetRun(run)
}

// ... other delegated methods
```

#### 4.2 Update Views to Use New Cache
```go
// No changes needed! Views continue using SimpleCache
// which now has improved behavior under the hood
```

### Phase 5: Add Cache Management (Day 6)

#### 5.1 Cache Statistics
```go
// internal/tui/cache/stats.go
package cache

type CacheStats struct {
    PermanentRuns   int
    ActiveRuns      int
    Repositories    int
    DiskUsageBytes  int64
    MemoryUsageBytes int64
    HitRate         float64
}

func (h *HybridCache) GetStats() CacheStats {
    // Collect statistics from both layers
}
```

#### 5.2 Cache Maintenance
```go
// internal/tui/cache/maintenance.go
package cache

// CleanupOldRuns removes runs older than retention period
func (p *PermanentCache) CleanupOldRuns(retention time.Duration) error {
    // Remove old terminal runs from disk
}

// Compact reorganizes disk storage
func (p *PermanentCache) Compact() error {
    // Reorganize and optimize disk storage
}
```

## Migration Path

1. **Phase 1**: Implement new cache layers alongside existing SimpleCache
2. **Phase 2**: Update SimpleCache to delegate to HybridCache
3. **Phase 3**: Test with existing views (no changes needed)
4. **Phase 4**: Add monitoring and metrics
5. **Phase 5**: Remove old cache code

## Success Metrics

### Performance
- [ ] Terminal runs load in <10ms (from disk)
- [ ] 90% reduction in API calls for completed runs
- [ ] Memory usage reduced by 50% for large run lists
- [ ] Offline mode works for viewing completed runs

### Code Quality
- [ ] Clear separation between cache layers
- [ ] Each cache type has single responsibility
- [ ] All cache operations through interfaces
- [ ] 100% backward compatibility with existing views

### User Experience
- [ ] Instant load for completed runs
- [ ] Smooth offline experience
- [ ] Faster dashboard load times
- [ ] Reduced network usage

## Testing Strategy

### Unit Tests
```go
func TestPermanentCache_OnlyStoresTerminalRuns(t *testing.T) {
    cache := NewPermanentCache("test-user-123")
    
    // Should not store active run
    activeRun := models.RunResponse{
        ID: "test-1",
        Status: models.StatusRunning,
    }
    err := cache.SetRun(activeRun)
    assert.NoError(t, err)
    
    _, found := cache.GetRun("test-1")
    assert.False(t, found)
    
    // Should store terminal run
    terminalRun := models.RunResponse{
        ID: "test-2",
        Status: models.StatusDone,
    }
    err = cache.SetRun(terminalRun)
    assert.NoError(t, err)
    
    cached, found := cache.GetRun("test-2")
    assert.True(t, found)
    assert.Equal(t, terminalRun, *cached)
}
```

### Integration Tests
```go
func TestHybridCache_UserSeparation(t *testing.T) {
    // Create caches for different users
    cache1 := NewHybridCache("user-1")
    cache2 := NewHybridCache("user-2")
    
    // Add run to user 1's cache
    run1 := models.RunResponse{
        ID: "run-1",
        Status: models.StatusDone,
    }
    cache1.SetRun(run1)
    
    // User 2 should not see user 1's run
    _, found := cache2.GetRun("run-1")
    assert.False(t, found)
    
    // User 1 should see their own run
    cached, found := cache1.GetRun("run-1")
    assert.True(t, found)
    assert.Equal(t, run1, *cached)
}

func TestHybridCache_StatusTransition(t *testing.T) {
    cache := NewHybridCache("test-user")
    
    // Start with active run
    run := models.RunResponse{
        ID: "test-1",
        Status: models.StatusRunning,
    }
    cache.SetRun(run)
    
    // Should be in session cache
    cached, _ := cache.GetRun("test-1")
    assert.Equal(t, models.StatusRunning, cached.Status)
    
    // Update to terminal status
    run.Status = models.StatusDone
    cache.SetRun(run)
    
    // Should move to permanent cache
    cached, _ = cache.GetRun("test-1")
    assert.Equal(t, models.StatusDone, cached.Status)
    
    // Should persist across cache recreation
    cache2 := NewHybridCache()
    cached, found := cache2.GetRun("test-1")
    assert.True(t, found)
    assert.Equal(t, models.StatusDone, cached.Status)
}
```

## Implementation Checklist

### Week 1
- [ ] Create cache interfaces
- [ ] Implement PermanentCache for disk storage
- [ ] Implement SessionCache for TTL memory
- [ ] Create HybridCache facade
- [ ] Add cache routing logic

### Week 2
- [ ] Update SimpleCache to use HybridCache
- [ ] Add cache statistics
- [ ] Implement cache maintenance
- [ ] Write comprehensive tests
- [ ] Performance benchmarks

### Week 3
- [ ] Deploy and monitor
- [ ] Tune cache parameters
- [ ] Add debug commands
- [ ] Documentation update
- [ ] Remove old code

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Disk I/O performance | Use async writes, batch operations |
| Cache inconsistency | Version cache entries, add validation |
| Storage growth | Implement retention policies, compression |
| Migration failures | Keep backward compatibility layer |

## Notes

- Maintains embedded cache pattern (no globals)
- Backward compatible with existing views
- Progressive enhancement approach
- Can disable disk cache via environment variable
- Test with `XDG_CONFIG_HOME` for isolation

## References

- Original issue: `/tasks/code-smells.md` - Section 3: Cache Architecture Chaos
- Current implementation: `/internal/tui/cache/simple.go`
- TTLCache docs: https://github.com/jellydator/ttlcache
- XDG spec: https://specifications.freedesktop.org/basedir-spec/

---

**Created**: 2025-08-12
**Author**: AI Assistant
**Status**: Ready for Implementation
**Priority**: P1 (High)
**Estimated Effort**: 2 weeks