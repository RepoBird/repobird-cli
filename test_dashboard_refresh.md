# Dashboard Auto-Refresh Test Plan

## Implementation Complete ✅

### What was implemented:
1. **BulkResultsView** sets a refresh flag when navigating to dashboard
2. **Dashboard** checks for this flag on WindowSizeMsg (which fires when view becomes active)
3. When flag is detected, dashboard:
   - Clears the flag
   - Invalidates active runs cache
   - Triggers data reload

### Key Code Changes:

#### 1. BulkResultsView (`bulk_results.go`):
- Lines 233-236: Sets refresh flag before navigating to dashboard
- Lines 307-311: Also sets flag when using button navigation
- Lines 324-327: Sets flag for all dashboard navigation paths

#### 2. Dashboard (`dashboard.go`):
- Lines 269-281: Checks for refresh flag on WindowSizeMsg
- Lines 587-589: 'r' key also uses InvalidateActiveRuns() instead of Clear()

#### 3. Cache Layer (`simple.go`, `hybrid.go`):
- `InvalidateActiveRuns()` method added to selectively clear only active runs
- Preserves terminal/completed runs in cache for performance

### How it works:
1. User submits bulk runs and sees results
2. User presses 'q' or selects [DASH] button
3. BulkResultsView sets "dashboard_needs_refresh" flag
4. Dashboard receives WindowSizeMsg when it becomes active
5. Dashboard detects flag, clears it, and refreshes data
6. New runs appear immediately without manual refresh

### Benefits:
- Automatic refresh when returning from bulk results
- No stale data issues
- Preserves performance by only clearing active runs
- Works with both 'q' key and [DASH] button navigation
- 'r' key on dashboard also uses targeted cache invalidation