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
   - No clear lock acquisition order between layers

3. **Unclear data flow**:
   - Ambiguous routing between session and permanent cache
   - Complex state transitions between cache layers
   - No clear single source of truth for different data types

## Proposed Solution: Simplified Hybrid Cache with Clear Lock Ordering

### Design Principles
1. **Keep hybrid benefits**: Memory for hot data, disk for persistence
2. **Single-writer pattern**: Only one goroutine writes to cache at a time
3. **Clear data ownership**: Define which cache owns what data
4. **Lock-free reads**: Implement copy-on-read for hot paths
5. **Simple lock hierarchy**: Never hold upper lock while calling lower layer

### Benefits
- **Eliminates deadlocks**: Clear lock ordering and boundaries
- **Preserves performance**: Memory cache for UI responsiveness
- **Maintains persistence**: Disk cache survives restarts  
- **Simpler mental model**: Clear data flow and ownership
- **Backward compatible**: No migration needed

## Implementation Plan

### Phase 1: Fix Lock Ordering and Boundaries
1. **Define clear lock hierarchy**: SimpleCache → HybridCache → (SessionCache | PermanentCache)
2. **Never hold parent lock when calling child**: Release SimpleCache lock before calling HybridCache
3. **Use single-writer pattern**: Queue writes through channels to avoid concurrent modifications
4. **Copy data for reads**: Return copies to avoid holding locks during processing

### Phase 2: Simplify Data Flow
1. **Clear ownership rules**:
   - Terminal runs (DONE/FAILED/CANCELLED) → PermanentCache only
   - Active runs (RUNNING/PENDING < 2hrs) → SessionCache only  
   - Old/stuck runs (> 2hrs) → PermanentCache only
2. **Single routing decision**: Make cache destination decision once at write time
3. **No cross-cache moves**: Once in permanent, stays permanent

### Phase 3: Implement Lock-Free Patterns
1. **Read-copy-update for hot paths**: Copy data structures for reads
2. **Atomic operations**: Use atomic.Value for frequently read data
3. **Channel-based writes**: Queue write operations to serialize access

## Detailed Tasks

### Task 1: Analyze Current Deadlock Pattern
- [x] Identify lock acquisition order in SetRepositoryData loop
- [x] Map out all mutex interactions between cache layers
- [x] Document which operations hold multiple locks
- [ ] Create lock dependency graph

### Task 2: Fix SimpleCache Lock Ordering
- [ ] Remove SimpleCache mutex from methods that call HybridCache
- [ ] Implement copy-on-read for GetRuns/GetRun methods
- [ ] Add write queue for SetRepositoryData to serialize writes
- [ ] Ensure no lock is held during HybridCache calls

### Task 3: Fix HybridCache Lock Management
- [ ] Remove HybridCache.mu from GetRuns (not needed with proper child locking)
- [ ] Implement single-decision routing (no state changes after initial write)
- [ ] Add clear separation between read and write paths
- [ ] Ensure permanent and session caches are never locked simultaneously

### Task 4: Simplify SessionCache Locking
- [ ] Remove redundant mutex operations (ttlcache has internal locking)
- [ ] Use ttlcache's thread-safe methods directly
- [ ] Implement batch operations to reduce lock acquisitions
- [ ] Add read-only fast path for common queries

### Task 5: Fix PermanentCache File I/O
- [ ] Move file I/O outside of lock critical section
- [ ] Implement write-ahead logging for atomic operations
- [ ] Add memory buffer for recent reads (no lock needed)
- [ ] Use atomic file operations (write temp + rename)

### Task 6: Update Dashboard Integration
- [ ] Batch repository data updates instead of loop
- [ ] Use goroutines with result channels for parallel caching
- [ ] Implement progress callback to avoid blocking UI
- [ ] Add timeout/context for cache operations

### Task 7: Testing & Validation
- [ ] Unit test for concurrent SetRepositoryData calls
- [ ] Stress test with 100+ parallel cache operations
- [ ] Benchmark before/after lock contention
- [ ] Integration test with real dashboard data
- [ ] Test with Go race detector enabled

### Task 8: Documentation
- [ ] Document new lock hierarchy in code comments
- [ ] Update CLAUDE.md with cache patterns
- [ ] Add troubleshooting guide for cache issues
- [ ] Create architecture diagram showing data flow

## Code Changes Required

### 1. SimpleCache - Remove Lock During HybridCache Calls
```go
// internal/tui/cache/simple.go
func (c *SimpleCache) SetRepositoryData(repoName string, runs []*models.RunResponse, details map[string]*models.RunResponse) {
    // Prepare data without lock
    key := fmt.Sprintf("repo:%s", repoName)
    data := &RepositoryData{
        Name:        repoName,
        Runs:        runs,
        Details:     details,
        LastUpdated: time.Now(),
    }
    
    // Call hybrid cache without holding SimpleCache lock
    // HybridCache will handle its own locking
    c.hybrid.SetFormData(key, data)
}

func (c *SimpleCache) GetRuns() []models.RunResponse {
    // Get runs without lock - HybridCache handles thread safety
    runs, _ := c.hybrid.GetRuns()
    
    // Return copy to avoid mutations
    result := make([]models.RunResponse, len(runs))
    copy(result, runs)
    return result
}
```

### 2. HybridCache - Single Decision Routing
```go
// internal/tui/cache/hybrid.go
func (h *HybridCache) SetRun(run models.RunResponse) error {
    // Make routing decision once - no state changes
    if shouldPermanentlyCache(run) {
        // Direct to permanent, no session interaction
        if h.permanent != nil {
            return h.permanent.SetRun(run)
        }
    } else {
        // Direct to session, no permanent interaction
        return h.session.SetRun(run)
    }
    return nil
}

func (h *HybridCache) GetRuns() ([]models.RunResponse, bool) {
    // No mutex needed - child caches handle their own locking
    runMap := make(map[string]models.RunResponse)
    
    // Get from caches in parallel
    var wg sync.WaitGroup
    var permMu, sessMu sync.Mutex
    
    // Permanent cache goroutine
    wg.Add(1)
    go func() {
        defer wg.Done()
        if h.permanent != nil {
            if runs, found := h.permanent.GetAllRuns(); found {
                permMu.Lock()
                for _, run := range runs {
                    runMap[run.ID] = run
                }
                permMu.Unlock()
            }
        }
    }()
    
    // Session cache goroutine  
    wg.Add(1)
    go func() {
        defer wg.Done()
        if runs, found := h.session.GetRuns(); found {
            sessMu.Lock()
            for _, run := range runs {
                runMap[run.ID] = run
            }
            sessMu.Unlock()
        }
    }()
    
    wg.Wait()
    
    // Convert to slice
    runs := make([]models.RunResponse, 0, len(runMap))
    for _, run := range runMap {
        runs = append(runs, run)
    }
    
    return runs, len(runs) > 0
}
```

### 3. Dashboard - Batch Cache Updates
```go
// internal/tui/views/dash_data.go
func (d *DashboardView) cacheRepositoryData(repositories []models.Repository, allRuns []*models.RunResponse) {
    // Use worker pool to avoid lock contention
    type cacheJob struct {
        repo    models.Repository
        runs    []*models.RunResponse
        details map[string]*models.RunResponse
    }
    
    jobs := make(chan cacheJob, len(repositories))
    
    // Queue all jobs
    for _, repo := range repositories {
        repoRuns := d.filterRunsByRepository(allRuns, repo.Name)
        repoDetails := make(map[string]*models.RunResponse)
        for _, run := range repoRuns {
            if detail, exists := d.detailsCache[run.GetIDString()]; exists {
                repoDetails[run.GetIDString()] = detail
            }
        }
        
        jobs <- cacheJob{
            repo:    repo,
            runs:    repoRuns,
            details: repoDetails,
        }
    }
    close(jobs)
    
    // Process with limited concurrency
    var wg sync.WaitGroup
    for i := 0; i < 3; i++ { // Max 3 concurrent cache writes
        wg.Add(1)
        go func() {
            defer wg.Done()
            for job := range jobs {
                d.cache.SetRepositoryData(job.repo.Name, job.runs, job.details)
            }
        }()
    }
    
    wg.Wait()
}
```

### 4. PermanentCache - Lock-Free File Operations
```go
// internal/tui/cache/permanent.go
func (p *PermanentCache) SetRun(run models.RunResponse) error {
    if !shouldPermanentlyCache(run) {
        return nil
    }
    
    // Prepare data outside lock
    runDir := filepath.Join(p.baseDir, "runs")
    tempPath := filepath.Join(runDir, run.ID+".tmp")
    finalPath := filepath.Join(runDir, run.ID+".json")
    
    data, err := json.MarshalIndent(run, "", "  ")
    if err != nil {
        return err
    }
    
    // Ensure directory exists (idempotent)
    os.MkdirAll(runDir, 0700)
    
    // Write to temp file (no lock needed)
    if err := os.WriteFile(tempPath, data, 0600); err != nil {
        return err
    }
    
    // Atomic rename (no lock needed)
    return os.Rename(tempPath, finalPath)
}

func (p *PermanentCache) GetRun(id string) (*models.RunResponse, bool) {
    // Read without lock - file system handles concurrency
    path := filepath.Join(p.baseDir, "runs", id+".json")
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, false
    }
    
    var run models.RunResponse
    if err := json.Unmarshal(data, &run); err != nil {
        return nil, false
    }
    
    if !shouldPermanentlyCache(run) {
        return nil, false
    }
    
    return &run, true
}
```

## Testing Strategy

### Concurrent Access Tests
```go
func TestNoCacheDeadlock(t *testing.T) {
    cache := NewSimpleCache()
    defer cache.Stop()
    
    // Simulate dashboard loading pattern
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(idx int) {
            defer wg.Done()
            repo := fmt.Sprintf("repo-%d", idx)
            runs := generateTestRuns(10)
            
            // This previously caused deadlock
            cache.SetRepositoryData(repo, runs, nil)
        }(i)
    }
    
    done := make(chan bool)
    go func() {
        wg.Wait()
        done <- true
    }()
    
    select {
    case <-done:
        // Success - no deadlock
    case <-time.After(5 * time.Second):
        t.Fatal("Deadlock detected - operations did not complete")
    }
}
```

### Race Condition Detection
```bash
# Run with race detector
go test -race ./internal/tui/cache/...

# Stress test with high concurrency
go test -run=TestConcurrent -count=100 -parallel=10
```

## Rollback Plan
1. Keep current implementation tagged before changes
2. Add feature flag `REPOBIRD_USE_OLD_CACHE=true` to revert
3. Monitor for any performance regressions
4. Have hotfix ready if issues found in production

## Success Criteria
- [ ] No deadlocks when calling SetRepositoryData in loop
- [ ] Dashboard loads without hanging
- [ ] Race detector finds no issues
- [ ] Concurrent test with 100+ goroutines passes
- [ ] No performance regression (< 1s dashboard load)
- [ ] Code is simpler and clearer

## Timeline
- Phase 1: Fix lock ordering (2 hours)
- Phase 2: Simplify data flow (1 hour)
- Phase 3: Lock-free patterns (2 hours)
- Testing & validation (1 hour)
- Documentation (30 minutes)

Total estimated time: 6.5 hours

## Notes
- The key insight is that we don't need complex multi-layer locking
- Clear ownership boundaries eliminate most synchronization needs
- File systems already handle concurrent access well
- Go's race detector is invaluable for validating the fixes
- This maintains backward compatibility while fixing the core issue