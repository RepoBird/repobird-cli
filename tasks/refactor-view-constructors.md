# Refactor View Constructors - Clean State Management

## ⚠️ Prerequisites
**IMPORTANT**: This refactoring depends on completing the cache deadlock fix first (`tasks/fix-cache-deadlock.md`). The cache fix:
- Resolves deadlock issues with simplified hybrid cache
- Establishes the shared cache architecture that views will use
- Must be completed before starting this refactoring

## Overview
The TUI views currently have complex constructors that violate clean architecture principles. Views are tightly coupled through parent state passing, leading to constructor proliferation and difficult maintenance. This refactoring will simplify constructors to minimal parameters and use Bubble Tea's native message-based architecture for state management.

## Current Problems

### 1. Constructor Proliferation
RunDetailsView has 5 different constructors:
- `NewRunDetailsView(client, run)`
- `NewRunDetailsViewWithCache(client, run, parentRuns, cached, cachedAt, detailsCache, cache)`
- `NewRunDetailsViewWithConfig(config)` 
- `NewRunDetailsViewWithDashboardState(client, run, parentRuns, cached, cachedAt, detailsCache, width, height, selectedRepo, selectedRun, selectedDetail, focusedColumn)`
- `NewRunDetailsViewWithCacheAndDimensions(client, run, parentRuns, cached, cachedAt, detailsCache, width, height)`

### 2. Parent State Coupling
Views receive and store parent view state:
```go
// RunDetailsView stores parent data it shouldn't know about
type RunDetailsView struct {
    parentRuns         []models.RunResponse  // Why?
    parentCached       bool                  // Why? 
    parentCachedAt     time.Time            // Why?
    parentDetailsCache map[string]*models.RunResponse // Why?
    dashboardSelectedRepoIdx    int         // Dashboard's concern
    dashboardSelectedRunIdx     int         // Dashboard's concern
    dashboardSelectedDetailLine int         // Dashboard's concern
    dashboardFocusedColumn      int         // Dashboard's concern
}
```

### 3. Violation of Single Responsibility
Views are responsible for:
- Their own rendering (✓ correct)
- Their own state (✓ correct)
- Parent view state (✗ wrong)
- Creating child views (✗ wrong)
- Managing navigation (✗ wrong)

### 4. Difficult Testing
Testing requires mocking complex constructor parameters and parent state.

## Proposed Solution

> **IMPORTANT**: See `tasks/refactor-navigation-pattern.md` for:
> - Shared ScrollableList component specification
> - Standard KeyMap definitions
> - Navigation message patterns
> - App router implementation

### Core Principles
1. **Minimal Constructors**: Only pass essential, immutable dependencies
2. **Self-Loading Views**: Views load their own data in `Init()`
3. **Message-Based Navigation**: Use Bubble Tea messages for view transitions
4. **Shared Cache Instance**: Pass app-level cache to views for data consistency
5. **No Parent Coupling**: Views don't know about parent state

### New Constructor Pattern

```go
// BEFORE: Complex constructor with parent state
func NewRunDetailsView(
    client APIClient,
    run models.RunResponse,
    parentRuns []models.RunResponse,
    parentCached bool,
    parentCachedAt time.Time,
    parentDetailsCache map[string]*models.RunResponse,
    cache *cache.SimpleCache,
) *RunDetailsView

// AFTER: Minimal constructor with shared cache
func NewRunDetailsView(client APIClient, cache *cache.SimpleCache, runID string) *RunDetailsView {
    return &RunDetailsView{
        client: client,
        cache:  cache,  // Shared app-level cache (simplified hybrid after deadlock fix)
        runID:  runID,
        // Everything else loads in Init()
    }
}
```

### Navigation Pattern

Instead of views creating child views:

```go
// WRONG: Dashboard creates details view directly
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            // Creating child view directly - tight coupling!
            return NewRunDetailsView(d.client, d.selectedRun, d.runs, ...), nil
        }
}

// RIGHT: Dashboard returns navigation message
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        if msg.String() == "enter" {
            // Save any needed state to cache
            d.cache.SetNavigationContext("dashboard", d.getState())
            // Return message for parent to handle
            return d, func() tea.Msg {
                return NavigateToDetailsMsg{RunID: d.selectedRun.ID}
            }
        }
}

// App/parent handles navigation
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case NavigateToDetailsMsg:
        // App creates the view with minimal params and shared cache
        return NewRunDetailsView(a.client, a.cache, msg.RunID), nil
    }
}
```

### State Loading Pattern

```go
func (v *RunDetailsView) Init() tea.Cmd {
    return tea.Batch(
        v.loadRunDetails(),      // Load from cache or API
        v.restoreViewState(),    // Restore any saved view state
        textinput.Blink,
    )
}

func (v *RunDetailsView) loadRunDetails() tea.Cmd {
    return func() tea.Msg {
        // First check cache
        if cached := v.cache.GetRunDetails(v.runID); cached != nil {
            return runDetailsLoadedMsg{run: cached}
        }
        
        // Load from API
        run, err := v.client.GetRun(v.runID)
        if err != nil {
            return errMsg{err: err}
        }
        
        // Cache for next time
        v.cache.SetRunDetails(v.runID, run)
        return runDetailsLoadedMsg{run: run}
    }
}
```

## Implementation Plan

### Prerequisites
- [ ] Complete cache deadlock fix from `tasks/fix-cache-deadlock.md` (5.5 hours)
  - This establishes the simplified hybrid cache architecture
  - Views will receive the fixed cache instance from app level
  - Prevents deadlocks and improves performance

### Phase 1: Define Navigation Messages (After cache fix)
- [ ] Create `internal/tui/messages/navigation.go`
- [ ] Define standard navigation messages:
  - `NavigateToDetailsMsg{RunID string}`
  - `NavigateToDashboardMsg{}`
  - `NavigateToListMsg{}`
  - `NavigateBackMsg{}`

### Phase 2: Refactor RunDetailsView (Week 1)
- [ ] Simplify constructor to `NewRunDetailsView(client, cache, runID)`
- [ ] Remove all parent-related fields
- [ ] Implement `loadRunDetails()` in `Init()`
- [ ] Update tests to use new constructor

### Phase 3: Refactor Parent Views (Week 2)
- [ ] Update Dashboard to return navigation messages
- [ ] Update List view to return navigation messages
- [ ] Update Create view to return navigation messages
- [ ] Remove direct child view creation

### Phase 4: Update App Router (Week 2)
- [ ] Implement navigation message handling in app.go
- [ ] Initialize app with shared cache instance
- [ ] Pass shared cache to all view constructors
- [ ] Create views with minimal constructors (client, cache, ID)
- [ ] Manage view stack for back navigation

### Phase 5: Refactor Other Views (Week 3)
- [ ] Apply same pattern to CreateView
- [ ] Apply same pattern to ListView
- [ ] Apply same pattern to BulkView

## Success Metrics

### Quantitative
- [ ] Maximum 3 parameters per constructor (client + cache + ID/config)
- [ ] Zero parent state fields in view structs
- [ ] 100% of navigation via messages
- [ ] 50% reduction in constructor test complexity
- [ ] Single shared cache instance across all views

### Qualitative
- [ ] Views are self-contained and reusable
- [ ] Navigation logic centralized in app router
- [ ] Tests are simpler and more focused
- [ ] Code is more maintainable

## Migration Strategy

### Backward Compatibility
1. Keep old constructors temporarily with deprecation notices
2. Update one view at a time
3. Run tests after each view update
4. Remove deprecated constructors after all updates

### Testing Plan
1. Create tests for new minimal constructors
2. Test navigation messages
3. Test state loading from cache
4. Test error handling

## Example: Complete RunDetailsView Refactor

```go
// details.go - Minimal, self-contained view
type RunDetailsView struct {
    client   APIClient
    runID    string
    cache    *cache.SimpleCache  // Shared from app
    
    // View's own state only
    run      *models.RunResponse
    loading  bool
    error    error
    viewport viewport.Model
}

func NewRunDetailsView(client APIClient, cache *cache.SimpleCache, runID string) *RunDetailsView {
    return &RunDetailsView{
        client: client,
        cache:  cache,  // Received from app router
        runID:  runID,
        viewport: viewport.New(80, 20),
    }
}

func (v *RunDetailsView) Init() tea.Cmd {
    return v.loadRunDetails()
}

func (v *RunDetailsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case runDetailsLoadedMsg:
        v.run = msg.run
        v.loading = false
        v.updateContent()
        return v, nil
        
    case tea.KeyMsg:
        if msg.String() == "q" {
            // Don't create parent view - just signal navigation
            return v, func() tea.Msg { return NavigateBackMsg{} }
        }
    }
    // ... handle other messages
}
```

## Anti-Patterns to Avoid

### ❌ DON'T: Pass parent state through constructors
```go
NewView(client, data, parentState, parentCache, parentDimensions, ...)
```

### ✅ DO: Pass only essential dependencies
```go
NewView(client, cache, id)  // cache is shared app-level instance
```

### ❌ DON'T: Create child views directly
```go
return NewChildView(...), nil
```

### ✅ DO: Return navigation messages
```go
return v, func() tea.Msg { return NavigateToChildMsg{} }
```

### ❌ DON'T: Store parent state in child
```go
type ChildView struct {
    parentSelectedIndex int
    parentCache map[string]interface{}
}
```

### ✅ DO: Load own state from shared cache if needed
```go
func (v *ChildView) Init() tea.Cmd {
    // Cache is shared, so data is consistent across views
    if ctx := v.cache.GetContext("navigation"); ctx != nil {
        // Use context if needed
    }
    // Load view-specific data
    return v.loadData()
}
```

## Related Documentation
- [Bubble Tea Architecture](https://github.com/charmbracelet/bubbletea#architecture)
- [Elm Architecture](https://guide.elm-lang.org/architecture/)
- `tasks/refactor-global-state.md` - Previous refactoring removing global state

## Conclusion
This refactoring will result in cleaner, more maintainable code that follows Bubble Tea's intended architecture. Views will be truly independent, reusable components that communicate through messages rather than tight coupling.