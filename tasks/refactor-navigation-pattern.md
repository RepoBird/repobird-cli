# Core Refactoring: Shared Components and Navigation Pattern

## THIS IS THE CENTRAL REFERENCE FILE
All view refactoring tasks should reference this file for:
1. Shared ScrollableList component
2. Shared Form component 
3. Standard KeyMaps
4. Navigation message patterns
5. App router implementation

## Overview
Currently, views create child views directly, passing parent state through complex constructors. This creates tight coupling and violates Bubble Tea's message-based architecture. We need a centralized navigation system where views communicate through messages and a central router manages view transitions.

## Current Anti-Pattern

### Views Creating Views
```go
// Dashboard creating Create view
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case "n":
        createView := NewCreateRunViewWithConfig(config)
        return createView, nil  // WRONG!
}

// Create view creating Details view
func (v *CreateRunView) handleRunCreated(msg) (tea.Model, tea.Cmd) {
    detailsView := NewRunDetailsViewWithCache(...)
    return detailsView, nil  // WRONG!
}

// List view creating Details view
func (v *RunListView) openRunDetails(run) (tea.Model, tea.Cmd) {
    detailsView := NewRunDetailsViewWithCache(...)
    return detailsView, nil  // WRONG!
}
```

## Core Components to Create

### 1. Shared ScrollableList Component
**File**: `internal/tui/components/scrollable_list.go`

Used by: Dashboard, List, Details, Bulk views

```go
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

func NewScrollableList(opts ...ScrollableListOption) *ScrollableList
func WithColumns(n int) ScrollableListOption
func WithKeyNavigation(enabled bool) ScrollableListOption  // For status-like views
func WithValueNavigation(enabled bool) ScrollableListOption // For normal lists
func WithDimensions(width, height int) ScrollableListOption
func WithKeymaps(km KeyMap) ScrollableListOption

// Handles ALL navigation, returns messages (not views!)
func (s *ScrollableList) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (s *ScrollableList) View() string
func (s *ScrollableList) SetItems(items [][]string)
```

### 2. Shared Form Component
**File**: `internal/tui/components/form.go`

Used by: Create view (special case - not a scrollable list)

```go
type FormComponent struct {
    fields       []FormField
    focusIndex   int
    insertMode   bool
    keymaps      FormKeyMap
}

type FormField struct {
    Name     string
    Type     FieldType  // TextInput, TextArea, Select
    Value    string
    Required bool
}

func NewForm(opts ...FormOption) *FormComponent
func WithFields(fields []FormField) FormOption
func WithFormKeymaps(km FormKeyMap) FormOption

// Handles form navigation and input
func (f *FormComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (f *FormComponent) View() string
func (f *FormComponent) GetValues() map[string]string
```

### 3. Standard KeyMaps
**File**: `internal/tui/components/keys.go` (ALREADY EXISTS - extend it)

Note: This file already exists with a KeyMap struct. We should extend the existing implementation rather than creating a new file.

```go
// Existing KeyMap in components/keys.go already has most bindings
// Just need to add Form-specific keymaps

// Form-specific keymaps (add to existing keys.go)
type FormKeyMap struct {
    KeyMap // Embed standard
    
    // Form-specific
    NextField     key.Binding  // tab
    PrevField     key.Binding  // shift+tab
    InsertMode    key.Binding  // i
    NormalMode    key.Binding  // esc
    Submit        key.Binding  // ctrl+s
}

// DefaultKeyMap already exists
// Add DefaultFormKeyMap
var DefaultFormKeyMap = FormKeyMap{
    KeyMap: DefaultKeyMap,
    // ... form-specific bindings
}
```

## Proposed Solution: Message-Based Navigation

### 1. Navigation Messages
Create `internal/tui/messages/navigation.go`:
```go
package messages

// Base navigation message
type NavigationMsg interface {
    IsNavigation() bool
}

// Specific navigation messages
type NavigateToCreateMsg struct {
    SelectedRepository string // Optional context
}

type NavigateToDetailsMsg struct {
    RunID      string
    FromCreate bool // Optional context
}

type NavigateToDashboardMsg struct{}

type NavigateToListMsg struct {
    SelectedIndex int // Optional: restore selection
}

type NavigateToBulkMsg struct{}

type NavigateBackMsg struct{}

type NavigateToErrorMsg struct {
    Error   error
    Message string
}

// Implement interface
func (NavigateToCreateMsg) IsNavigation() bool     { return true }
func (NavigateToDetailsMsg) IsNavigation() bool    { return true }
func (NavigateToDashboardMsg) IsNavigation() bool  { return true }
func (NavigateToListMsg) IsNavigation() bool       { return true }
func (NavigateToBulkMsg) IsNavigation() bool       { return true }
func (NavigateBackMsg) IsNavigation() bool         { return true }
func (NavigateToErrorMsg) IsNavigation() bool      { return true }
```

### 2. App Router
Update `internal/tui/app.go` to implement full Bubble Tea Model:
```go
type App struct {
    client     api.Client
    viewStack  []tea.Model  // Navigation history
    current    tea.Model
    cache      *cache.SimpleCache
}

// IMPORTANT: App must now implement tea.Model interface
func (a *App) Init() tea.Cmd {
    // Initialize with dashboard view
    a.current = views.NewDashboardView(a.client)
    return a.current.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle navigation messages first
    if navMsg, ok := msg.(messages.NavigationMsg); ok {
        return a.handleNavigation(navMsg)
    }
    
    // Otherwise delegate to current view
    newModel, cmd := a.current.Update(msg)
    
    // Check if the model changed (navigation occurred)
    if newModel != a.current {
        // Old pattern - view created child
        // We should log warning and handle gracefully
        a.current = newModel
    }
    
    return a, cmd
}

func (a *App) handleNavigation(msg messages.NavigationMsg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case messages.NavigateToCreateMsg:
        // Save current view to stack
        a.viewStack = append(a.viewStack, a.current)
        
        // Create new view with minimal params
        a.current = views.NewCreateRunView(a.client)
        
        // Set navigation context if provided
        if msg.SelectedRepository != "" {
            a.cache.SetNavigationContext("selected_repo", msg.SelectedRepository)
        }
        
        return a, a.current.Init()
        
    case messages.NavigateToDetailsMsg:
        a.viewStack = append(a.viewStack, a.current)
        
        // Create Details view with just ID
        a.current = views.NewRunDetailsView(a.client, msg.RunID)
        
        return a, a.current.Init()
        
    case messages.NavigateToDashboardMsg:
        // Clear stack - dashboard is home
        a.viewStack = nil
        a.current = views.NewDashboardView(a.client)
        // Clear navigation context when going home
        a.cache.ClearAllNavigationContext()
        return a, a.current.Init()
        
    case messages.NavigateBackMsg:
        if len(a.viewStack) > 0 {
            // Pop from stack
            a.current = a.viewStack[len(a.viewStack)-1]
            a.viewStack = a.viewStack[:len(a.viewStack)-1]
            
            // Refresh the view
            return a, a.current.Init()
        }
        // No history - go to dashboard
        return a.handleNavigation(messages.NavigateToDashboardMsg{})
        
    case messages.NavigateToListMsg:
        a.viewStack = append(a.viewStack, a.current)
        a.current = views.NewRunListView(a.client)
        return a, a.current.Init()
        
    case messages.NavigateToBulkMsg:
        a.viewStack = append(a.viewStack, a.current)
        a.current = views.NewBulkView(a.client)
        return a, a.current.Init()
    }
    
    return a, nil
}

func (a *App) View() string {
    // Delegate rendering to current view
    return a.current.View()
}

// Update Run() method to use App as the Model
func (a *App) Run() error {
    p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
    _, err := p.Run()
    return err
}
```

### 3. View Navigation Updates

#### Dashboard
```go
func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        switch msg.String() {
        case "n":
            // Get selected repository
            repo := d.getSelectedRepository()
            // Return navigation message with context
            return d, func() tea.Msg {
                return messages.NavigateToCreateMsg{
                    SelectedRepository: repo,
                }
            }
            
        case "enter":
            if run := d.getSelectedRun(); run != nil {
                return d, func() tea.Msg {
                    return messages.NavigateToDetailsMsg{
                        RunID: run.GetIDString(),
                    }
                }
            }
            
        case "l":
            return d, func() tea.Msg {
                return messages.NavigateToListMsg{}
            }
            
        case "b":
            return d, func() tea.Msg {
                return messages.NavigateToBulkMsg{}
            }
        }
}
```

#### Create View
```go
func (v *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        v.error = msg.err
        return v, nil
    }
    
    // Navigate to details
    return v, func() tea.Msg {
        return messages.NavigateToDetailsMsg{
            RunID:      msg.run.GetIDString(),
            FromCreate: true,
        }
    }
}

func (v *CreateRunView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        if msg.String() == "esc" && v.inputMode == components.NormalMode {
            // Navigate back
            return v, func() tea.Msg {
                return messages.NavigateBackMsg{}
            }
        }
}
```

#### Details View
```go
func (v *RunDetailsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "esc":
            return v, func() tea.Msg {
                return messages.NavigateBackMsg{}
            }
        case "d":
            // Go to dashboard
            return v, func() tea.Msg {
                return messages.NavigateToDashboardMsg{}
            }
        }
}
```

#### Bulk View Navigation (Complex Multi-Step)
```go
func (v *BulkView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        switch v.state {
        case FileSelection:
            if msg.String() == "enter" {
                // Move to next step internally
                v.state = RunEditing
                return v, nil
            }
        case RunEditing:
            if msg.String() == "ctrl+s" {
                // Submit all runs, then navigate
                return v, tea.Batch(
                    v.submitRuns(),
                    func() tea.Msg {
                        return messages.NavigateToDashboardMsg{}
                    },
                )
            }
        }
        
        // Always allow back navigation
        if msg.String() == "esc" {
            if v.state == FileSelection {
                // Navigate back from first step
                return v, func() tea.Msg {
                    return messages.NavigateBackMsg{}
                }
            }
            // Go to previous step
            v.state = v.previousState()
            return v, nil
        }
}
```

### 4. Context Sharing via Cache

**IMPORTANT**: The cache implementation is being refactored in `tasks/fix-cache-deadlock.md` to fix deadlock issues.
The context methods below will be implemented AFTER the cache deadlock fix is complete.

#### Context Methods (To Be Added After Cache Fix)

These methods will be added to SimpleCache after the deadlock issues are resolved:

```go
// Simple context storage without complex locking
// Implementation follows single-writer pattern from fix-cache-deadlock.md

func (c *SimpleCache) SetContext(key string, value interface{}) {
    // Will use the new lock-free patterns from cache fix
    // No nested locking, follows clear lock hierarchy
    c.contextData.Store(key, value)
}

func (c *SimpleCache) GetContext(key string) interface{} {
    // Lock-free read using atomic operations
    if val, ok := c.contextData.Load(key); ok {
        return val
    }
    return nil
}

func (c *SimpleCache) ClearContext(key string) {
    // Atomic delete operation
    c.contextData.Delete(key)
}

// Navigation-specific context that doesn't persist
func (c *SimpleCache) SetNavigationContext(key string, value interface{}) {
    // Temporary context for navigation between views
    // Automatically cleared on dashboard navigation
    c.SetContext("nav:"+key, value)
}

func (c *SimpleCache) GetNavigationContext(key string) interface{} {
    return c.GetContext("nav:" + key)
}

func (c *SimpleCache) ClearAllNavigationContext() {
    // Called when returning to dashboard
    // Clears all nav:* keys
    c.contextData.Range(func(k, v interface{}) bool {
        if key, ok := k.(string); ok && strings.HasPrefix(key, "nav:") {
            c.contextData.Delete(k)
        }
        return true
    })
}
```

## How Views Should Use These Components

### Dashboard View
```go
type DashboardView struct {
    client   APIClient
    cache    *cache.SimpleCache
    listView *components.ScrollableList // Uses shared component
}

func (d *DashboardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Delegate navigation to shared component
    if cmd := d.listView.Update(msg); cmd != nil {
        return d, cmd
    }
    // Handle dashboard-specific keys only
}
```

### List View
```go
type RunListView struct {
    client   APIClient
    cache    *cache.SimpleCache
    listView *components.ScrollableList // Uses shared component
}
```

### Details View (Special: Key Navigation)
```go
func NewRunDetailsView(client, runID) *RunDetailsView {
    return &RunDetailsView{
        listView: components.NewScrollableList(
            components.WithKeyNavigation(true), // Can navigate between keys!
            components.WithValueNavigation(true),
        ),
    }
}
```

### Create View (Special: Form)
```go
type CreateRunView struct {
    client APIClient
    cache  *cache.SimpleCache
    form   *components.FormComponent // Uses form, not list!
}
```

## Implementation Plan & Task Order

### Related Task Files (In Execution Order)
This is the central navigation refactoring that enables all other view refactoring tasks. The following tasks depend on this one and should be executed in this order:

1. **`tasks/fix-cache-deadlock.md`** - Fix cache deadlock issues FIRST (prerequisite for all other tasks)
2. **This file** (`tasks/refactor-navigation-pattern.md`) - Core navigation infrastructure
3. **`tasks/refactor-view-constructors.md`** - Simplify all view constructors after navigation is ready
4. **`tasks/refactor-dashboard-view.md`** - Refactor Dashboard to use new navigation and shared components
5. **`tasks/refactor-list-view.md`** - Refactor List view to use new navigation and shared components  
6. **`tasks/refactor-create-constructors.md`** - Refactor Create view constructors and form handling

### Phase 1: Create Navigation Infrastructure (This Task)
- [ ] Create `internal/tui/messages/navigation.go`
- [ ] Define all navigation message types
- [ ] Add navigation interface

### Phase 2: Update App Router (This Task)
- [ ] Transform app.go to full Bubble Tea Model (Init, Update, View)
- [ ] Add `handleNavigation` method to app.go
- [ ] Implement view stack for history
- [ ] Handle all navigation messages

### Phase 3: Create Shared Components (This Task)
- [ ] Create `internal/tui/components/scrollable_list.go`
- [ ] Create `internal/tui/components/form.go`
- [ ] Extend existing `internal/tui/components/keys.go` with FormKeyMap

### Phase 4: Update Views to Use Messages (Individual Task Files)
- [ ] Update Dashboard navigation (see `tasks/refactor-dashboard-view.md`)
- [ ] Update Create view navigation (see `tasks/refactor-create-constructors.md`)
- [ ] Update List view navigation (see `tasks/refactor-list-view.md`)
- [ ] Update Details view navigation (included here)
- [ ] Update Bulk view navigation (included here)

### Phase 5: Remove Direct View Creation (Part of Individual Tasks)
- [ ] Remove all `NewXxxView` calls from views
- [ ] Remove parent state passing
- [ ] Clean up constructors

### Phase 6: Add Context Methods to Cache (After Cache Fix)
- [ ] Wait for `tasks/fix-cache-deadlock.md` completion
- [ ] Add SetNavigationContext/GetNavigationContext/ClearAllNavigationContext using sync.Map
- [ ] Use atomic operations to avoid lock contention
- [ ] Document context usage patterns

### Phase 7: Testing & Documentation
- [ ] Update all view tests to use navigation messages
- [ ] Add router tests
- [ ] Update documentation

## Success Metrics
- [ ] Zero direct view creation in views
- [ ] All navigation via messages
- [ ] View stack for back navigation
- [ ] Minimal context via cache
- [ ] No parent state in constructors

## Benefits

### 1. Loose Coupling
- Views don't know about each other
- Easy to add new views
- Easy to change navigation flow

### 2. Better Testing
- Test views in isolation
- Mock navigation messages
- Test router separately

### 3. Consistent Navigation
- Central place for all navigation logic
- Easy to add features (breadcrumbs, history)
- Consistent back button behavior

### 4. Clean Architecture
- Follows Bubble Tea patterns
- Similar to web SPA routers
- Clear separation of concerns

## Example Flow: Create → Details

### Old Way (Direct Creation)
```
1. User submits run in Create view
2. Create view creates Details view with parent state
3. Create view returns Details view
4. Details view has all parent data
```

### New Way (Message-Based)
```
1. User submits run in Create view
2. Create view returns NavigateToDetailsMsg{RunID: "123"}
3. App router receives message
4. App router creates Details view with just ID
5. Details view loads its own data in Init()
6. No parent state passed
```

## Anti-Patterns to Avoid

### ❌ DON'T: Return different models
```go
func (v *SomeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return NewOtherView(...), nil  // WRONG!
}
```

### ✅ DO: Return navigation messages
```go
func (v *SomeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    return v, func() tea.Msg { return NavigateToOtherMsg{} }
}
```

### ❌ DON'T: Pass parent state
```go
NewDetailsView(client, parentRuns, parentCache, ...)
```

### ✅ DO: Pass minimal params
```go
NewDetailsView(client, runID)
```

### ❌ DON'T: Manage navigation in views
```go
type View struct {
    parentView tea.Model
    returnTo   string
}
```

### ✅ DO: Let router manage navigation
```go
// Router handles all navigation state
type App struct {
    viewStack []tea.Model
}
```

## Error Navigation & Recovery

### Error View Handling
```go
// NavigateToErrorMsg can replace current view or push to stack
type NavigateToErrorMsg struct {
    Error       error
    Message     string
    Recoverable bool    // Can user go back?
    ReturnTo    string  // View to return to after acknowledgment
}

func (a *App) handleNavigation(msg messages.NavigationMsg) (tea.Model, tea.Cmd) {
    case messages.NavigateToErrorMsg:
        if msg.Recoverable {
            // Push to stack so user can go back
            a.viewStack = append(a.viewStack, a.current)
        } else {
            // Replace current view, clear stack
            a.viewStack = nil
        }
        
        a.current = views.NewErrorView(msg.Error, msg.Message, msg.Recoverable)
        return a, a.current.Init()
}

// Error view handles recovery
func (e *ErrorView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter", "esc":
            if e.recoverable {
                // Go back to previous view
                return e, func() tea.Msg {
                    return messages.NavigateBackMsg{}
                }
            } else {
                // Go to dashboard (home)
                return e, func() tea.Msg {
                    return messages.NavigateToDashboardMsg{}
                }
            }
        }
}
```

## View Init() Patterns

### Purpose of Init() Method
Each view's Init() method should:
1. Load initial data (from cache or API)
2. Set up subscriptions/polling if needed
3. Initialize sub-components
4. Return initial commands

### Examples for Each View Type

```go
// Dashboard Init - Load all data columns
func (d *DashboardView) Init() tea.Cmd {
    return tea.Batch(
        d.loadRepositories(),    // Load from cache/API
        d.loadRecentRuns(),      // Load from cache/API  
        d.loadRunDetails(),      // Load selected run details
        d.startPolling(),        // Start polling for updates
    )
}

// List View Init - Load runs list
func (l *RunListView) Init() tea.Cmd {
    // Check for cached position from navigation context
    if idx := l.cache.GetNavigationContext("list_selected_index"); idx != nil {
        l.selectedIndex = idx.(int)
    }
    
    return tea.Batch(
        l.loadRuns(),           // Load all runs
        l.restoreScrollPosition(), // Restore previous position
    )
}

// Create View Init - Setup form
func (c *CreateRunView) Init() tea.Cmd {
    // Pre-populate from navigation context if available
    if repo := c.cache.GetNavigationContext("selected_repo"); repo != nil {
        c.form.SetField("repository", repo.(string))
    }
    
    return tea.Batch(
        c.loadGitInfo(),        // Detect current repo/branch
        c.loadTemplates(),      // Load saved templates
        c.focusFirstField(),    // Focus first input
    )
}

// Details View Init - Load specific run
func (d *RunDetailsView) Init() tea.Cmd {
    return tea.Batch(
        d.loadRun(),           // Load run from cache/API
        d.startPolling(),      // Poll if run is active
    )
}

// Bulk View Init - Setup file selector
func (b *BulkView) Init() tea.Cmd {
    return tea.Batch(
        b.loadConfigFiles(),   // Scan for .json files
        b.initFileSelector(),  // Setup file selector component
    )
}
```

## Testing Examples

### Testing Navigation Messages
```go
// navigation_test.go
func TestNavigationMessages(t *testing.T) {
    tests := []struct {
        name     string
        msg      messages.NavigationMsg
        expected string
    }{
        {
            name:     "navigate to create",
            msg:      messages.NavigateToCreateMsg{},
            expected: "create_view",
        },
        {
            name: "navigate to details with ID",
            msg:  messages.NavigateToDetailsMsg{RunID: "123"},
            expected: "details_view",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.True(t, tt.msg.IsNavigation())
        })
    }
}
```

### Testing Router
```go
// app_test.go
func TestAppRouter(t *testing.T) {
    client := &MockAPIClient{}
    app := NewApp(client)
    
    // Test navigation to create view
    navMsg := messages.NavigateToCreateMsg{
        SelectedRepository: "test/repo",
    }
    
    model, cmd := app.handleNavigation(navMsg)
    assert.NotNil(t, model)
    assert.IsType(t, &App{}, model)
    
    // Verify view stack
    appModel := model.(*App)
    assert.Len(t, appModel.viewStack, 1)
    assert.IsType(t, &views.CreateRunView{}, appModel.current)
    
    // Test back navigation
    backMsg := messages.NavigateBackMsg{}
    model, _ = app.handleNavigation(backMsg)
    
    appModel = model.(*App)
    assert.Len(t, appModel.viewStack, 0)
}
```

### Testing Views with Navigation
```go
// dashboard_test.go
func TestDashboardNavigation(t *testing.T) {
    dash := NewDashboardView(&MockAPIClient{})
    
    // Test pressing 'n' returns navigation message
    model, cmd := dash.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
    
    assert.Equal(t, dash, model) // View returns itself
    assert.NotNil(t, cmd)
    
    // Execute command to get message
    msg := cmd()
    navMsg, ok := msg.(messages.NavigateToCreateMsg)
    assert.True(t, ok)
    assert.NotNil(t, navMsg)
}
```

### Mocking Navigation in Tests
```go
// test_helpers.go
type MockRouter struct {
    navigations []messages.NavigationMsg
}

func (m *MockRouter) HandleNavigation(msg messages.NavigationMsg) (tea.Model, tea.Cmd) {
    m.navigations = append(m.navigations, msg)
    return m, nil
}

func TestViewWithMockRouter(t *testing.T) {
    router := &MockRouter{}
    view := NewSomeView()
    
    // Trigger navigation
    _, cmd := view.Update(someMsg)
    msg := cmd()
    
    // Verify navigation was requested
    router.HandleNavigation(msg.(messages.NavigationMsg))
    assert.Len(t, router.navigations, 1)
}
```

## Migration Strategy

### Step 1: Add Without Breaking
1. Add navigation messages
2. Add router handling
3. Keep old navigation working

### Step 2: Gradual Migration
1. Update one view at a time
2. Test each view thoroughly
3. Keep both patterns temporarily

### Step 3: Remove Old Pattern
1. Remove direct view creation
2. Remove parent state passing
3. Clean up constructors

## Related Files
- `internal/tui/app.go` - Main router
- `internal/tui/messages/navigation.go` - New file
- All view files need updates

## Notes
- This is the most important refactor
- Enables all other clean architecture changes
- Should be done first or in parallel
- Will make the codebase much cleaner