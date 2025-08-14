# Navigation Refactor - Clarified Architecture

## Current Reality Check

You're absolutely right - the confusion stems from mixing different types of "views":

### Three Types of UI Elements

1. **Full Views** (Navigate via App Router)
   - Dashboard → Details → (can navigate back)
   - Dashboard → Create → (can navigate back)  
   - Dashboard → Status → (can navigate back)
   - Dashboard → List → (can navigate back)
   - Dashboard → Bulk → (can navigate back)
   - All show `[VIEW_NAME]` in status line

2. **Overlays** (Toggle on Dashboard only)
   - Help overlay (`?` to open, `?` or `esc` to close)
   - Status info overlay (`s` to open, `s` or `esc` to close)
   - These render OVER the dashboard, not as separate views

3. **Modal States** (Within a view)
   - FZF mode (filter overlay within current view)
   - Insert mode (in Create view)
   - These modify the current view's behavior

## The Real Problem

Currently we have **TWO different "Status" features** which is confusing:
- **Status View** (`StatusView`) - Full view showing API status, accessed via navigation
- **Status Info Overlay** (`showStatusInfo`) - Dashboard-only overlay showing user info

This is why `[STATUS]` appears in my spec - it's actually the overlay, not the Status view!

## What Actually Needs to Change

### Current Behavior (Mostly Good)
- ✅ `q` already goes back from most child views
- ✅ `ESC` already works for overlays and modals
- ✅ `?` already toggles help overlay
- ✅ Status lines already show view names

### Actual Changes Needed

1. **Add `h` for back navigation**
   - Currently: `q`, `b`, `ESC`, `backspace` all do back (confusing)
   - Proposed: `h` = primary back key (vim/ranger style)
   - Keep `q` as "quit to dashboard" from child views

2. **Fix the Status confusion**
   - Rename dashboard's status overlay to avoid confusion
   - Option A: Call it "Info" overlay (`i` key)
   - Option B: Keep `s` but call it `[INFO]` in status line
   - The actual Status View should show `[STATUS]` when navigated to

3. **Make dashboard `q` quit the app**
   - Currently: `q` goes back (but there's nowhere to go)
   - Proposed: `q` quits app from dashboard (with confirmation)

4. **Remove redundant back keys**
   - Remove `b` for back (keep it for Bulk on dashboard only)
   - Remove `backspace` for back (typing only)
   - Keep `ESC` for modal/overlay cancel only

## Minimal Implementation Plan

### Phase 1: Core Changes
```go
// internal/tui/keymap/core.go
registry.Register("h", ActionNavigateBack, "go back")
registry.Register("q", ActionNavigateToDashboard, "quit to dashboard") 
// Remove: registry.Register("b", ActionNavigateBack, ...)
// Remove: registry.Register("backspace", ActionNavigateBack, ...)
// Keep ESC for modal cancellation only
```

### Phase 2: View-Specific Updates

#### Dashboard
- Keep `h` for column navigation (moves between columns left)
- Keep `l` for column navigation (moves between columns right)
- Make `q` quit app (not navigate back)
- `s` navigates to Status View (full view)

#### All Child Views (Details, Create, Status, List, Bulk)
- Enable `h` → NavigateBackMsg
- Keep `q` → NavigateToDashboardMsg (not back to parent)
- Update status line help text

### Phase 3: Status Line Updates

| View | Current | Proposed |
|------|---------|----------|
| Dashboard | `[q]back` | `[q]quit` |
| Details | `[q/ESC/b]back` | `[h]back [q]dashboard` |
| Create | `[ESC]cancel [b]back` | `[h]back [q]dashboard` |
| Status | `[q/ESC/b]back` | `[h]back [q]dashboard` |
| List | `[b]back` | `[h]back [q]dashboard` |
| Bulk | `[ESC]back [q]quit` | `[h]back [q]dashboard` |

## Summary of Real Changes

### What's Actually Changing:
1. **Add `h` as primary back navigation** (new vim/ranger pattern)
2. **Remove `b` and `backspace` for back** (reduce confusion)
3. **Dashboard `q` quits app** (not back)
4. **Child view `q` goes to dashboard** (not parent)
5. **Clarify Status vs Info overlay naming**

### What's NOT Changing:
- View architecture (still full views via router)
- Status line format (already shows `[VIEW_NAME]`)
- ESC behavior (already cancels modals/overlays)
- Toggle overlays (already work with same key)
- FZF mode (already works fine)

## Benefits of This Simplified Approach

1. **Minimal Breaking Changes** - Most keys work the same
2. **Clear Mental Model** - `h` back, `l` forward (vim/ranger)
3. **No Accidental Exits** - Can't quit with double `q` from child views
4. **Less Confusion** - Fewer keys that do the same thing
5. **Familiar Pattern** - Vim users already know h/j/k/l

## Implementation Effort

This is actually a much smaller change than the original spec suggested:
- ~10 lines in keymap/core.go
- Update HandleKey in 5-6 views
- Update status line text in same views
- No architectural changes needed

## Existing Infrastructure (Already In Place)

### 1. CoreViewKeymap Interface (`internal/tui/keymap/core.go`)
The codebase already has a robust key customization system:

```go
type CoreViewKeymap interface {
    // Disable specific keys for a view
    IsKeyDisabled(keyString string) bool
    
    // Override default key behavior
    HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd)
}
```

**Current Implementations:**
- **Dashboard**: Disables back keys (`b`, `esc`), overrides `b` for bulk navigation
- **Bulk**: Custom ESC handling for file browser mode transitions
- **Create**: Mode-aware key handling (insert vs normal mode)
- **Other views**: Can implement interface as needed

### 2. Centralized Key Processing
All keys flow through `App.processKeyWithFiltering()` which:
1. Checks if view disabled the key via `IsKeyDisabled()`
2. Lets view handle via `HandleKey()` if implemented
3. Routes to global actions (force quit)
4. Converts navigation actions to messages
5. Delegates view-specific keys to view's `Update()`

### 3. CoreKeyRegistry
Central registry maintains all default key mappings:
```go
registry.Register("b", ActionNavigateBack, "go back")
registry.Register("q", ActionNavigateBack, "go back") 
registry.Register("esc", ActionNavigateBack, "go back")
registry.Register("backspace", ActionNavigateBack, "go back")
// etc...
```

### 4. Status Line System
Each view has its own `renderStatusLine()` method:

**Examples:**
- **Dashboard**: `func (d *DashboardView) renderStatusLine(layoutName string)`
- **Create**: Shows `[CREATE]` or `[CREATE] [INPUT]` based on mode
- **Bulk**: Dynamic help text based on current mode
- **Status**: `[STATUS]` with `[j/k]navigate [h/l]columns [y]copy`
- **Error**: `[ERROR]` with recovery-specific help

**Status Line Pattern:**
```go
func (v *ViewName) renderStatusLine() string {
    helpText := "[h]back [q]dashboard [r]refresh"
    return fmt.Sprintf("[%s] %s", viewName, helpText)
}
```

## What This Means for Implementation

The refactor only needs to:

### 1. Update CoreKeyRegistry (internal/tui/keymap/core.go)
```go
// Change these lines:
registry.Register("h", ActionNavigateBack, "go back")  // NEW
registry.Register("q", ActionNavigateToDashboard, "dashboard")  // CHANGED
// registry.Register("b", ActionNavigateBack, "go back")  // REMOVE
// registry.Register("backspace", ActionNavigateBack, "go back")  // REMOVE
```

### 2. Update View HandleKey Methods
Only views that need special behavior:
- **Dashboard**: Make `q` quit app instead of back
- **Dashboard**: Keep `h` for left column navigation (already works)
- **Create/Bulk**: Already handle modes correctly

### 3. Update Status Line Help Text
Simple string changes in each view's `renderStatusLine()`:
- Change `[q/ESC/b]back` to `[h]back [q]dashboard`
- Dashboard: Change `[q]back` to `[q]quit`

### 4. No New Infrastructure Needed
- ✅ Key customization system already exists
- ✅ Status line system already per-view
- ✅ Central key processing already works
- ✅ Navigation messages already routed

The architecture is already perfectly set up for this change!