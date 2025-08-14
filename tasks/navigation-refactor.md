# RepoBird TUI Navigation Refactor

## Executive Summary
Consolidate and unify navigation behavior across all TUI views to provide an intuitive, consistent experience that follows vim/ranger patterns while preventing accidental exits.

## Current Issues
1. **Inconsistent Back Navigation**: Multiple keys (`b`, `q`, `ESC`, `backspace`) all go back, creating confusion
2. **Accidental Exit Risk**: `q` sometimes quits, sometimes goes back - users can accidentally exit with `q`, `q`
3. **No Hierarchical Navigation**: Can't use `h` to go back after using `l`/`enter` to go forward
4. **Unclear Toggle Behavior**: Status (`s`) and help (`?`) overlays have inconsistent closing behavior
5. **Mixed Metaphors**: Mixing file manager (ranger) and editor (vim) navigation patterns

## Proposed Navigation System

### Core Principles
1. **Vim-style movement**: `h/j/k/l` for left/down/up/right navigation
2. **Ranger-style hierarchy**: `h` goes to parent, `l` or `enter` goes to child
3. **ESC for modal cancel**: Cancel current mode/overlay/input, NOT for navigation
4. **No accidental exits**: Multiple safeguards against unintended quits
5. **Toggle consistency**: Overlays toggle with same key that opened them

### Key Bindings

#### Primary Navigation Keys
| Key | Action | Context | Notes |
|-----|--------|---------|-------|
| `h` | Navigate left/parent | All views | Go back to parent view or move left in columns |
| `l` | Navigate right/child | All views | Enter child view or move right in columns |
| `j` | Move down | Lists/columns | Select next item |
| `k` | Move up | Lists/columns | Select previous item |
| `enter` | Enter/select | All views | Same as `l` - enter child view |
| `ESC` | Cancel mode | Modal contexts | Exit FZF, overlays, insert mode - NOT navigation |

#### Special Navigation
| Key | Action | Context | Notes |
|-----|--------|---------|-------|
| `q` | Quit to dashboard | Child views only | Goes to dashboard, NOT parent |
| `Q` | Force quit app | All views | Capital Q always quits with confirmation |
| `ctrl+c` | Force quit app | All views | Emergency exit |
| `backspace` | Type/delete | Input fields only | No longer navigation |
| `tab` | Next field/column | Forms/columns | Cycle through fields |
| `shift+tab` | Previous field | Forms | Reverse cycle |

#### View-Specific Actions
| Key | Action | Context | Notes |
|-----|--------|---------|-------|
| `n` | New run | Dashboard/lists | Navigate to create view |
| `b` | Bulk operations | Dashboard only | Special dashboard action |
| `B` | Bulk operations | All views | Global bulk navigation |
| `r` | Refresh | All views | Reload current data |
| `f` | Filter/FZF | Lists | Activate fuzzy search |
| `/` | Search | Lists | Alternative search activation |

#### Overlay Toggles
| Key | Action | Context | Notes |
|-----|--------|---------|-------|
| `?` | Toggle help | All views | Press again to close, `h` also goes back |
| `ESC` | Close overlay | Any overlay | Universal overlay close |

### Navigation Hierarchy

```
Dashboard (root)
├─ [h/l: move between columns] [enter: select]
├─ Create Run View
│  └─ [h: back to dashboard]
├─ Run Details View  
│  └─ [h: back to dashboard]
├─ Status View
│  └─ [h: back to dashboard]
├─ Bulk View
│  └─ [h/ESC: back to dashboard based on mode]
└─ List View
   └─ [h: back to dashboard]
```

### View-Specific Behavior

#### Dashboard
- **Root view**: Cannot go back (nowhere to go)
- **Miller columns**: `h/l` move between columns, `j/k` move within column
- **Direct quit**: `q` quits app (with confirmation if unsaved changes)
- **Overlays**: `?` for help (toggle behavior)
- **Status**: `s` navigates to Status View (full view, not overlay)

#### Child Views (Details, Status, Create, List)
- **Back navigation**: `h` returns to dashboard
- **Quit behavior**: `q` returns to dashboard (NOT app quit)
- **Force quit**: `Q` or `ctrl+c` quits app from any view

#### Create View
- **Insert mode**: `h` types character (navigation disabled)
- **Normal mode**: `h` returns to dashboard
- **ESC**: Exit insert mode (vim-style)

#### Bulk View
- **File browser mode**: `ESC` goes back one level, second `ESC` returns to dashboard
- **Run list mode**: `h` returns to file browser
- **Progress mode**: Navigation disabled during execution

#### Overlays (Help, Status Info)
- **Toggle close**: Same key that opened (`?` or `s`)
- **ESC close**: Universal overlay closer
- **No quit**: `q` doesn't close overlay (prevents confusion)

### Implementation Plan

#### Phase 1: Core Keymap Changes
1. Update `internal/tui/keymap/core.go`:
   - Remove `q` from `ActionNavigateBack` 
   - Add `h` as primary `ActionNavigateBack`
   - Keep `ESC` for modal cancellation only
   - Add `ActionNavigateToDashboard` for `q` in child views

2. Update `App.processKeyWithFiltering()`:
   - Handle view context for `q` key (dashboard vs child)
   - Implement confirmation for dashboard quit
   - Route `h` to navigation in non-input contexts

#### Phase 2: View Updates
1. **Dashboard**: 
   - Disable `h` for back navigation
   - Enable `q` for quit with confirmation
   - Implement toggle behavior for overlays

2. **Child Views**:
   - Enable `h` for back to dashboard
   - Change `q` to navigate to dashboard
   - Remove `backspace` navigation

3. **Create View**:
   - Disable `h` navigation in insert mode
   - Enable `h` navigation in normal mode

#### Phase 3: Special Cases
1. **Bulk View**: 
   - Context-aware ESC handling
   - Mode-specific navigation

2. **FZF Mode**:
   - ESC cancels FZF only
   - No navigation side effects

### Migration Guide for Users

#### Breaking Changes
- `q` no longer goes back in child views (goes to dashboard instead)
- `backspace` no longer navigates back (typing only)
- `b` only works on dashboard for bulk operations

#### New Patterns
- Use `h` to go back/left (ranger-style)
- Use `l` or `enter` to go forward/right
- Press overlay key again to close (`?` for help, `s` for status)
- Use `Q` for force quit from anywhere

### Benefits
1. **Consistency**: Same keys work the same way everywhere
2. **Intuitiveness**: Follows established vim/ranger patterns
3. **Safety**: No accidental exits with double `q`
4. **Discoverability**: Clear visual hierarchy with h/l navigation
5. **Efficiency**: Fewer keys to remember, muscle memory from vim

### Testing Requirements
1. Test navigation flow: Dashboard → Child → Dashboard
2. Test overlay toggles open and close properly
3. Test `q` behavior differs between dashboard and children
4. Test `h` navigation in all contexts
5. Test ESC only affects modals, not navigation
6. Test force quit (`Q`, `ctrl+c`) from all views
7. Test bulk view mode transitions

### Documentation Updates
1. Update help overlay with new key bindings
2. Update status line hints per view
3. Update README with navigation guide
4. Update CLAUDE.md with new patterns

## Status Line Text Specifications

### Consistent Format
All status lines follow the pattern:
- **Left**: `[VIEW_NAME]` - Current view indicator  
- **Center**: Help text with key bindings (show `← h` for back on child views)
- **Right**: Data info (if applicable)

### View-Specific Status Lines

#### Dashboard View
```
Normal Mode:
[DASH] [h/l]columns [j/k]navigate [enter]select [n]new [B]bulk [r]refresh [f]filter [?]help [s]status [q]quit

FZF Mode:
[DASH-FZF] [↑↓]navigate [enter]select [esc]cancel

Help Overlay (? pressed):
[HELP] [← h/?/esc]close [j/k]navigate [enter]select
```

#### Create Run View
```
Normal Mode:
[CREATE] [← h]back [tab]next field [enter]edit [ctrl+s]submit [esc]cancel

Insert Mode:
[CREATE-INSERT] [esc]exit insert [tab]next field [ctrl+s]submit [ctrl+c]cancel

FZF Mode (repository selection):
[CREATE-FZF] [↑↓]navigate [enter]select [esc]cancel
```

#### Details View
```
Normal Mode:
[DETAILS] [← h]back [j/k]navigate [y]copy [o]open URL [r]refresh [q]dashboard

Loading:
[DETAILS] Loading run details...

Polling:
[DETAILS] [← h]back [j/k]navigate [y]copy [o]open URL • Auto-refresh: ON
```

#### Status View  
```
[STATUS] [← h]back [j/k]navigate [y]copy [Y]copy all [r]refresh [q]dashboard
```

#### List View
```
[LIST] [← h]back [j/k]navigate [enter]view details [n]new [r]refresh [f]filter [q]dashboard

FZF Mode:
[LIST-FZF] [↑↓]navigate [enter]select [esc]cancel
```

#### Bulk View
```
File Browser Mode:
[BULK] [← h]back [j/k]navigate [enter]select file [esc]cancel [q]dashboard

Run List Mode:
[BULK-RUNS] [← h]back to files [j/k]navigate [space]toggle [enter]submit [q]dashboard

Progress Mode:
[BULK-PROGRESS] Processing... [ctrl+c]cancel

Results Mode:
[BULK-RESULTS] [← h]restart [j/k]navigate [enter]view details [q]dashboard
```

#### Error View
```
[ERROR] [← h]back [q]dashboard [Q]force quit
```

### Key Binding Consistency Rules

1. **Navigation Keys (Always Consistent)**:
   - `h` = Back/left (to parent or dashboard)
   - `l` = Forward/right (to child view)
   - `j` = Down
   - `k` = Up
   - `enter` = Select/enter child

2. **View Actions (Context-Aware)**:
   - `n` = New (where applicable)
   - `r` = Refresh (where data can be refreshed)
   - `f` = Filter/FZF (where lists exist)
   - `y` = Copy (where text can be copied)
   - `o` = Open URL (where URLs exist)

3. **Modal Controls**:
   - `esc` = Cancel current mode/overlay
   - `tab` = Next field (forms)
   - `space` = Toggle selection (checkboxes)

4. **Exit Controls**:
   - `q` = Return to dashboard (from child views)
   - `Q` = Force quit application (all views)
   - `ctrl+c` = Emergency exit (all views)

5. **Overlay Toggles**:
   - `?` = Toggle help (press again to close)
   - `s` = Toggle status (press again to close)
   - `esc` = Close any overlay

### Implementation Checklist

For each view, ensure:
- [ ] Status line shows correct view name in brackets
- [ ] Help text matches actual implemented key bindings
- [ ] Key actions are consistent with global patterns
- [ ] Modal states show different help text
- [ ] Overlay states show how to close
- [ ] Loading states show appropriate message
- [ ] FZF modes show filter-specific controls

### Testing Status Lines

Test each status line by:
1. Verifying view name matches current view
2. Testing each key shown in help text works as described
3. Checking modal transitions update help text
4. Confirming overlay toggles show correct close instructions
5. Ensuring no keys are shown that don't work
6. Validating no working keys are missing from help

## Decision Log

### Why `h` for back instead of `q`?
- **Vim/Ranger convention**: Both use h/l for hierarchy navigation
- **Spatial mapping**: Left/right maps to back/forward
- **Prevents accidents**: Can't quit by mistake when trying to go back

### Why `q` to dashboard in child views?
- **Quick escape**: One key to get back to main view
- **Vim pattern**: `q` quits current mode/view
- **User expectation**: `q` should "quit" something (the current view)

### Why toggle overlays with same key?
- **Intuitive**: Same key opens and closes
- **Common pattern**: Many TUIs use this (htop's F-keys, etc.)
- **Reduces cognitive load**: Don't need to remember different close key

### Why keep ESC for modals only?
- **Vim convention**: ESC exits modes, not navigation
- **Clear purpose**: Always means "cancel current operation"
- **FZF compatibility**: Needed for search cancellation