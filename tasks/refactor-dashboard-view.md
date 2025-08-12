# Refactor Dashboard View - Remove Parent State Coupling

## Overview
The Dashboard view currently creates child views directly and passes excessive parent state to them. This violates Bubble Tea's message-based architecture and creates tight coupling between views.

## Current Problems

### 1. Direct Child View Creation
Dashboard creates child views directly instead of returning navigation messages:
```go
// WRONG: Dashboard creating CreateView directly
case "n":
    config := CreateRunViewConfig{
        Client:             d.client,
        ParentRuns:         d.allRuns,
        ParentCached:       d.cached,
        ParentCachedAt:     d.cachedAt,
        ParentDetailsCache: d.detailsCache,
        SelectedRepository: selectedRepo,
        Cache:              d.cache,
    }
    createView := NewCreateRunViewWithConfig(config)
    return createView, nil

// WRONG: Dashboard creating DetailsView with dashboard state
case "enter":
    detailsView := NewRunDetailsViewWithDashboardState(
        d.client,
        d.selectedRunData,
        d.allRuns,
        d.cached,
        d.cachedAt,
        d.detailsCache,
        d.width, d.height,
        d.selectedRepoIdx,
        d.selectedRunIdx,
        d.selectedDetailLine,
        d.focusedColumn,
    )
    return detailsView, nil
```

### 2. Storing Child View References
```go
type DashboardView struct {
    runListView *RunListView  // Why store child view?
    // ... other fields
}
```

### 3. Passing Parent State to Children
Dashboard passes:
- All runs data (`d.allRuns`)
- Cache timestamps (`d.cached`, `d.cachedAt`)
- Details cache (`d.detailsCache`)
- Dashboard selection state (indices, focused column)
- Parent dimensions

## Current Violations of Clean Architecture

### What Dashboard is Doing Wrong:
- **Creating child views** (✗ wrong) - Returns new view models directly
- **Managing navigation** (✗ wrong) - Decides which view comes next
- **Passing parent state** (✗ wrong) - Sends its data to children
- **Custom viewport implementation** (✗ wrong) - Should use shared component

## Proposed Solution

> **IMPORTANT**: See `tasks/refactor-navigation-pattern.md` for:
> - Shared ScrollableList component specification
> - Standard KeyMap definitions
> - Navigation message patterns
> - App router implementation

## Proposed Solution

### 1. Use Shared Scrollable List Component
Dashboard should use the same scrollable viewport component as other views:
```go
// Use shared component
type DashboardView struct {
    client    APIClient
    cache     *cache.SimpleCache
    listView  *components.ScrollableList // Shared component!
    
    // Dashboard-specific data
    repositories []models.Repository
    runs        []*models.RunResponse
}

// Configure the shared component
func NewDashboardView(client APIClient) *DashboardView {
    return &DashboardView{
        client: client,
        cache:  cache.NewSimpleCache(),
        listView: components.NewScrollableList(
            components.WithColumns(3),        // 3-column layout
            components.WithDimensions(80, 20),
            components.WithKeyNavigation(true), // Allow key navigation
            components.WithValueNavigation(true), // Allow value navigation
        ),
    }
}
```

### 2. Use Navigation Messages (Don't Manage Navigation)
```go
// navigation_messages.go
type NavigateToCreateMsg struct {
    SelectedRepository string // Only what's truly needed
}

type NavigateToDetailsMsg struct {
    RunID string // Just the ID, view loads its own data
}

type NavigateBackMsg struct {
    // No data needed
}
```

### 3. Dashboard Only Handles Its Own State
```go
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // First, delegate to the shared scrollable list component
    if cmd := d.listView.HandleKey(msg); cmd != nil {
        return d, cmd
    }
    
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Only handle dashboard-specific keys
        // Navigation keys are NOT handled here!
        switch msg.String() {
        case "r":
            // Refresh data - dashboard specific
            return d, d.loadDashboardData()
        case "f":
            // Filter - dashboard specific
            d.showFilterInput = true
            return d, nil
        }
        // Navigation keys (n, enter, q) are handled by the shared keymap!
    }
}
```

### 3. Simplify Dashboard Constructor
```go
// BEFORE: Complex initialization
func NewDashboardView(client APIClient) *DashboardView {
    dashboard := &DashboardView{
        client: client,
        cache:  cache.NewSimpleCache(),
        // ... other fields
    }
    dashboard.runListView = NewRunListView(client) // Remove this
    return dashboard
}

// AFTER: Simple, self-contained
func NewDashboardView(client APIClient) *DashboardView {
    return &DashboardView{
        client:   client,
        cache:    cache.NewSimpleCache(),
        viewport: viewport.New(80, 20),
        // Only own state
    }
}
```

### 4. Dashboard Loads Own Data
```go
func (d *DashboardView) Init() tea.Cmd {
    return tea.Batch(
        d.loadDashboardData(),  // Load from cache or API
        d.loadUserInfo(),       // Load user info
        spinner.Tick,
    )
}

func (d *DashboardView) loadDashboardData() tea.Cmd {
    return func() tea.Msg {
        // First check cache
        if repos := d.cache.GetRepositories(); repos != nil {
            return dashboardDataLoadedMsg{repositories: repos}
        }
        
        // Load from API
        repos, err := d.client.ListRepositories()
        if err != nil {
            return errMsg{err: err}
        }
        
        // Cache for next time
        d.cache.SetRepositories(repos)
        return dashboardDataLoadedMsg{repositories: repos}
    }
}
```

## Shared Components Needed

### ScrollableList Component
```go
// components/scrollable_list.go
type ScrollableList struct {
    viewport     viewport.Model
    items        [][]string      // Multi-column data
    selected     int
    focusedCol   int
    
    // Configuration
    columns      int
    keyNav       bool  // Navigate between keys (like status view)
    valueNav     bool  // Navigate between values (normal)
    keymaps      KeyMap
}

type ScrollableListOption func(*ScrollableList)

func WithColumns(n int) ScrollableListOption
func WithKeyNavigation(enabled bool) ScrollableListOption
func WithValueNavigation(enabled bool) ScrollableListOption
func WithCustomKeymaps(km KeyMap) ScrollableListOption

// Handles standard navigation, returns navigation messages
func (s *ScrollableList) HandleKey(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, s.keymaps.Up):
            s.selected--
            return nil
        case key.Matches(msg, s.keymaps.Down):
            s.selected++
            return nil
        case key.Matches(msg, s.keymaps.Enter):
            // Return navigation message, not create view!
            return func() tea.Msg {
                return ItemSelectedMsg{Index: s.selected}
            }
        }
    }
}
```

### Standard Keymaps
```go
// components/keymaps.go
type KeyMap struct {
    Up     key.Binding
    Down   key.Binding
    Left   key.Binding
    Right  key.Binding
    Enter  key.Binding
    Back   key.Binding
    Quit   key.Binding
    New    key.Binding
    Delete key.Binding
    // View-specific keys can be added
}

var DefaultKeyMap = KeyMap{
    Up:     key.NewBinding(key.WithKeys("k", "up")),
    Down:   key.NewBinding(key.WithKeys("j", "down")),
    Left:   key.NewBinding(key.WithKeys("h", "left")),
    Right:  key.NewBinding(key.WithKeys("l", "right")),
    Enter:  key.NewBinding(key.WithKeys("enter")),
    Back:   key.NewBinding(key.WithKeys("esc", "backspace")),
    Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c")),
    New:    key.NewBinding(key.WithKeys("n")),
    Delete: key.NewBinding(key.WithKeys("d")),
}
```

## Implementation Steps

### Phase 1: Create Shared Components
- [ ] Create `components/scrollable_list.go`
- [ ] Create `components/keymaps.go`
- [ ] Implement standard navigation handling
- [ ] Add configuration options

### Phase 2: Remove Child View Storage
- [ ] Remove `runListView` field from DashboardView
- [ ] Remove any other child view references
- [ ] Update rendering to not depend on child views

### Phase 2: Implement Navigation Messages
- [ ] Create `internal/tui/messages/navigation.go`
- [ ] Define `NavigateToCreateMsg`, `NavigateToDetailsMsg`, etc.
- [ ] Update Dashboard to return these messages

### Phase 3: Remove Parent State Passing
- [ ] Remove `ParentRuns`, `ParentCached`, etc. from child view creation
- [ ] Update child views to load their own data
- [ ] Use cache for truly shared data only

### Phase 4: Update App Router
- [ ] Handle navigation messages in app.go
- [ ] Create views with minimal parameters
- [ ] Manage view stack for navigation

## Success Metrics
- [ ] Dashboard constructor has only `client` parameter
- [ ] No child view references in Dashboard struct
- [ ] All navigation via messages
- [ ] No parent state passed to children

## Example: Clean Dashboard
```go
// dashboard.go - Clean, self-contained
type DashboardView struct {
    client APIClient
    cache  *cache.SimpleCache
    
    // Own state only
    repositories []models.Repository
    selectedRepo int
    runs         []*models.RunResponse
    selectedRun  int
    
    loading bool
    error   error
}

func NewDashboardView(client APIClient) *DashboardView {
    return &DashboardView{
        client: client,
        cache:  cache.NewSimpleCache(),
    }
}

func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "n":
            // Don't create child, just signal navigation
            return d, func() tea.Msg {
                return NavigateToCreateMsg{
                    SelectedRepository: d.getSelectedRepoName(),
                }
            }
        case "enter":
            // Don't create child, just signal navigation
            return d, func() tea.Msg {
                return NavigateToDetailsMsg{
                    RunID: d.getSelectedRunID(),
                }
            }
        }
    }
    // ... handle other messages
}
```

## Anti-Patterns to Avoid

### ❌ DON'T: Create child views directly
```go
return NewCreateRunViewWithConfig(config), nil
```

### ✅ DO: Return navigation messages
```go
return d, func() tea.Msg { return NavigateToCreateMsg{} }
```

### ❌ DON'T: Pass parent state to children
```go
NewDetailsView(client, parentRuns, parentCache, dashboardState, ...)
```

### ✅ DO: Pass only essential data
```go
// In app router:
NewDetailsView(client, msg.RunID)
```

### ❌ DON'T: Store child view references
```go
type DashboardView struct {
    runListView *RunListView
}
```

### ✅ DO: Keep views independent
```go
type DashboardView struct {
    // Only own state
}
```

## Related Files
- `internal/tui/views/dashboard.go`
- `internal/tui/views/dash_*.go` (helper files)
- `internal/tui/app.go` (will handle navigation)

## Dependencies
- Must be done before child view refactoring
- Requires navigation message infrastructure
- Requires app router updates