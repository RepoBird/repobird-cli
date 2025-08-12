# Task: Refactor Global State Anti-Pattern

## Issue Summary
The RepoBird CLI codebase currently uses global variables and package-level initialization, violating Go best practices and creating significant technical debt. This refactoring will implement proper dependency injection to eliminate global state.

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

## Proposed Solution

### Phase 1: Create Dependency Container Infrastructure

#### 1.1 Define Core Interfaces
```go
// internal/container/interfaces.go
package container

import (
    "context"
    "time"
    "github.com/repobird/repobird-cli/internal/models"
)

// CacheService defines the cache operations
type CacheService interface {
    // Run operations
    GetRuns(ctx context.Context, userID int) ([]models.RunResponse, error)
    SetRuns(ctx context.Context, userID int, runs []models.RunResponse, ttl time.Duration) error
    InvalidateRuns(ctx context.Context, userID int) error
    
    // Run details operations
    GetRunDetails(ctx context.Context, runID string) (*models.RunResponse, error)
    SetRunDetails(ctx context.Context, runID string, details *models.RunResponse, ttl time.Duration) error
    
    // User info operations
    GetUserInfo(ctx context.Context) (*models.UserInfo, error)
    SetUserInfo(ctx context.Context, info *models.UserInfo, ttl time.Duration) error
    
    // Form data operations
    GetFormData(ctx context.Context) (*FormData, error)
    SetFormData(ctx context.Context, data *FormData) error
}

// ConfigService defines configuration operations
type ConfigService interface {
    GetAPIKey() string
    GetAPIURL() string
    GetEnvironment() string
    SetAPIKey(key string) error
    SetAPIURL(url string) error
    Validate() error
}

// APIClient defines the API operations
type APIClient interface {
    CreateRun(ctx context.Context, req *models.CreateRunRequest) (*models.RunResponse, error)
    GetRun(ctx context.Context, runID string) (*models.RunResponse, error)
    ListRuns(ctx context.Context, params *models.ListParams) ([]models.RunResponse, error)
    VerifyAuth(ctx context.Context) (*models.UserInfo, error)
}
```

#### 1.2 Create Container Structure
```go
// internal/container/container.go
package container

import (
    "sync"
    "github.com/repobird/repobird-cli/internal/api"
)

// Container holds all application dependencies
type Container struct {
    mu        sync.RWMutex
    cache     CacheService
    config    ConfigService
    apiClient APIClient
    
    // Lazy initialization flags
    cacheInit     sync.Once
    configInit    sync.Once
    apiClientInit sync.Once
}

// New creates a new dependency container
func New() *Container {
    return &Container{}
}

// Cache returns the cache service, initializing if needed
func (c *Container) Cache() CacheService {
    c.cacheInit.Do(func() {
        if c.cache == nil {
            c.cache = NewDefaultCache()
        }
    })
    return c.cache
}

// Config returns the config service, initializing if needed
func (c *Container) Config() ConfigService {
    c.configInit.Do(func() {
        if c.config == nil {
            c.config = NewDefaultConfig()
        }
    })
    return c.config
}

// APIClient returns the API client, initializing if needed
func (c *Container) APIClient() APIClient {
    c.apiClientInit.Do(func() {
        if c.apiClient == nil {
            c.apiClient = api.NewClient(c.Config())
        }
    })
    return c.apiClient
}

// WithCache sets a custom cache implementation
func (c *Container) WithCache(cache CacheService) *Container {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.cache = cache
    return c
}

// WithConfig sets a custom config implementation
func (c *Container) WithConfig(config ConfigService) *Container {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.config = config
    return c
}

// WithAPIClient sets a custom API client implementation
func (c *Container) WithAPIClient(client APIClient) *Container {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.apiClient = client
    return c
}
```

### Phase 2: Refactor Cache Implementation

#### 2.1 Convert GlobalCache to Service
```go
// internal/cache/service.go
package cache

import (
    "context"
    "sync"
    "time"
    "github.com/repobird/repobird-cli/internal/models"
)

// Service implements the CacheService interface
type Service struct {
    mu              sync.RWMutex
    runs            []models.RunResponse
    runsCachedAt    time.Time
    runsTTL         time.Duration
    details         map[string]*models.RunResponse
    detailsAt       map[string]time.Time
    terminalDetails map[string]*models.RunResponse
    userInfo        *models.UserInfo
    userInfoTime    time.Time
    formData        *FormData
    persistentCache *PersistentCache
    fileHashCache   *FileHashCache
}

// NewService creates a new cache service
func NewService(userID *int) (*Service, error) {
    pc, err := NewPersistentCacheForUser(userID)
    if err != nil {
        // Log error but continue with memory-only cache
        pc = nil
    }
    
    s := &Service{
        details:         make(map[string]*models.RunResponse),
        detailsAt:       make(map[string]time.Time),
        terminalDetails: make(map[string]*models.RunResponse),
        persistentCache: pc,
        fileHashCache:   NewFileHashCacheForUser(userID),
        runsTTL:         5 * time.Minute, // Default TTL
    }
    
    // Load persisted terminal runs
    if pc != nil {
        if terminalRuns, err := pc.LoadAllTerminalRuns(); err == nil {
            s.terminalDetails = terminalRuns
        }
    }
    
    return s, nil
}

// GetRuns implements CacheService
func (s *Service) GetRuns(ctx context.Context, userID int) ([]models.RunResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    // Check if cache is still valid
    if time.Since(s.runsCachedAt) > s.runsTTL {
        return nil, ErrCacheExpired
    }
    
    return s.runs, nil
}

// SetRuns implements CacheService
func (s *Service) SetRuns(ctx context.Context, userID int, runs []models.RunResponse, ttl time.Duration) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.runs = runs
    s.runsCachedAt = time.Now()
    if ttl > 0 {
        s.runsTTL = ttl
    }
    
    return nil
}

// Additional methods implementation...
```

#### 2.2 Create Migration Helper
```go
// internal/cache/migration.go
package cache

import (
    "github.com/repobird/repobird-cli/internal/container"
)

// GetGlobalCache returns the global cache wrapped in the service interface
// DEPRECATED: This is a migration helper and will be removed
func GetGlobalCache() container.CacheService {
    if globalCache == nil {
        initializeCache()
    }
    return &legacyWrapper{cache: globalCache}
}

// legacyWrapper wraps the old GlobalCache to implement CacheService
type legacyWrapper struct {
    cache *GlobalCache
}

// Implement CacheService methods wrapping GlobalCache methods
func (w *legacyWrapper) GetRuns(ctx context.Context, userID int) ([]models.RunResponse, error) {
    return w.cache.GetRuns()
}

// Continue implementing wrapper methods...
```

### Phase 3: Refactor Commands

#### 3.1 Convert Commands to Functions
```go
// internal/commands/root.go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/repobird/repobird-cli/internal/container"
)

// NewRootCommand creates the root command with dependencies
func NewRootCommand(c *container.Container) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "repobird",
        Short: "RepoBird CLI - AI-powered code generation",
        Long:  `RepoBird CLI allows you to submit and track AI-powered code generation tasks.`,
        PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
            // Initialize configuration
            return c.Config().Validate()
        },
    }
    
    // Add subcommands with dependencies
    cmd.AddCommand(
        NewRunCommand(c),
        NewStatusCommand(c),
        NewAuthCommand(c),
        NewConfigCommand(c),
        NewTUICommand(c),
        NewVersionCommand(),
        NewCompletionCommand(),
        NewDocsCommand(),
    )
    
    // Add global flags
    cmd.PersistentFlags().Bool("debug", false, "Enable debug output")
    cmd.PersistentFlags().String("api-url", "", "Override API URL")
    
    return cmd
}

// NewRunCommand creates the run command with dependencies
func NewRunCommand(c *container.Container) *cobra.Command {
    var follow bool
    var skipConfirmation bool
    
    cmd := &cobra.Command{
        Use:   "run [task-file]",
        Short: "Submit a new run from a task JSON file",
        Args:  cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            // Use injected dependencies
            client := c.APIClient()
            cache := c.Cache()
            config := c.Config()
            
            // Implementation using dependencies...
            return runTask(cmd.Context(), client, cache, config, args[0], follow, skipConfirmation)
        },
    }
    
    cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow run status until completion")
    cmd.Flags().BoolVar(&skipConfirmation, "yes", false, "Skip confirmation prompt")
    
    return cmd
}

// Continue converting other commands...
```

#### 3.2 Update Main Entry Point
```go
// cmd/repobird/main.go
package main

import (
    "fmt"
    "os"
    
    "github.com/repobird/repobird-cli/internal/commands"
    "github.com/repobird/repobird-cli/internal/container"
)

func main() {
    // Create dependency container
    c := container.New()
    
    // Create root command with dependencies
    rootCmd := commands.NewRootCommand(c)
    
    // Execute command
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### Phase 4: Update TUI Views

#### 4.1 Refactor View Constructors
```go
// internal/tui/views/dashboard.go
package views

import (
    "github.com/repobird/repobird-cli/internal/container"
)

// DashboardView represents the dashboard view
type DashboardView struct {
    container *container.Container
    // other fields...
}

// NewDashboardView creates a new dashboard view with dependencies
func NewDashboardView(c *container.Container) *DashboardView {
    return &DashboardView{
        container: c,
        // initialize other fields...
    }
}

// Methods use injected dependencies
func (v *DashboardView) loadData() error {
    cache := v.container.Cache()
    client := v.container.APIClient()
    
    // Use cache and client...
    return nil
}
```

### Phase 5: Testing Infrastructure

#### 5.1 Create Mock Implementations
```go
// internal/container/mocks/cache_mock.go
package mocks

import (
    "context"
    "sync"
    "time"
    "github.com/repobird/repobird-cli/internal/models"
)

// MockCache is a mock implementation of CacheService for testing
type MockCache struct {
    mu          sync.RWMutex
    runs        []models.RunResponse
    runDetails  map[string]*models.RunResponse
    userInfo    *models.UserInfo
    
    // Track method calls for assertions
    getCalls    int
    setCalls    int
    invalidated bool
}

// NewMockCache creates a new mock cache
func NewMockCache() *MockCache {
    return &MockCache{
        runDetails: make(map[string]*models.RunResponse),
    }
}

// GetRuns implements CacheService
func (m *MockCache) GetRuns(ctx context.Context, userID int) ([]models.RunResponse, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    m.getCalls++
    return m.runs, nil
}

// SetRuns implements CacheService
func (m *MockCache) SetRuns(ctx context.Context, userID int, runs []models.RunResponse, ttl time.Duration) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.setCalls++
    m.runs = runs
    return nil
}

// Helper methods for test assertions
func (m *MockCache) GetCallCount() int {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.getCalls
}

func (m *MockCache) WasInvalidated() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.invalidated
}
```

#### 5.2 Update Tests
```go
// internal/commands/run_test.go
package commands

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/repobird/repobird-cli/internal/container"
    "github.com/repobird/repobird-cli/internal/container/mocks"
)

func TestRunCommand(t *testing.T) {
    // Create container with mocks
    c := container.New()
    mockCache := mocks.NewMockCache()
    mockConfig := mocks.NewMockConfig()
    mockClient := mocks.NewMockAPIClient()
    
    c.WithCache(mockCache).
      WithConfig(mockConfig).
      WithAPIClient(mockClient)
    
    // Create command with mocked dependencies
    cmd := NewRunCommand(c)
    
    // Test command execution
    err := cmd.Execute()
    assert.NoError(t, err)
    
    // Verify interactions
    assert.Equal(t, 1, mockCache.GetCallCount())
    assert.True(t, mockClient.CreateRunCalled())
}
```

## Implementation Plan

### Week 1: Foundation
- [ ] Day 1-2: Create container package with interfaces
- [ ] Day 3-4: Implement basic container with lazy initialization
- [ ] Day 5: Create mock implementations for testing

### Week 2: Cache Refactoring
- [ ] Day 1-2: Convert GlobalCache to Service implementation
- [ ] Day 3: Create legacy wrapper for backward compatibility
- [ ] Day 4: Update cache tests with mocks
- [ ] Day 5: Verify all cache operations work correctly

### Week 3: Command Refactoring
- [ ] Day 1: Convert root and version commands
- [ ] Day 2: Convert run and status commands
- [ ] Day 3: Convert auth and config commands
- [ ] Day 4: Convert remaining commands
- [ ] Day 5: Update main.go and integration tests

### Week 4: TUI and Cleanup
- [ ] Day 1-2: Update TUI views to use container
- [ ] Day 3: Remove global variables
- [ ] Day 4: Remove legacy wrappers
- [ ] Day 5: Final testing and documentation

## Testing Strategy

### Unit Tests
1. Test each service implementation in isolation
2. Use mocks for all dependencies
3. Verify thread-safety with concurrent tests
4. Test error conditions and edge cases

### Integration Tests
1. Test command execution with real services
2. Verify cache persistence works correctly
3. Test configuration loading and validation
4. Ensure backward compatibility during migration

### Performance Tests
1. Benchmark cache operations before/after refactoring
2. Test memory usage with large datasets
3. Verify no performance regression

## Rollback Plan

If issues arise during deployment:
1. Legacy wrappers allow gradual migration
2. Feature flags can toggle between old/new implementation
3. Each phase can be deployed independently
4. Git tags at each phase for easy rollback

## Success Metrics

### Code Quality
- [ ] Zero global variables (except in main.go)
- [ ] All dependencies injected explicitly
- [ ] 100% of commands use dependency injection
- [ ] All cache operations go through interfaces

### Testing
- [ ] Test coverage increases to 80%+
- [ ] All tests can run in parallel
- [ ] Mock implementations for all services
- [ ] No test pollution between runs

### Performance
- [ ] No performance regression
- [ ] Memory usage remains stable
- [ ] Cache operations maintain O(1) complexity
- [ ] Startup time unchanged

### Maintainability
- [ ] Clear separation of concerns
- [ ] Easy to add new dependencies
- [ ] Simple to test new features
- [ ] Reduced coupling between components

## Migration Checklist

### Pre-Migration
- [ ] Create feature branch: `refactor/global-state`
- [ ] Document current behavior
- [ ] Create comprehensive test suite
- [ ] Set up performance benchmarks

### During Migration
- [ ] Implement in small, reviewable PRs
- [ ] Maintain backward compatibility
- [ ] Run tests after each change
- [ ] Update documentation as needed

### Post-Migration
- [ ] Remove all legacy code
- [ ] Update developer documentation
- [ ] Train team on new patterns
- [ ] Monitor for issues in production

## Risk Assessment

### High Risk
- Breaking existing CLI commands
- Data loss from cache changes
- Performance degradation

### Mitigation
- Extensive testing at each phase
- Legacy wrappers for compatibility
- Gradual rollout with monitoring
- Easy rollback capability

## Dependencies

### Required Before Starting
- Approval from team lead
- Complete test coverage of current behavior
- Performance benchmarks established

### External Dependencies
- No new external libraries required
- Uses standard Go patterns
- Compatible with existing tooling

## Notes

1. This refactoring follows Go best practices and industry standards
2. The container pattern is widely used in Go applications
3. Dependency injection improves testability significantly
4. This change enables future improvements like:
   - Multiple cache backends
   - Configuration hot-reloading
   - Plugin architecture
   - Better monitoring/observability

## References

- [Go Best Practices - Dependency Injection](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber's Dig - Dependency Injection Framework](https://github.com/uber-go/dig)
- [Google's Wire - Compile-time DI](https://github.com/google/wire)
- [Clean Architecture in Go](https://github.com/bxcodec/go-clean-arch)

## Questions to Address

1. Should we use a DI framework (Dig/Wire) or manual injection?
   - Recommendation: Start with manual injection for simplicity
   
2. How to handle configuration changes at runtime?
   - Recommendation: Immutable config, restart for changes
   
3. Should cache be persistent across restarts?
   - Recommendation: Yes, maintain current persistent cache behavior

4. How to handle backward compatibility?
   - Recommendation: Legacy wrappers during migration period

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