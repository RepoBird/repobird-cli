# Dashboard Layouts Guide

## Overview

Miller Columns layout system for efficient hierarchical navigation in RepoBird TUI.

## Related Documentation
- **[TUI Guide](TUI-GUIDE.md)** - Complete TUI documentation
- **[Keymap Architecture](KEYMAP-ARCHITECTURE.md)** - Key handling
- **[Architecture Overview](ARCHITECTURE.md)** - TUI layer design

## Miller Columns Design

Inspired by macOS Finder and Ranger file manager, providing hierarchical navigation:

```
┌─ RepoBird Dashboard ─────────────────────────────────────────┐
│ Repositories (12) │ Runs (47)        │ Details             │
├──────────────────┼──────────────────┼─────────────────────┤
│ > myorg/frontend │ ✓ Fix auth bug   │ Run ID: run-123     │
│   myorg/backend  │ ⚡ Add logging    │ Status: Completed   │
│   myorg/mobile   │ ✗ Update deps    │ Branch: fix/auth    │
│   team/service   │ ○ Refactor API   │ PR: #456           │
└──────────────────┴──────────────────┴─────────────────────┘
```

## Column Structure

### 1. Repositories Column (Left)
**Purpose:** Repository selection and overview

**Content:**
- Repository name (org/repo)
- Run count indicator
- Status summary icon
- Active selection highlight

**Behavior:**
- Arrow keys navigate
- Enter selects and moves to Runs
- Tab cycles to next column

### 2. Runs Column (Middle)
**Purpose:** Run list for selected repository

**Content:**
- Run title
- Status icon (✓ done, ⚡ running, ✗ failed, ○ pending)
- Timestamp
- Branch name

**Behavior:**
- Filters by selected repository
- Enter shows details
- Space toggles selection

### 3. Details Column (Right)
**Purpose:** Full run information

**Content:**
- Complete run metadata
- Prompt and context
- Error messages
- PR links
- Logs (scrollable)

**Behavior:**
- Scrollable viewport
- Copy to clipboard (y)
- Open PR in browser (o)

## Navigation Flow

### Forward Navigation
```
Repository → Tab/Enter → Runs → Tab/Enter → Details
          ↓                   ↓                    ↓
    Select repo         Select run          View details
```

### Backward Navigation
```
Details → Shift+Tab/h → Runs → Shift+Tab/h → Repository
       ↓                     ↓                        ↓
   Back to list        Back to repos          (or quit)
```

## Layout Modes

### Standard (Default)
Equal column widths optimized for 80+ character terminals:
```
Width distribution: 30% | 35% | 35%
Min terminal width: 80 chars
```

### Compact
For narrow terminals (60-79 chars):
```
Width distribution: 25% | 35% | 40%
Shows abbreviated content
```

### Wide
For wide terminals (120+ chars):
```
Width distribution: 25% | 30% | 45%
More detail in rightmost column
```

## Implementation Details

### File Organization
Dashboard split across focused files:
- `dashboard.go` - Core Update/View/Init (838 lines)
- `dash_navigation.go` - Key handling (454 lines)  
- `dash_rendering.go` - Layout rendering (584 lines)
- `dash_state.go` - State management (263 lines)
- `dash_data.go` - Data loading (461 lines)

### Rendering Pipeline
1. Calculate column widths based on terminal size
2. Apply borders and padding
3. Render column headers
4. Populate content with proper truncation
5. Apply syntax highlighting and icons
6. Compose final view

### Responsive Design
```go
func calculateColumnWidths(totalWidth int) (repo, runs, details int) {
    if totalWidth < 80 {
        // Compact mode
        repo = totalWidth * 25 / 100
        runs = totalWidth * 35 / 100
        details = totalWidth - repo - runs - 4 // borders
    } else if totalWidth >= 120 {
        // Wide mode
        repo = totalWidth * 25 / 100
        runs = totalWidth * 30 / 100
        details = totalWidth - repo - runs - 4
    } else {
        // Standard mode
        repo = totalWidth * 30 / 100
        runs = totalWidth * 35 / 100
        details = totalWidth - repo - runs - 4
    }
    return
}
```

## Status Indicators

### Repository Status
- `●` Active runs
- `✓` All completed
- `✗` Has failures
- `○` No runs

### Run Status  
- `✓` Completed successfully
- `⚡` Running
- `✗` Failed
- `○` Pending
- `⊘` Cancelled
- `⏸` Paused

### Visual Hierarchy
```
Selected:  ▶ item   (arrow indicator)
Focused:   [item]   (brackets)
Active:    *item*   (asterisks)
```

## Performance Optimization

### Efficient Rendering
- Only re-render changed columns
- Cache formatted strings
- Lazy load details on selection
- Virtual scrolling for long lists

### Data Management
- Batch API calls
- Cache run data
- Progressive loading
- Background refresh

## Keyboard Shortcuts

### Navigation
- `Tab` - Next column
- `Shift+Tab` - Previous column
- `h/l` - Left/right (vim style)
- `j/k` - Up/down in column
- `g/G` - Top/bottom of list

### Actions
- `Enter` - Select/expand
- `Space` - Toggle selection
- `f` - Filter/search (FZF)
- `r` - Refresh data
- `n` - New run
- `d` - Delete run
- `y` - Copy to clipboard

### View Control
- `1/2/3` - Focus column directly
- `/` - Search in column
- `s` - Sort options
- `v` - Toggle details

## Customization

### Theme Support
```yaml
tui:
  theme: dark
  colors:
    selected: blue
    success: green
    error: red
    warning: yellow
```

### Column Preferences
```yaml
dashboard:
  columns:
    repository_width: 30
    runs_width: 35
    details_width: 35
  show_icons: true
  truncate_style: ellipsis
```

## Accessibility

### Screen Reader Support
- Semantic column headers
- ARIA-like annotations
- Keyboard-only navigation
- High contrast mode

### Terminal Compatibility
- Fallback for limited color terminals
- ASCII-only mode
- Works in tmux/screen
- SSH-friendly

## Best Practices

1. **Start in Repository Column** - Natural left-to-right flow
2. **Highlight Active Column** - Clear focus indication
3. **Preserve Selection** - Remember position when navigating back
4. **Progressive Disclosure** - Details load on demand
5. **Consistent Icons** - Same status symbols throughout

## Testing

### Layout Testing
```go
func TestDashboardLayout(t *testing.T) {
    tests := []struct {
        width    int
        expected layoutMode
    }{
        {60, CompactMode},
        {80, StandardMode},
        {120, WideMode},
    }
    // ...
}
```

### Navigation Testing
```go
func TestColumnNavigation(t *testing.T) {
    dashboard := NewDashboard()
    
    // Test forward navigation
    dashboard.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
    assert.Equal(t, RunsColumn, dashboard.activeColumn)
    
    // Test backward navigation  
    dashboard.HandleKey(tea.KeyMsg{Type: tea.KeyShiftTab})
    assert.Equal(t, RepoColumn, dashboard.activeColumn)
}
```