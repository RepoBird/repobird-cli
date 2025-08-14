# Help View Architecture Refactor

## Problem Statement

The help system in RepoBird CLI is currently implemented as an **overlay** within the dashboard view rather than a proper standalone view. This creates several UX inconsistencies:

### Current Issues
1. **Navigation Inconsistency**: Normal navigation commands don't work as expected:
   - `h` (back) doesn't work - help isn't in the navigation stack
   - `q` exits the program (dashboard behavior) instead of closing help
   - `b` and `ESC` close help but this is handled specially within dashboard

2. **Not a True View**: Help is implemented as:
   - Dashboard overlay state (`showDocs = true` in dashboard.go:562)
   - Special handling in `dash_help_overlay.go`
   - Not part of the navigation stack managed by App router

3. **Keymap Processing Issues**:
   - `ActionNavigateHelp` is explicitly ignored in app.go:474
   - Dashboard handles '?' key directly instead of through navigation
   - Key processing still thinks it's in dashboard context

## Current Implementation Analysis

### Files Involved
- `internal/tui/components/help_view.go` - Reusable help component (good foundation)
- `internal/tui/views/dash_help_overlay.go` - Dashboard-specific overlay handling
- `internal/tui/views/dashboard.go` - Sets `showDocs = true` on '?' key
- `internal/tui/keymap/core.go` - Defines `ActionNavigateHelp` but it's not used
- `internal/tui/app.go` - Ignores `ActionNavigateHelp` (line 474)

### Architecture Pattern Violation
The current implementation violates the established TUI patterns:
- **❌ Not using navigation messages** - Help toggle is handled within dashboard
- **❌ Not a proper view in the stack** - Can't navigate back properly
- **❌ Inconsistent with other views** - All other views (Details, Status, Create, etc.) are proper views

## Proposed Solution

### 1. Create Standalone HelpView

Create `internal/tui/views/help.go` that follows the standard view pattern:

```go
type HelpView struct {
    client APIClient
    cache  *cache.SimpleCache
    
    // Embed the existing help component
    helpComponent *components.HelpView
    
    // Standard view fields
    layout *components.WindowLayout
    width  int
    height int
    keys   *keymap.KeyMap
    
    // Implement CoreViewKeymap for proper key handling
    disabledKeys map[string]bool
}

// Standard constructor pattern
func NewHelpView(client APIClient, cache *cache.SimpleCache) *HelpView {
    return &HelpView{
        client:        client,
        cache:         cache,
        helpComponent: components.NewHelpView(),
        disabledKeys: map[string]bool{
            // Help view specific disabled keys if any
        },
    }
}

// Implement tea.Model interface
func (h *HelpView) Init() tea.Cmd { ... }
func (h *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (h *HelpView) View() string { ... }

// Implement CoreViewKeymap interface
func (h *HelpView) IsKeyDisabled(keyString string) bool { ... }
func (h *HelpView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) { ... }
```

### 2. Add Navigation Message

Add to `internal/tui/messages/navigation.go`:

```go
// NavigateToHelpMsg requests navigation to the help view
type NavigateToHelpMsg struct{}

func (NavigateToHelpMsg) IsNavigation() bool { return true }
```

### 3. Update App Router

Modify `internal/tui/app.go`:

```go
// In processKeyWithFiltering() - handle ActionNavigateHelp
case keymap.ActionNavigateHelp:
    navMsg = messages.NavigateToHelpMsg{}

// In handleNavigation() - add case for help
case messages.NavigateToHelpMsg:
    view := views.NewHelpView(a.client, a.cache)
    a.pushView(view)
    return view, view.Init()
```

### 4. Remove Dashboard Overlay Code

Clean up dashboard implementation:
- Remove `showDocs` field from DashboardView
- Remove help handling from dashboard Update()
- Delete `dash_help_overlay.go` (move any useful code to new HelpView)
- Remove special '?' key handling from dashboard

### 5. Update Keymap Registry

Ensure proper keymap handling:
- '?' should trigger `ActionNavigateHelp` globally
- Help view should handle standard navigation keys properly
- 'q', 'b', 'h', 'ESC' should all navigate back from help

## Implementation Checklist

### Phase 1: Create HelpView Structure
- [ ] Create `internal/tui/views/help.go` with proper view structure
- [ ] Implement tea.Model interface (Init, Update, View)
- [ ] Implement CoreViewKeymap interface
- [ ] Use WindowLayout for consistent borders
- [ ] Embed existing `components.HelpView` for content

### Phase 2: Navigation Integration
- [ ] Add `NavigateToHelpMsg` to `internal/tui/messages/navigation.go`
- [ ] Update App router in `app.go` to handle `NavigateToHelpMsg`
- [ ] Change `ActionNavigateHelp` handling from ignored to creating navigation message
- [ ] Add help view to the navigation stack properly

### Phase 3: Remove Dashboard Overlay
- [ ] Remove `showDocs` field from DashboardView struct
- [ ] Remove `docsCurrentPage` and `docsSelectedRow` fields
- [ ] Remove help handling from dashboard's Update() method (line 516, 562-564, 830)
- [ ] Delete `internal/tui/views/dash_help_overlay.go`
- [ ] Remove special '?' key handling from dashboard

### Phase 4: Testing & Validation
- [ ] Test navigation flow: Dashboard → Help → Back to Dashboard
- [ ] Test navigation flow: Other View → Help → Back to Original View
- [ ] Test all navigation keys work properly in help (q, b, h, ESC)
- [ ] Test scrolling and copy functionality still works
- [ ] Test terminal resize while in help view
- [ ] Verify status line shows [HELP] correctly

### Phase 5: Documentation & Cleanup
- [ ] Add unit tests for HelpView
- [ ] Add integration tests for help navigation
- [ ] Update `docs/tui-guide.md` to list Help as a proper view
- [ ] Update any references to help overlay in documentation
- [ ] Run linting and formatting

## Benefits of This Refactor

1. **Consistent Navigation**: Help behaves like all other views
2. **Proper History Stack**: Can navigate back correctly
3. **Clean Architecture**: Follows established patterns
4. **Better Maintainability**: Single responsibility for each view
5. **Improved UX**: Users get expected navigation behavior
6. **Testability**: Can test help view in isolation

## Testing Requirements

### Unit Tests
- Test HelpView initialization
- Test key handling (navigation keys)
- Test Update/View cycles
- Test WindowLayout integration

### Integration Tests
- Test navigation to/from help from various views
- Test history stack with help navigation
- Test keymap behavior in help context
- Test terminal resize while in help view

## Migration Notes

- This is a breaking change in internal architecture but not in user-facing behavior
- The help content itself doesn't change, just how it's displayed
- Existing help component (`components/help_view.go`) can be reused
- Status line should show [HELP] consistently

## Related Documentation

- See `docs/tui-guide.md` for TUI implementation patterns
- See `docs/keymap-architecture.md` for key handling details
- See `CLAUDE.md` section on "TUI Implementation Patterns" for navigation architecture

## Timeline Estimate

- Implementation: 2-3 hours
- Testing: 1-2 hours  
- Documentation updates: 30 minutes
- Total: ~4-5 hours

## Risk Assessment

- **Low Risk**: Changes are isolated to TUI layer
- **No API Changes**: Backend unaffected
- **Backward Compatible**: User experience improves but doesn't break
- **Rollback Plan**: Can revert to overlay approach if issues arise