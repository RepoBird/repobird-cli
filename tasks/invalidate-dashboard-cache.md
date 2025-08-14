# Task: Invalidate Dashboard Cache After Bulk Submission

## Objective
Ensure the dashboard refreshes its data when returning from the bulk results view, so users see their newly created runs immediately.

## Current Behavior
- Dashboard uses hybrid cache system (memory + disk)
- Cache has TTL for active runs (5 minutes)
- Terminal runs are persisted to disk
- Dashboard may show stale data after bulk submission

## Implementation Plan

### 1. Identify Cache Invalidation Points
- **Primary**: When navigating to dashboard from BulkResultsView
- **Secondary**: After successful bulk submission (in BulkView)
- **Method**: Clear or mark dashboard cache as stale

### 2. Cache Invalidation Strategies

#### Option A: Clear Entire Dashboard Cache
- Simple and guaranteed to work
- Forces fresh API call on dashboard return
- May lose legitimate cached data

#### Option B: Clear Only Runs Cache
- More targeted approach
- Preserves repository list cache
- Requires identifying specific cache keys

#### Option C: Mark Cache as Stale
- Add timestamp/version to cache entries
- Dashboard checks if cache needs refresh
- More complex but preserves data integrity

### 3. Implementation Locations

#### In BulkResultsView
- When user presses 'q' or selects [DASH] button
- Before navigation message is sent
- Clear relevant cache entries

#### In App Router (app.go)
- When handling NavigateToDashboardMsg
- Check source view type
- If from BulkResults, invalidate cache

#### In Dashboard View
- Add refresh logic in Init()
- Check for "needs_refresh" flag
- Force data reload if flag is set

### 4. Code References

#### Cache Methods (internal/tui/cache/simple.go)
```go
func (c *SimpleCache) ClearRuns(repository string)
func (c *SimpleCache) InvalidateCache()
func (c *SimpleCache) SetNeedsRefresh(key string, value bool)
```

#### Navigation Points
- `bulk_results.go:233-235` - Dashboard navigation from 'q' key
- `bulk_results.go:301-303` - Dashboard navigation from [DASH] button
- `app.go:196-224` - NavigateToDashboardMsg handler

#### Dashboard Data Loading
- `dash_data.go` - loadDashboardData() method
- Uses cache.GetRuns() and cache.SetRuns()

### 5. Recommended Approach

**Use Option B: Clear Only Runs Cache**

1. Add method to SimpleCache:
   ```go
   func (c *SimpleCache) InvalidateRunsCache() {
       // Clear all run-related cache entries
       c.ClearAllRuns()
   }
   ```

2. Call before navigation in BulkResultsView:
   ```go
   case key.Matches(msg, v.keys.Dashboard):
       // Invalidate dashboard cache for fresh data
       v.cache.InvalidateRunsCache()
       return v, func() tea.Msg {
           return messages.NavigateToDashboardMsg{}
       }
   ```

3. Alternatively, set a refresh flag:
   ```go
   v.cache.SetNavigationContext("dashboard_needs_refresh", true)
   ```

4. Check flag in dashboard Init():
   ```go
   if needsRefresh := d.cache.GetNavigationContext("dashboard_needs_refresh"); needsRefresh != nil {
       d.forceRefresh = true
       d.cache.SetNavigationContext("dashboard_needs_refresh", nil)
   }
   ```

### 6. Testing Plan

1. Submit bulk runs successfully
2. Navigate to dashboard from results
3. Verify new runs appear immediately
4. Check that API call is made (not using stale cache)
5. Verify repository list is still cached (if unchanged)

### 7. Edge Cases

- Handle partial submission failures
- Multiple bulk submissions in sequence
- Navigation via back button vs direct navigation
- Cache invalidation during active polling

## Decision: Implement Option B with Targeted Cache Clearing

This provides the best balance of:
- Immediate data freshness
- Minimal performance impact
- Simple implementation
- Preserves unrelated cached data