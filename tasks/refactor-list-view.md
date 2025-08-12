# Refactor List View - Remove Parent State and Cache Passing

## Current Violations of Clean Architecture

### What List View is Doing Wrong:
- **Creating child views** (✗ wrong) - Creates Details view directly
- **Managing navigation** (✗ wrong) - Decides when to open Details
- **Passing parent state** (✗ wrong) - Sends all cache data to Details
- **Custom viewport implementation** (✗ wrong) - Should use shared ScrollableList

## Overview
The List view has multiple constructors that pass cache data to child views and stores parent state unnecessarily. It creates Details view directly instead of using navigation messages.

## Current Problems

### 1. Multiple Constructors with Cache Data
```go
func NewRunListView(client APIClient) *RunListView {
    // Calls NewRunListViewWithCache internally
}

func NewRunListViewWithCache(
    client APIClient,
    runs []models.RunResponse,
    cached bool,
    cachedAt time.Time,
    detailsCache map[string]*models.RunResponse,
    selectedIndex int,
    cache *cache.SimpleCache,
) *RunListView

func NewRunListViewWithCacheAndDimensions(
    client APIClient,
    runs []models.RunResponse,
    cached bool,
    cachedAt time.Time,
    detailsCache map[string]*models.RunResponse,
    selectedIndex int,
    width int,
    height int,
) *RunListView
```

### 2. Direct Details View Creation
```go
// In list.go:394, 401, 483, 491
func (v *RunListView) openRunDetails(run models.RunResponse) (tea.Model, tea.Cmd) {
    // WRONG: Creating Details view with all cache data
    detailsView := NewRunDetailsViewWithCache(
        v.client, 
        run, 
        v.runs,           // Passing all runs
        v.cached,         // Passing cache state
        v.cachedAt,       // Passing cache time
        v.detailsCache,   // Passing entire cache
        v.cache,
    )
    detailsView.width = v.width
    detailsView.height = v.height
    return detailsView, nil
}
```

### 3. Storing Cache Metadata
```go
type RunListView struct {
    // Cache metadata that could be internal
    runs         []models.RunResponse
    cached       bool
    cachedAt     time.Time
    detailsCache map[string]*models.RunResponse
    
    // Should just have:
    cache *cache.SimpleCache
}
```

### 4. Complex Cache Retry Logic
```go
// Lines 468-495: Complex retry logic for cache misses
if v.cacheRetryCount < v.maxCacheRetries {
    v.cacheRetryCount++
    time.Sleep(100 * time.Millisecond)
    // Try again...
    if detailed, ok := v.detailsCache[runID]; ok {
        detailsView := NewRunDetailsViewWithCache(...)
        // etc...
    }
}
```

## Proposed Solution

> **IMPORTANT**: See `tasks/refactor-navigation-pattern.md` for:
> - Shared ScrollableList component specification
> - Standard KeyMap definitions
> - Navigation message patterns
> - App router implementation

### 1. Use Shared ScrollableList Component
```go
type RunListView struct {
    client   APIClient
    cache    *cache.SimpleCache
    listView *components.ScrollableList // Shared component!
    
    // List-specific data only
    loading bool
    error   error
}

func NewRunListView(client APIClient) *RunListView {
    return &RunListView{
        client: client,
        cache:  cache.NewSimpleCache(),
        listView: components.NewScrollableList(
            components.WithColumns(4), // ID, Status, Repo, Time
            components.WithValueNavigation(true),
            components.WithKeymaps(components.DefaultKeyMap),
        ),
    }
}
```

### 2. Single Simple Constructor
```go
// AFTER: One simple constructor
func NewRunListView(client APIClient) *RunListView {
    return &RunListView{
        client: client,
        cache:  cache.NewSimpleCache(),
        table:  table.New(),
    }
}
```

### 2. Clean Struct - Cache Encapsulation
```go
type RunListView struct {
    client APIClient
    cache  *cache.SimpleCache
    
    // View state only
    table        table.Model
    selectedIdx  int
    loading      bool
    error        error
    
    // Remove: runs, cached, cachedAt, detailsCache
    // The cache handles all that internally
}
```

### 3. Navigation via Messages
```go
func (v *RunListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            if run := v.getSelectedRun(); run != nil {
                // Don't create Details view
                return v, func() tea.Msg {
                    return NavigateToDetailsMsg{
                        RunID: run.GetIDString(),
                    }
                }
            }
        case "q":
            return v, tea.Quit
        }
    }
}
```

### 4. Self-Loading Data
```go
func (v *RunListView) Init() tea.Cmd {
    return v.loadRuns()
}

func (v *RunListView) loadRuns() tea.Cmd {
    return func() tea.Msg {
        // Check cache first
        if runs := v.cache.GetRunsList(); runs != nil {
            return runsLoadedMsg{runs: runs}
        }
        
        // Load from API
        runs, err := v.client.ListRuns()
        if err != nil {
            return errMsg{err: err}
        }
        
        // Cache for next time
        v.cache.SetRunsList(runs)
        return runsLoadedMsg{runs: runs}
    }
}
```

### 5. Simplified Run Access
```go
func (v *RunListView) getSelectedRun() *models.RunResponse {
    runs := v.cache.GetRunsList()
    if runs == nil || v.selectedIdx >= len(runs) {
        return nil
    }
    return &runs[v.selectedIdx]
}
```

## Implementation Steps

### Phase 1: Remove Multiple Constructors
- [ ] Delete `NewRunListViewWithCache`
- [ ] Delete `NewRunListViewWithCacheAndDimensions`
- [ ] Simplify `NewRunListView` to not call other constructors

### Phase 2: Clean Up Struct
- [ ] Remove `runs` field (use cache)
- [ ] Remove `cached` field
- [ ] Remove `cachedAt` field
- [ ] Remove `detailsCache` field
- [ ] Remove cache retry fields

### Phase 3: Fix Navigation
- [ ] Update key handler to return `NavigateToDetailsMsg`
- [ ] Remove `openRunDetails` method
- [ ] Remove direct Details view creation

### Phase 4: Update Cache Usage
- [ ] Use cache methods instead of struct fields
- [ ] Load data in Init()
- [ ] Handle cache internally

## File Changes

### list.go
```go
// Simplified constructor
func NewRunListView(client APIClient) *RunListView {
    return &RunListView{
        client: client,
        cache:  cache.NewSimpleCache(),
        table:  table.New(),
    }
}

// Clean Update method
func (v *RunListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            if run := v.getSelectedRun(); run != nil {
                return v, func() tea.Msg {
                    return NavigateToDetailsMsg{
                        RunID: run.GetIDString(),
                    }
                }
            }
        }
    case runsLoadedMsg:
        v.updateTable(msg.runs)
        v.loading = false
    }
    return v, nil
}
```

## Success Metrics
- [ ] Single constructor: `NewRunListView(client)`
- [ ] No cache metadata in struct
- [ ] No direct Details view creation
- [ ] Navigation via messages only
- [ ] Cache handled internally

## Benefits
1. **Simpler API**: One constructor, one parameter
2. **Better Encapsulation**: Cache details hidden
3. **Loose Coupling**: No dependency on Details view
4. **Easier Testing**: Mock client, that's it
5. **Cleaner Code**: Remove ~100 lines of boilerplate

## Anti-Patterns to Avoid

### ❌ DON'T: Pass cache data to children
```go
NewRunDetailsViewWithCache(client, run, v.runs, v.cached, v.cachedAt, v.detailsCache, v.cache)
```

### ✅ DO: Pass only IDs
```go
return NavigateToDetailsMsg{RunID: run.GetIDString()}
```

### ❌ DON'T: Store cache metadata
```go
type RunListView struct {
    runs      []models.RunResponse
    cached    bool
    cachedAt  time.Time
}
```

### ✅ DO: Use cache directly
```go
func (v *RunListView) getRuns() []models.RunResponse {
    return v.cache.GetRunsList()
}
```

### ❌ DON'T: Complex retry logic
```go
if v.cacheRetryCount < v.maxCacheRetries {
    // Complex retry with sleep...
}
```

### ✅ DO: Simple cache check
```go
if runs := v.cache.GetRunsList(); runs != nil {
    // Use cached
} else {
    // Load fresh
}
```

## Example: Clean List View
```go
type RunListView struct {
    client APIClient
    cache  *cache.SimpleCache
    
    // View state only
    table       table.Model
    selectedIdx int
    loading     bool
}

func NewRunListView(client APIClient) *RunListView {
    return &RunListView{
        client: client,
        cache:  cache.NewSimpleCache(),
        table:  table.New(),
    }
}

func (v *RunListView) Init() tea.Cmd {
    return v.loadRuns()
}

func (v *RunListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            if run := v.getSelectedRun(); run != nil {
                return v, func() tea.Msg {
                    return NavigateToDetailsMsg{RunID: run.GetIDString()}
                }
            }
        }
    }
    // Handle table updates...
    return v, nil
}
```

## Related Tasks
- See `tasks/refactor-dashboard-view.md` for similar patterns
- See `tasks/refactor-view-constructors.md` for Details view
- See `tasks/refactor-navigation-pattern.md` for app router

## Testing Plan
1. List loads and displays runs
2. Selection and navigation work
3. Cache is used properly
4. No parent state leaks
5. Performance unchanged

## Notes
- List view is simpler than Dashboard/Create
- Focus on removing cache passing
- Keep table functionality intact