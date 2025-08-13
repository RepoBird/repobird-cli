# Unified Clipboard/Yank System Implementation Plan

## Problem Statement

Currently, the TUI has inconsistent yank/clipboard blink animations across different views:

- **Dashboard**: Single blink with 150ms sleep + 250ms duration check - inconsistent timing
- **Details View**: Uses 100ms tick + 250ms duration - doesn't really blink properly  
- **Status View**: Blinks 4-6 times (too much) with 100ms tick intervals
- **Create View**: Similar to Details but different timing patterns
- **Help Component**: Has its own clipboard implementation

**User Requirement**: ONE consistent blink - quick like 200ms - across all views.

## Current Issues

1. **Inconsistent Timing**: Different views use different blink durations (150ms, 250ms, 600ms)
2. **Multiple Blinks**: Status view blinks 3+ times instead of single quick flash
3. **Code Duplication**: Each view implements its own clipboard/yank logic
4. **Different Message Types**: `yankBlinkMsg`, `statusYankBlinkMsg`, etc.
5. **Maintenance Burden**: Changes require updating multiple files

## Solution: Centralized ClipboardManager Component

### Architecture Design

Create a reusable `ClipboardManager` component in `internal/tui/components/` that:

1. **Provides consistent 200ms single blink behavior**
2. **Handles clipboard operations with unified error handling**
3. **Follows established component patterns** (like WindowLayout, ScrollableList)
4. **Uses standard Bubble Tea message flow**
5. **Embeds in view structs** (no global state)

### Component Interface

```go
type ClipboardManager struct {
    isBlinking     bool
    blinkStartTime time.Time
}

type ClipboardBlinkMsg struct{}

// Core methods
func NewClipboardManager() ClipboardManager
func (c *ClipboardManager) CopyWithBlink(text, description string) (tea.Cmd, error)
func (c *ClipboardManager) Update(msg tea.Msg) (ClipboardManager, tea.Cmd)
func (c *ClipboardManager) IsBlinking() bool
func (c *ClipboardManager) ShouldHighlight() bool // true during 200ms window
```

### Integration Pattern

Views will embed the ClipboardManager and delegate clipboard operations:

```go
type DashboardView struct {
    // ... existing fields
    clipboardManager components.ClipboardManager
}

// Usage in yank operations
func (d *DashboardView) handleYankOperation(text string) tea.Cmd {
    cmd, err := d.clipboardManager.CopyWithBlink(text, "run ID")
    if err != nil {
        // Handle error with status message
    }
    return cmd
}

// In Update method
case components.ClipboardBlinkMsg:
    d.clipboardManager, cmd := d.clipboardManager.Update(msg)
    return d, cmd

// In rendering
if d.clipboardManager.ShouldHighlight() {
    // Apply highlight style for 200ms
}
```

## Implementation Plan

### Phase 1: Core Component Creation

**Files to Create:**
- `internal/tui/components/clipboard_manager.go`
- `internal/tui/components/clipboard_manager_test.go`

**Key Features:**
- 200ms blink timing (single flash)
- Standard message handling with `ClipboardBlinkMsg`
- Error handling for clipboard failures
- Thread-safe state management

### Phase 2: View Migration (One at a time)

**Dashboard View** (`internal/tui/views/dashboard.go` + `dash_clipboard.go`)
- Replace `yankBlink` + `yankBlinkTime` fields with `clipboardManager`
- Update `dash_clipboard.go` methods to use ClipboardManager
- Remove `yankBlinkMsg` handling
- Update rendering logic in `dash_rendering.go`

**Details View** (`internal/tui/views/details.go` + `details_clipboard.go`)
- Replace `yankBlink` + `yankBlinkTime` fields with `clipboardManager`
- Update `details_clipboard.go` methods to use ClipboardManager
- Update rendering logic in `details_rendering.go`

**Status View** (`internal/tui/views/status.go`)
- Replace `yankBlinking` + `yankBlinkCount` fields with `clipboardManager`
- Remove `statusYankBlinkMsg` and `handleYankBlink()` method
- Update clipboard operations to use single blink instead of multiple

**Create View** (`internal/tui/views/create.go.current`)
- Replace `yankBlink` + `yankBlinkTime` fields with `clipboardManager`
- Remove `startYankBlinkAnimation()` method

**Help Component** (`internal/tui/components/help_view.go`)
- Replace `yankBlink` + `yankBlinkTime` fields with `clipboardManager`
- Update clipboard operations

### Phase 3: Cleanup and Testing

**Remove Old Code:**
- Remove all `yankBlinkMsg`, `statusYankBlinkMsg` message types
- Remove all `yankBlink*` fields from view structs
- Remove duplicate `startYankBlinkAnimation()` methods
- Clean up imports and unused code

**Update Tests:**
- Migrate existing clipboard tests to use ClipboardManager
- Add comprehensive component tests
- Verify 200ms timing across all views
- Test error handling scenarios

## Files Affected

### New Files
- `internal/tui/components/clipboard_manager.go`
- `internal/tui/components/clipboard_manager_test.go`

### Modified Files
- `internal/tui/views/dashboard.go`
- `internal/tui/views/dash_clipboard.go`
- `internal/tui/views/dash_rendering.go`
- `internal/tui/views/details.go`
- `internal/tui/views/details_clipboard.go`
- `internal/tui/views/details_rendering.go`
- `internal/tui/views/status.go`
- `internal/tui/views/create.go.current`
- `internal/tui/components/help_view.go`
- `internal/tui/views/dash_messages.go` (remove yankBlinkMsg)

### Test Files to Update
- `internal/tui/views/dashboard_yank_test.go`
- `internal/tui/views/dashboard_yank_truncate_test.go`
- `internal/tui/views/details_cursor_test.go`
- All view tests that involve clipboard operations

## Benefits

1. **Consistent User Experience**: Single 200ms blink across all views
2. **Code Reuse**: Eliminate duplicate clipboard handling code
3. **Easier Maintenance**: Single source of truth for clipboard behavior
4. **Better Testing**: Centralized component with comprehensive tests
5. **Future Extensibility**: Easy to add new clipboard features

## Risk Mitigation

1. **Phased Implementation**: Migrate one view at a time to minimize breakage
2. **Comprehensive Testing**: Test each phase before moving to the next
3. **Backward Compatibility**: Maintain existing key bindings and behavior
4. **Rollback Plan**: Each phase can be reverted independently if issues arise

## Success Criteria

1. All views show consistent 200ms single blink on clipboard operations
2. No code duplication for clipboard/yank functionality
3. All existing tests pass with new implementation
4. User experience remains the same except for consistent timing
5. Code is more maintainable and extensible

## Timeline

- **Phase 1**: 2-3 hours (component creation + tests)
- **Phase 2**: 4-6 hours (view migration, 1-2 views per hour)
- **Phase 3**: 2-3 hours (cleanup + comprehensive testing)

**Total Estimated Time**: 8-12 hours