# Task: Refactor Global State Anti-Pattern (Bubble Tea Native Approach)

## Issue Summary
The RepoBird CLI codebase currently uses global variables and package-level initialization, violating Go best practices and creating significant technical debt. This refactoring will implement Bubble Tea's native state management pattern for the TUI, eliminating global state while keeping the solution simple and framework-aligned.

## 🎯 TL;DR - The Simple Solution

Instead of complex dependency injection:
1. **Use Bubble Tea's Model** as the state container (it's designed for this!)
2. **Add ttlcache library** for automatic TTL and cleanup
3. **Embed cache in TUI views** - no globals, no DI
4. **Keep CLI commands as-is** - they don't need changes
5. **One week of work** instead of four!

## Current Problems

### 1. Global Variables
- **`globalCache`** in `/internal/cache/cache.go:64`
  - Initialized via `init()` function at package import
  - Shared mutable state across entire application
  - No thread-safety guarantees in concurrent operations
  
- **`dashboardCache`** in `/internal/cache/dashboard_cache.go:41`
  - Duplicate cache implementation
  - Initialized separately from globalCache
  - Inconsistent with main cache patterns

### 2. Package-Level Command Variables
Found 18 instances of global command definitions:
- `/internal/commands/root.go:21` - `var rootCmd`
- `/internal/commands/run.go:24` - `var runCmd`
- `/internal/commands/status.go:27` - `var statusCmd`
- `/internal/commands/auth.go:20,26,108,142,182` - auth-related commands
- `/internal/commands/config.go:24,30,97,178` - config commands
- `/internal/commands/tui.go:13` - `var tuiCmd`
- And others in completion.go, docs.go

### 3. Direct Viper Access
- Configuration accessed directly via `viper.GetString()` throughout codebase
- No abstraction layer for configuration management
- Difficult to mock for testing

## Impact Analysis

### Testing Challenges
- Cannot isolate unit tests due to shared global state
- Race conditions in parallel test execution
- Impossible to test different cache configurations simultaneously
- Commands cannot be tested with different dependencies

### Maintenance Issues
- Hidden dependencies make code harder to understand
- Difficult to trace data flow through application
- Changes to global state affect entire application unpredictably

### Scalability Problems
- Cannot run multiple instances with different configurations
- Thread-safety concerns in concurrent operations
- Memory leaks possible from unbounded cache growth

## Why NOT Complex Dependency Injection?

After researching Bubble Tea best practices and examining the codebase:

1. **Bubble Tea has its own state management pattern** - The Model IS the state container
2. **CLI commands barely use cache** - Only bulk.go uses file hash caching
3. **TUI is the primary cache consumer** - All cache usage is in TUI views
4. **DI adds unnecessary complexity** - For a single-user CLI tool, it's overkill
5. **Testing is simpler with models** - Create a model with test data, no mocks needed

## Proposed Solution: Bubble Tea Native Approach

### Core Principle
Use Bubble Tea's Model as the single source of truth for application state, including cache. This follows the framework's intended patterns and keeps the code simple.

### Phase 1: Add TTL Cache Library

#### 1.1 Install ttlcache
```bash
go get github.com/jellydator/ttlcache/v3
go get github.com/adrg/xdg  # For cross-platform directories
```

#### 1.2 Why ttlcache?
- **Simple API** - Get/Set/Delete with automatic TTL expiration
- **Thread-safe** - Built-in mutex protection
- **Performant** - Optimized for <10,000 items
- **Well-maintained** - Active development, widely used
- **No configuration hell** - Works out of the box

```go
// internal/tui/cache/simple.go
package cache

import (
    "time"
    "github.com/jellydator/ttlcache/v3"
    "github.com/repobird/repobird-cli/internal/models"
)

// SimpleCache wraps ttlcache for RepoBird's needs
type SimpleCache struct {
    cache *ttlcache.Cache[string, any]
}

// NewSimpleCache creates a cache with sensible defaults
func NewSimpleCache() *SimpleCache {
    cache := ttlcache.New[string, any](
        ttlcache.WithTTL[string, any](5 * time.Minute),
        ttlcache.WithCapacity[string, any](10000),
    )
    
    // Start automatic cleanup
    go cache.Start()
    
    return &SimpleCache{cache: cache}
}

// GetRuns retrieves cached runs
func (c *SimpleCache) GetRuns() []models.RunResponse {
    if item := c.cache.Get("runs"); item != nil {
        if runs, ok := item.Value().([]models.RunResponse); ok {
            return runs
        }
    }
    return nil
}

// SetRuns caches runs with TTL
func (c *SimpleCache) SetRuns(runs []models.RunResponse) {
    c.cache.Set("runs", runs, ttlcache.DefaultTTL)
}

// GetUserInfo retrieves cached user info
func (c *SimpleCache) GetUserInfo() *models.UserInfo {
    if item := c.cache.Get("userInfo"); item != nil {
        if info, ok := item.Value().(*models.UserInfo); ok {
            return info
        }
    }
    return nil
}

// SetUserInfo caches user info
func (c *SimpleCache) SetUserInfo(info *models.UserInfo) {
    c.cache.Set("userInfo", info, 10*time.Minute)
}

// Clear removes all cached items
func (c *SimpleCache) Clear() {
    c.cache.DeleteAll()
}
```

### Phase 2: Update TUI Model to Include Cache

#### 2.1 Embed Cache in Bubble Tea Model
```go
// internal/tui/views/dashboard.go
package views

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/repobird/repobird-cli/internal/tui/cache"
)

// DashboardView holds all state for the dashboard
type DashboardView struct {
    // This IS our state container - no globals!
    cache     *cache.SimpleCache
    apiClient APIClient
    
    // View-specific state
    width     int
    height    int
    runs      []models.RunResponse
    loading   bool
    err       error
}

// NewDashboardView creates a dashboard with embedded cache
func NewDashboardView(client APIClient) *DashboardView {
    return &DashboardView{
        cache:     cache.NewSimpleCache(),  // Cache is part of the model!
        apiClient: client,
    }
}

// Update handles all state changes through messages
func (v *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case runsLoadedMsg:
        v.runs = msg.runs
        v.cache.SetRuns(msg.runs)  // Cache through the model
        v.loading = false
        
    case tea.WindowSizeMsg:
        v.width = msg.Width
        v.height = msg.Height
    }
    
    return v, nil
}
```

#### 2.2 Use Commands for Async Operations (Bubble Tea Way)
```go
// internal/tui/views/commands.go
package views

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/repobird/repobird-cli/internal/api"
)

// Messages for state updates
type runsLoadedMsg struct {
    runs []models.RunResponse
    err  error
}

type userInfoLoadedMsg struct {
    info *models.UserInfo
    err  error
}

// Commands for async operations
func fetchRunsCmd(client APIClient) tea.Cmd {
    return func() tea.Msg {
        runs, err := client.ListRuns(nil)
        return runsLoadedMsg{runs, err}
    }
}

func fetchUserInfoCmd(client APIClient) tea.Cmd {
    return func() tea.Msg {
        info, err := client.VerifyAuth()
        return userInfoLoadedMsg{info, err}
    }
}

// Load from cache or fetch
func loadRunsCmd(cache *cache.SimpleCache, client APIClient) tea.Cmd {
    return func() tea.Msg {
        // Try cache first
        if runs := cache.GetRuns(); runs != nil {
            return runsLoadedMsg{runs, nil}
        }
        
        // Cache miss, fetch from API
        runs, err := client.ListRuns(nil)
        if err == nil {
            cache.SetRuns(runs)
        }
        return runsLoadedMsg{runs, err}
    }
}
```

### Phase 3: Handle CLI Commands Minimally

#### 3.1 Keep CLI Commands Simple (No Major Changes Needed!)
```go
// internal/commands/bulk.go - The ONLY CLI command that uses cache
package commands

import (
    "github.com/repobird/repobird-cli/internal/cache"
)

// No changes needed for most commands!
// They already don't use global cache

var bulkCmd = &cobra.Command{
    Use: "bulk",
    RunE: func(cmd *cobra.Command, args []string) error {
        // For bulk command, just create a local cache instance
        fileHashCache := cache.NewFileHashCache()
        // Use it for duplicate detection
        // ...
    },
}

// That's it! Other commands stay as-is with their var definitions
// No complex refactoring needed for CLI commands
```

#### 3.2 Update TUI Entry Point Only
```go
// internal/commands/tui.go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/repobird/repobird-cli/internal/tui"
)

var tuiCmd = &cobra.Command{  // Keep as var - it's fine!
    Use:   "tui",
    Short: "Launch terminal user interface",
    RunE: func(cmd *cobra.Command, args []string) error {
        // TUI gets its own cache instance in the model
        app := tui.NewApp(apiClient)
        return app.Run()
    },
}

// main.go stays almost the same!
// No dependency injection framework needed
```

### Phase 4: Add Optional Persistence

#### 4.1 Cross-Platform Cache Persistence
```go
// internal/tui/cache/persistence.go
package cache

import (
    "encoding/json"
    "os"
    "path/filepath"
    "github.com/adrg/xdg"  // Cross-platform directories
)

type CacheData struct {
    Runs     []models.RunResponse `json:"runs"`
    UserInfo *models.UserInfo     `json:"userInfo"`
    SavedAt  time.Time            `json:"savedAt"`
}

// SaveToDisk persists cache to disk (called on quit)
func (c *SimpleCache) SaveToDisk() error {
    configDir, _ := xdg.ConfigHome()  // ~/.config on Linux, AppData on Windows
    cacheFile := filepath.Join(configDir, "repobird", "cache.json")
    
    // Create directory if needed
    os.MkdirAll(filepath.Dir(cacheFile), 0700)
    
    data := CacheData{
        Runs:     c.GetRuns(),
        UserInfo: c.GetUserInfo(),
        SavedAt:  time.Now(),
    }
    
    jsonData, _ := json.Marshal(data)
    return os.WriteFile(cacheFile, jsonData, 0600)
}

// LoadFromDisk restores cache from disk (called on start)
func (c *SimpleCache) LoadFromDisk() error {
    configDir, _ := xdg.ConfigHome()
    cacheFile := filepath.Join(configDir, "repobird", "cache.json")
    
    data, err := os.ReadFile(cacheFile)
    if err != nil {
        return err  // File doesn't exist, that's OK
    }
    
    var cacheData CacheData
    if err := json.Unmarshal(data, &cacheData); err != nil {
        return err
    }
    
    // Only restore if cache is less than 1 hour old
    if time.Since(cacheData.SavedAt) < time.Hour {
        if cacheData.Runs != nil {
            c.SetRuns(cacheData.Runs)
        }
        if cacheData.UserInfo != nil {
            c.SetUserInfo(cacheData.UserInfo)
        }
    }
    
    return nil
}
```

### Phase 5: Simple Testing (No Mocks Needed!)

#### 5.1 Test with Real Cache
```go
// internal/tui/views/dashboard_test.go
package views

import (
    "testing"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/stretchr/testify/assert"
    "github.com/repobird/repobird-cli/internal/tui/cache"
)

func TestDashboardView(t *testing.T) {
    // Create a real cache - no mocks!
    view := &DashboardView{
        cache:     cache.NewSimpleCache(),
        apiClient: &testClient{},  // Simple test client
    }
    
    // Test cache operations
    testRuns := []models.RunResponse{{ID: "test-1"}}
    view.cache.SetRuns(testRuns)
    
    // Verify cache works
    cached := view.cache.GetRuns()
    assert.Equal(t, testRuns, cached)
    
    // Test Bubble Tea update
    model, cmd := view.Update(runsLoadedMsg{runs: testRuns})
    updatedView := model.(*DashboardView)
    assert.Equal(t, testRuns, updatedView.runs)
    assert.Nil(t, cmd)
}

// Simple test client - not a complex mock!
type testClient struct{}

func (t *testClient) ListRuns(params *models.ListParams) ([]models.RunResponse, error) {
    return []models.RunResponse{{ID: "test-1"}}, nil
}
```

#### 5.2 Test Commands with Messages
```go
// internal/tui/views/commands_test.go
package views

import (
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestLoadRunsCommand(t *testing.T) {
    cache := cache.NewSimpleCache()
    client := &testClient{}
    
    // Test cache hit
    cache.SetRuns([]models.RunResponse{{ID: "cached"}})
    cmd := loadRunsCmd(cache, client)
    msg := cmd()  // Execute command
    
    loadedMsg := msg.(runsLoadedMsg)
    assert.NoError(t, loadedMsg.err)
    assert.Equal(t, "cached", loadedMsg.runs[0].ID)
    
    // Test cache miss
    cache.Clear()
    cmd = loadRunsCmd(cache, client)
    msg = cmd()
    
    loadedMsg = msg.(runsLoadedMsg)
    assert.NoError(t, loadedMsg.err)
    assert.Equal(t, "test-1", loadedMsg.runs[0].ID)
}
```

## Implementation Plan (Much Simpler!)

### Day 1-2: Add Cache Library
- [ ] Add ttlcache and xdg dependencies
- [ ] Create SimpleCache wrapper in internal/tui/cache
- [ ] Write basic tests for cache operations
- [ ] Add persistence methods (save/load from disk)

### Day 3-4: Update TUI Views
- [ ] Add cache field to DashboardView model
- [ ] Add cache field to other view models (List, Details, Create)
- [ ] Replace cache.GetGlobalCache() calls with model.cache
- [ ] Convert blocking operations to Bubble Tea commands

### Day 5: Migration & Cleanup
- [ ] Update bulk command to use local cache instance
- [ ] Remove global cache variables
- [ ] Remove init() functions from cache package
- [ ] Test full TUI flow with embedded cache

### Day 6-7: Testing & Polish
- [ ] Write integration tests for TUI with cache
- [ ] Test persistence across restarts
- [ ] Performance testing (should be faster!)
- [ ] Update documentation

That's it! One week instead of four!

## Testing Strategy (Simplified!)

### Unit Tests
1. Test cache operations with real ttlcache (no mocks!)
2. Test Bubble Tea models with messages
3. Test commands return correct messages
4. Test persistence save/load

### Integration Tests
1. Test full TUI flow with embedded cache
2. Test cache persistence across restarts
3. Verify CLI commands still work unchanged

### Performance Tests
1. ttlcache is already benchmarked and optimized
2. Should be faster than current global cache with locks
3. Memory usage should be lower (better GC with ttlcache)

## Rollback Plan

This approach is so simple, rollback is trivial:
1. Changes are isolated to TUI views
2. CLI commands remain unchanged
3. Can revert to global cache temporarily if needed
4. Each view can be migrated independently

## Success Metrics

### Code Quality
- [ ] Zero global cache variables
- [ ] Cache embedded in Bubble Tea models
- [ ] All TUI state flows through Update()
- [ ] CLI commands unchanged (except bulk)

### Testing
- [ ] Tests use real cache, not mocks
- [ ] Tests can run in parallel
- [ ] Model-based testing for TUI
- [ ] No test pollution

### Performance
- [ ] Faster than global cache (no lock contention)
- [ ] Lower memory usage (ttlcache GC-friendly)
- [ ] O(1) cache operations
- [ ] Instant startup (lazy loading)

### Maintainability
- [ ] Follows Bubble Tea patterns
- [ ] Simple, readable code
- [ ] Easy to understand state flow
- [ ] Less code overall

## Migration Checklist

### Pre-Migration
- [ ] Create feature branch: `refactor/tui-native-cache`
- [ ] Run existing tests as baseline
- [ ] Document which views use cache

### During Migration (1 Week!)
- [ ] Day 1-2: Add ttlcache dependency
- [ ] Day 3-4: Update TUI views
- [ ] Day 5: Clean up globals
- [ ] Day 6-7: Test and document

### Post-Migration
- [ ] Remove old cache package
- [ ] Update CLAUDE.md with new pattern
- [ ] Celebrate simpler code! 🎉

## Risk Assessment

### Minimal Risk!
- CLI commands unchanged (except bulk)
- TUI changes are isolated
- Cache library is battle-tested
- Simple rollback if needed

### Why This is Safer
- Following framework patterns (not fighting them)
- Using proven libraries (ttlcache)
- Smaller change surface area
- Easy to understand and debug

## Dependencies

### New Libraries (Minimal!)
```bash
go get github.com/jellydator/ttlcache/v3  # TTL cache
go get github.com/adrg/xdg               # Cross-platform dirs
```

### Why These Libraries?
- **ttlcache**: Simple, fast, well-maintained
- **xdg**: Handles Windows/Mac/Linux paths correctly
- Both have minimal dependencies
- Both are widely used and trusted

## Key Insights

### Why Bubble Tea Native is Better

1. **Works WITH the framework**: Bubble Tea expects state in the Model
2. **Simpler testing**: Just create models with test data
3. **Less code**: No interfaces, mocks, or DI containers
4. **Easier to understand**: State flows through Update()
5. **Better performance**: No lock contention between views

### What We Learned

- **Don't over-engineer**: This is a single-user CLI, not a web service
- **Use the right tool**: ttlcache solves our problem perfectly
- **Follow framework patterns**: Bubble Tea has opinions - follow them!
- **Isolate changes**: TUI refactoring doesn't need to touch CLI

### Future Benefits

- Easy to add Redis cache later (just swap SimpleCache)
- Can add cache metrics in one place
- Testing is straightforward
- New developers understand it immediately

## References

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Bubble Tea Best Practices](https://leg100.github.io/en/posts/building-bubbletea-programs/)
- [ttlcache Documentation](https://github.com/jellydator/ttlcache)
- [XDG Base Directory](https://github.com/adrg/xdg)
- [The Elm Architecture](https://guide.elm-lang.org/architecture/) (Bubble Tea's inspiration)

## FAQ

### Q: Why not use dependency injection?
**A**: Bubble Tea already has a pattern for state management - the Model. DI adds complexity without benefits for a TUI app.

### Q: What about the CLI commands?
**A**: They barely use cache (only bulk command). No need to refactor them.

### Q: Is ttlcache production-ready?
**A**: Yes! It's used by many production Go applications and is actively maintained.

### Q: What about testing?
**A**: Testing is actually easier! Create a model with a cache, send messages, assert on state. No mocks needed.

### Q: Will this break existing functionality?
**A**: No! CLI commands stay the same, TUI gets better. It's a win-win.

## Approval

- [ ] Technical Lead Review
- [ ] Architecture Review
- [ ] Security Review (for configuration handling)
- [ ] Documentation Review

---

**Created**: 2025-01-12
**Author**: AI Assistant
**Status**: Draft
**Target Version**: v2.0.0