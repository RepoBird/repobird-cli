# Fix Cache Deadlock Issue

## Problem Statement
The dashboard is experiencing deadlocks when loading data due to the complex multi-layer cache system with multiple mutexes. The hybrid cache (combining session and permanent cache) creates a complex locking hierarchy that leads to deadlocks when `SetRepositoryData` is called in a loop.

## Root Cause Analysis

### Current Architecture Issues
1. **Three-layer cache structure**:
   - SimpleCache wraps HybridCache
   - HybridCache combines SessionCache and PermanentCache  
   - Each layer has its own mutex

2. **Lock ordering problems**:
   - Session cache uses ttlcache library with internal locking
   - Permanent cache uses file I/O with mutex protection
   - Calling SetRepositoryData in a loop acquires locks in unpredictable order

3. **Complexity overhead**:
   - Duplicate data storage (memory + disk)
   - Complex TTL management
   - Difficult to debug and maintain

## Proposed Solution: Simplified Persistent Cache

### Design Principles
1. **Single source of truth**: Disk-based persistent cache only
2. **Simple locking**: One RWMutex or lock-free design
3. **OS caching**: Rely on OS file system cache for performance
4. **Atomic operations**: Ensure data integrity with atomic file writes

### Benefits
- **Eliminates deadlocks**: No complex lock hierarchies
- **Simpler code**: Easier to understand and maintain
- **Predictable performance**: Consistent behavior
- **Better reliability**: All data persisted, survives crashes
- **Reduced memory**: No duplicate data storage

## Implementation Plan

### Phase 1: Create New Simplified Cache
```go
// internal/tui/cache/simplified.go
type SimplifiedCache struct {
    baseDir   string
    memCache  map[string][]byte  // Simple read cache
    mu        sync.RWMutex        // Single mutex for memory cache
}
```

### Phase 2: Remove Complex Layers
1. Remove `HybridCache` completely
2. Remove `SessionCache` and ttlcache dependency
3. Simplify `SimpleCache` to use new implementation

### Phase 3: Optimize Performance
1. Add simple memory buffer for reads (no TTL, just invalidate on write)
2. Batch writes when possible
3. Use atomic file operations for safety

## Detailed Tasks

### Task 1: Analyze Current Usage
- [ ] Document all cache methods currently used
- [ ] Identify critical paths requiring optimization
- [ ] Measure current performance baseline

### Task 2: Implement SimplifiedCache
- [ ] Create new `simplified.go` with basic structure
- [ ] Implement core methods: Get, Set, Clear
- [ ] Add atomic file write operations
- [ ] Add simple memory read cache with RWMutex

### Task 3: Migrate SimpleCache
- [ ] Update SimpleCache to use SimplifiedCache
- [ ] Remove HybridCache references
- [ ] Update initialization code
- [ ] Ensure backward compatibility with existing cache files

### Task 4: Remove Old Code
- [ ] Delete `hybrid.go`
- [ ] Delete `session.go`
- [ ] Remove ttlcache dependency from go.mod
- [ ] Clean up unused imports

### Task 5: Update Dashboard Integration
- [ ] Remove temporary deadlock workaround in dash_data.go
- [ ] Re-enable cache operations
- [ ] Test dashboard loading with new cache

### Task 6: Testing
- [ ] Unit tests for SimplifiedCache
- [ ] Concurrent access tests
- [ ] Performance benchmarks
- [ ] Integration tests with dashboard

### Task 7: Documentation
- [ ] Update CLAUDE.md with new cache architecture
- [ ] Document cache file format
- [ ] Add troubleshooting guide

## Code Changes Required

### 1. SimplifiedCache Implementation
```go
func (c *SimplifiedCache) SetRepositoryData(repo string, runs []*RunResponse, details map[string]*RunResponse) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Serialize data
    data := RepositoryData{
        Repository: repo,
        Runs:      runs,
        Details:   details,
        CachedAt:  time.Now(),
    }
    
    // Write to disk atomically
    if err := c.writeAtomic(repo, data); err != nil {
        return err
    }
    
    // Update memory cache
    c.memCache[repo] = data
    
    return nil
}
```

### 2. Atomic File Operations
```go
func (c *SimplifiedCache) writeAtomic(key string, data interface{}) error {
    // Write to temp file first
    tempFile := filepath.Join(c.baseDir, key+".tmp")
    finalFile := filepath.Join(c.baseDir, key+".json")
    
    bytes, err := json.Marshal(data)
    if err != nil {
        return err
    }
    
    // Write to temp file
    if err := os.WriteFile(tempFile, bytes, 0644); err != nil {
        return err
    }
    
    // Atomic rename
    return os.Rename(tempFile, finalFile)
}
```

### 3. Dashboard Integration Fix
```go
// Re-enable caching in dash_data.go
repositories = d.updateRepositoryStats(repositories, allRuns)
d.cache.SetRepositoryOverview(repositories)

// Cache individual repositories without deadlock
for _, repo := range repositories {
    repoRuns := d.filterRunsByRepository(allRuns, repo.Name)
    // This will now use SimplifiedCache without deadlock
    d.cache.SetRepositoryData(repo.Name, repoRuns, nil)
}
```

## Testing Strategy

### Unit Tests
- Test concurrent reads/writes
- Test atomic file operations
- Test memory cache invalidation
- Test error handling

### Integration Tests
- Dashboard loading with real data
- Multiple concurrent TUI sessions
- Cache persistence across restarts
- Performance under load

### Performance Benchmarks
- Compare with old hybrid cache
- Measure read/write latency
- Test with large datasets (1000+ runs)
- Memory usage comparison

## Rollback Plan
If issues arise:
1. Keep old cache implementation in separate branch
2. Add feature flag to switch between implementations
3. Provide migration tool for cache format changes
4. Monitor error rates and performance metrics

## Success Criteria
- [ ] No deadlocks in dashboard loading
- [ ] Dashboard loads in < 1 second with cached data
- [ ] Memory usage reduced by 30%+
- [ ] Code complexity reduced (fewer lines, simpler logic)
- [ ] All existing tests pass
- [ ] No regression in user experience

## Timeline
- Phase 1-2: Core implementation (2 hours)
- Phase 3-4: Migration and cleanup (1 hour)  
- Phase 5-6: Testing and validation (2 hours)
- Phase 7: Documentation (30 minutes)

Total estimated time: 5.5 hours

## Notes
- The simplified approach trades a small amount of disk I/O for massive gains in simplicity and reliability
- Modern SSDs and OS file caching make disk-based caching very efficient
- This approach aligns with the "do one thing well" Unix philosophy
- Future optimizations can be added incrementally without architectural changes