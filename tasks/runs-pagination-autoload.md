# Runs Pagination Implementation Plan

## Problem Statement

Users with 100+ runs currently hit a limit where the dashboard only shows the latest 100 runs (sorted by createdAt desc). When they scroll to the 100th run and press down/j, the navigation **wraps around to the first run** instead of loading more data, even though the API supports pagination.

**Current wrap-around behavior:**
- User scrolls down to run #100 (last visible run)
- Pressing `j` or `down` wraps back to run #1
- User cannot access runs #101-250 despite API having `totalPages: 3`

## Solution Overview

Replace automatic scrolling with a **manual load button approach**:
- Show a special "Load More" row at the bottom of the runs list when more pages are available
- User can navigate to this row and press `ENTER` to load the next 100 runs  
- Cache all loaded pages persistently in `~/.cache/repobird/users/{userId}/runs/`
- Show loading status in the TUI status line during fetch
- Append new runs seamlessly to the existing list
- **Prevent wrap-around** when more data is available

## API Integration

The `/api/v1/runs` endpoint already supports:
- `page` parameter (default: 1)
- `limit` parameter (default: 10, max: 100) 
- `sortBy=createdAt&sortOrder=desc` for consistent ordering
- Returns metadata with `currentPage`, `total`, `totalPages`

Example requests:
```
GET /api/v1/runs?page=1&limit=100&sortBy=createdAt&sortOrder=desc
GET /api/v1/runs?page=2&limit=100&sortBy=createdAt&sortOrder=desc
```

## Cache Architecture

### Directory Structure
```
~/.cache/repobird/
├── users/
│   └── {userId}/
│       └── runs/
│           ├── metadata.json     # pagination metadata, timestamps
│           ├── page_1.json      # first 100 runs
│           ├── page_2.json      # next 100 runs  
│           └── page_N.json      # additional pages
```

### Cache Metadata Format
```json
{
  "userId": "user_abc123",
  "lastUpdated": "2024-01-20T10:00:00Z",
  "totalRuns": 250,
  "totalPages": 3,
  "currentLoadedPages": [1, 2],
  "hasMorePages": true,
  "cacheVersion": "1.0",
  "cacheTTL": 300
}
```

### Cache Strategy
- **TTL**: 5-10 minutes (configurable)
- **Invalidation**: Time-based, action-based (new runs created), manual refresh
- **Corruption handling**: Checksum validation, fallback to API rebuild
- **Cleanup**: Remove cache files older than 7 days

## TUI Implementation

### Load More Button Logic  
1. **Display Logic**: When `hasMore = true` (calculated from `currentPage < totalPages`), show a special "Load More" row at the bottom of runs list
2. **Visual Design**: The row should be clearly styled as an action button:
   ```
   ┌─ Normal runs ─────────────┐
   │ Run #100: Fix auth bug    │
   │ [ENTER] Load next 100 ... │ ← Special row  
   └───────────────────────────┘
   ```
3. **Navigation**: User can navigate to this row using normal j/k keys
4. **Activation**: Pressing `ENTER` on this row triggers the load action
5. **Prevent Wrap**: When at the last actual run, `j`/`down` should go to Load More row (if exists) instead of wrapping to first run

### Modified Navigation Logic
The existing navigation in `handleMillerColumnsNavigation()` needs modification:

**Current problematic behavior** (lines 1072-1077):
```go
case 1: // Runs column  
    if d.selectedRunIdx < len(d.filteredRuns)-1 {
        d.selectedRunIdx++
    } else if len(d.filteredRuns) > 0 {
        // Wrap to first item ← PROBLEM!
        d.selectedRunIdx = 0  
    }
```

**New behavior**:
```go  
case 1: // Runs column
    if d.selectedRunIdx < len(d.filteredRuns)-1 {
        d.selectedRunIdx++
    } else if len(d.filteredRuns) > 0 {
        // Check if we should show load more button
        if d.hasMorePages() {
            // Move to "Load More" row (virtual row at index len(filteredRuns))
            d.selectedRunIdx = len(d.filteredRuns) // Points to load more row
        } else {
            // No more pages, wrap to first item
            d.selectedRunIdx = 0
        }
    }
```

### Bubble Tea Messages
```go
type LoadMoreRunsMsg struct{}

type RunsLoadingMsg struct{ 
    Page int 
}

type RunsLoadedMsg struct{
    Page       int
    Runs       []Run  
    HasMore    bool
    TotalCount int
}

type RunsLoadErrorMsg struct{
    Page  int
    Error error
}
```

### Status Line Updates
- Normal: `"Dashboard | 150 of 250 runs loaded"`
- Loading: `"Dashboard | 150 of 250 runs loaded (loading page 2...)"`  
- Loaded: `"Dashboard | 250 of 250 runs loaded"`
- Error: `"Dashboard | 150 of 250 runs loaded (load failed, press 'r' to retry)"`

### Data Flow
1. **Initial Load**: Check cache validity → Load from cache OR fetch page 1 from API
2. **Load More Selection**: User navigates to Load More row → Presses ENTER
3. **Background Load**: API request → Cache response → Update TUI model
4. **Seamless Append**: Add new runs to existing list → Update Load More button state

## Implementation Phases

### Phase 1: Cache Infrastructure
- **File**: `internal/cache/runs_cache.go`
- Cache manager with CRUD operations
- Directory creation and permission handling  
- TTL and invalidation logic
- Unit tests

### Phase 2: API Pagination Enhancement
- **File**: `internal/api/pagination.go`
- Pagination state management
- Enhanced API client methods with page parameters
- Retry logic with exponential backoff
- Integration with existing error handling

### Phase 3: TUI Integration
- **File**: `internal/tui/views/dashboard.go` (modify)
- Modified navigation logic to prevent wrap-around when more pages available
- Load More row rendering and selection handling  
- ENTER key handling for Load More row activation
- Background loading commands
- Status line integration with total counts
- Message handling in Update() method

### Phase 4: Testing & Polish
- Integration tests with mock API
- Edge case testing (network failures, cache corruption)
- Performance testing with large datasets
- Documentation updates

## Key Implementation Files

### New Files
- `internal/cache/runs_cache.go` - Cache management
- `internal/cache/cache_test.go` - Cache tests
- `internal/api/pagination.go` - Pagination logic
- `docs/runs-pagination.md` - Feature documentation

### Modified Files  
- `internal/tui/views/dashboard.go` - Scroll detection, autoload
- `internal/api/client.go` - Pagination support
- `internal/models/run.go` - Enhanced run models if needed

## Edge Cases & Error Handling

### Edge Cases
- User with exactly 100 runs (no Load More button, allow normal wrap-around)
- Network offline during load (show error in status, keep Load More button) 
- API returns different total count between pages (handle data consistency)
- Multiple rapid ENTER presses on Load More (prevent duplicate requests)
- Cache directory permission issues (fallback to memory)
- User presses up/k from Load More row (should go to last actual run)

### Error Handling
- Show load errors briefly in status line, keep Load More button available
- Retry logic with exponential backoff for failed loads
- Cache corruption recovery (rebuild from API)
- Memory management for large datasets (consider virtualization after 1000+ runs)
- Prevent duplicate load requests with loading state tracking

## User Experience Goals

- **Intuitive**: Load More button is clearly discoverable and actionable
- **Informative**: Status line shows "X of Y total runs loaded" for context
- **Responsive**: No blocking during background operations
- **Reliable**: Graceful handling of network/cache issues
- **Performant**: Efficient memory usage and API calls
- **Predictable**: Navigation behavior is consistent and doesn't surprise users

## Performance Considerations

- **On-Demand Loading**: Only fetch when user explicitly requests via Load More button
- **Memory Management**: Don't keep excessive runs in memory (consider pagination for 1000+ runs)
- **Request Deduplication**: Prevent multiple simultaneous load requests
- **Cache Efficiency**: Minimize disk I/O operations with batched reads/writes
- **Rendering Optimization**: Efficiently update TUI without full re-renders

## Configuration Options

```yaml
# ~/.repobird/config.yaml
cache:
  runs:
    ttl: 300                    # Cache TTL in seconds (5 minutes)
    max_pages_memory: 10        # Max pages to keep in memory
    cleanup_days: 7             # Remove cache files after N days

pagination:
  page_size: 100               # Runs per page (max 100)
  load_more_style: "button"    # Style for Load More row
  prevent_duplicate_loads: true # Prevent multiple simultaneous loads
```

## Success Metrics

- Users can access all their runs regardless of count (100+ runs no longer hit wall)
- Load More button is intuitive and discoverable
- No unexpected wrap-around behavior that confuses users
- Cache reduces API calls and improves responsiveness  
- Status line provides clear "X of Y loaded" context
- No performance degradation with large run counts (1000+ runs)

## Key Differences from Auto-scroll Approach

| Aspect | Auto-scroll (Original) | Load More Button (New) |
|--------|----------------------|----------------------|
| **User Control** | Automatic, invisible | Manual, explicit |
| **Discovery** | Hidden behavior | Visible button |
| **Performance** | Aggressive pre-loading | On-demand loading |
| **UX Predictability** | Surprising scrolling | Clear user action |
| **Error Recovery** | Difficult to retry | Easy to retry with button |
| **Wrap-around Issue** | Still problematic | Completely solved |

## Future Enhancements

- Virtualized scrolling for very large datasets (1000+ runs)
- Keyboard shortcut for "Load All Remaining" (Ctrl+Shift+L)
- Search/filter integration with pagination
- Bulk operations across paginated results
- Export functionality for all runs across pages
- Progress bar for large bulk loads