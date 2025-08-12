# Refactor Create View Constructors - Remove Parent State Coupling

## Overview
The Create view has multiple constructors that accept parent state and directly creates Details view after run creation. This violates clean architecture and Bubble Tea's message-based patterns.

## Current Violations of Clean Architecture

### What Create View is Doing Wrong:
- **Creating child views** (✗ wrong) - Creates Details view after run creation
- **Managing navigation** (✗ wrong) - Decides to go to Details after submit
- **Storing parent state** (✗ wrong) - Keeps parentRuns, parentCache, etc.
- **Not using standard keymaps** (✗ wrong) - Custom input handling

## Current Problems

### 1. Multiple Constructors with Parent State
```go
// Constructor proliferation
func NewCreateRunView(client APIClient) *CreateRunView {
    // Creates cache and calls NewCreateRunViewWithCache
    return NewCreateRunViewWithCache(client, nil, false, time.Time{}, nil, cache)
}

func NewCreateRunViewWithConfig(config CreateRunViewConfig) *CreateRunView
func NewCreateRunViewWithCache(
    client APIClient,
    parentRuns []models.RunResponse,      // Parent state!
    parentCached bool,                     // Parent state!
    parentCachedAt time.Time,             // Parent state!
    parentDetailsCache map[string]*models.RunResponse, // Parent cache!
    cache *cache.SimpleCache,
) *CreateRunView
```

### 2. Storing Parent State in Struct
```go
type CreateRunView struct {
    // Parent state that shouldn't be here
    parentRuns         []models.RunResponse
    parentCached       bool
    parentCachedAt     time.Time
    parentDetailsCache map[string]*models.RunResponse
    
    // Actual create view fields...
}
```

### 3. Direct Child View Creation
```go
// In create_submission.go:275
func (v *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd) {
    // WRONG: Creating Details view directly with parent state
    detailsView := NewRunDetailsViewWithCacheAndDimensions(
        v.client,
        msg.run,
        v.parentRuns,        // Passing parent state
        v.parentCached,      // Passing parent state
        v.parentCachedAt,    // Passing parent state
        v.parentDetailsCache, // Passing parent cache
        v.width,
        v.height,
    )
    return detailsView, detailsView.Init()
}
```

### 4. Config with Parent Dependencies
```go
type CreateRunViewConfig struct {
    Client             APIClient
    ParentRuns         []models.RunResponse  // Why?
    ParentCached       bool                  // Why?
    ParentCachedAt     time.Time            // Why?
    ParentDetailsCache map[string]*models.RunResponse // Why?
    SelectedRepository string                // This is OK - context
    Cache              *cache.SimpleCache
}
```

## Proposed Solution

> **IMPORTANT**: See `tasks/refactor-navigation-pattern.md` for:
> - Shared Form component specification (Create view uses Form, not ScrollableList)
> - Standard FormKeyMap definitions
> - Navigation message patterns
> - App router implementation

### 1. Single Minimal Constructor
```go
// AFTER: One simple constructor
func NewCreateRunView(client APIClient) *CreateRunView {
    v := &CreateRunView{
        client: client,
        cache:  cache.NewSimpleCache(),
    }
    v.initializeInputFields()
    v.loadFormData() // Load from cache if exists
    
    // Check for context from previous view
    if repo := v.cache.GetContext("selected_repo"); repo != "" {
        v.fields[0].SetValue(repo)
        v.cache.ClearContext("selected_repo")
    }
    
    return v
}
```

### 2. Clean Struct - No Parent State
```go
type CreateRunView struct {
    client APIClient
    cache  *cache.SimpleCache
    
    // Form fields only
    fields       []textinput.Model
    promptArea   textarea.Model
    contextArea  textarea.Model
    filePathInput textinput.Model
    
    // View state only
    focusIndex      int
    submitting      bool
    isSubmitting    bool
    error           error
    runType         models.RunType
    showContext     bool
    
    // Components
    configLoader    *config.ConfigLoader
    repoSelector    *components.RepositorySelector
    statusLine      *components.StatusLine
    
    // NO parentRuns, parentCached, parentDetailsCache!
}
```

### 3. Form Component (Special Case - Not Scrollable List)
Create view is a form, not a scrollable list, so it needs different handling:
```go
type CreateRunView struct {
    client APIClient
    cache  *cache.SimpleCache
    form   *components.FormComponent // Shared form component
    
    // Form-specific state only
    submitting bool
    error      error
}

// Use shared form component with standard keymaps
func NewCreateRunView(client APIClient) *CreateRunView {
    return &CreateRunView{
        client: client,
        cache:  cache.NewSimpleCache(),
        form: components.NewForm(
            components.WithFields([]FormField{
                {Name: "repository", Type: TextInput},
                {Name: "prompt", Type: TextArea},
                {Name: "context", Type: TextArea, Optional: true},
            }),
            components.WithKeymaps(components.FormKeyMap), // Standard form navigation
        ),
    }
}
```

### 4. Navigation via Messages (Not Managing Navigation)
```go
// After run creation
func (v *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        v.error = msg.err
        v.submitting = false
        return v, nil
    }
    
    // Clear form and save success
    v.clearAllFields()
    v.cache.SetLastCreatedRun(msg.run.GetIDString())
    
    // Don't create Details view - return navigation message
    return v, func() tea.Msg {
        return NavigateToDetailsMsg{
            RunID: msg.run.GetIDString(),
            FromCreate: true,
        }
    }
}
```

### 4. Self-Loading Form Data
```go
func (v *CreateRunView) Init() tea.Cmd {
    // Load saved form data from cache
    if data := v.cache.GetFormData(); data != nil {
        v.populateFromCache(data)
    }
    
    // Auto-detect git repository
    v.autofillRepository()
    
    return tea.Batch(
        v.loadFileHashCache(),
        textinput.Blink,
        textarea.Blink,
    )
}
```

## Implementation Steps

### Phase 1: Remove Multiple Constructors
- [ ] Delete `NewCreateRunViewWithConfig`
- [ ] Delete `NewCreateRunViewWithCache`
- [ ] Update `NewCreateRunView` to be self-contained
- [ ] Update all callers (Dashboard, etc.)

### Phase 2: Remove Parent State from Struct
- [ ] Remove `parentRuns` field
- [ ] Remove `parentCached` field
- [ ] Remove `parentCachedAt` field
- [ ] Remove `parentDetailsCache` field
- [ ] Delete `CreateRunViewConfig` struct

### Phase 3: Fix Navigation
- [ ] Update `handleRunCreated` to return `NavigateToDetailsMsg`
- [ ] Remove direct Details view creation
- [ ] Add navigation message types

### Phase 4: Update Dashboard
- [ ] Dashboard should not pass parent state to Create
- [ ] Use cache context for selected repository
- [ ] Return navigation message

## File Changes

### create.go
```go
// Simple constructor
func NewCreateRunView(client APIClient) *CreateRunView {
    v := &CreateRunView{
        client: client,
        cache:  cache.NewSimpleCache(),
    }
    v.initializeInputFields()
    
    // Load any saved form data
    if data := v.cache.GetFormData(); data != nil {
        v.loadFormData(data)
    }
    
    // Check for repository context
    if repo := v.cache.GetContext("selected_repo"); repo != "" {
        v.fields[0].SetValue(repo)
    }
    
    return v
}
```

### create_submission.go
```go
func (v *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        // Handle error...
        return v, nil
    }
    
    // Return navigation message, not new view
    return v, func() tea.Msg {
        return NavigateToDetailsMsg{
            RunID: msg.run.GetIDString(),
        }
    }
}
```

### dashboard.go
```go
// Dashboard creates Create view
case "n":
    // Save context if needed
    if repo := d.getSelectedRepository(); repo != "" {
        d.cache.SetContext("selected_repo", repo)
    }
    
    // Return navigation message
    return d, func() tea.Msg {
        return NavigateToCreateMsg{}
    }
```

## Success Metrics
- [ ] Single constructor: `NewCreateRunView(client)`
- [ ] No parent state in CreateRunView struct
- [ ] No direct child view creation
- [ ] Navigation via messages only
- [ ] Form data persisted via cache

## Anti-Patterns to Avoid

### ❌ DON'T: Multiple constructors for different contexts
```go
NewCreateRunView(client)
NewCreateRunViewWithCache(client, parentState...)
NewCreateRunViewWithConfig(config)
```

### ✅ DO: Single constructor
```go
NewCreateRunView(client)  // Always this
```

### ❌ DON'T: Pass parent state
```go
type CreateRunViewConfig struct {
    ParentRuns []models.RunResponse
    ParentDetailsCache map[string]*models.RunResponse
}
```

### ✅ DO: Use cache for context
```go
// Set context before navigation
cache.SetContext("selected_repo", repo)

// Read context in Init()
if repo := cache.GetContext("selected_repo"); repo != "" {
    // Use it
}
```

### ❌ DON'T: Create child views
```go
return NewRunDetailsView(...), nil
```

### ✅ DO: Return messages
```go
return v, func() tea.Msg { return NavigateToDetailsMsg{...} }
```

## Related Tasks
- See `tasks/refactor-dashboard-view.md` for Dashboard changes
- See `tasks/refactor-view-constructors.md` for Details view
- See `tasks/refactor-navigation-pattern.md` for app router

## Testing Plan
1. Create run flow works end-to-end
2. Form data persists between navigations
3. Repository pre-selection from Dashboard works
4. Navigation to Details after creation works
5. Error handling preserved

## Notes
- This focuses on constructor/state issues, not code splitting
- The code splitting task (`refactor-create-view.md`) is separate
- Both can be done independently but this should come first