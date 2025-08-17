# Terminal User Interface (TUI) Guide

## Overview

Rich terminal interface for managing AI runs with Bubble Tea framework and message-based navigation.

## Related Documentation
- **[Architecture Overview](ARCHITECTURE.md)** - TUI layer design
- **[Keymap Architecture](KEYMAP-ARCHITECTURE.md)** - Key handling system
- **[Dashboard Layouts](DASHBOARD-LAYOUTS.md)** - Miller columns implementation
- **[Troubleshooting Guide](TROUBLESHOOTING.md)** - Debug logging

## Quick Start

```bash
repobird tui
```

## Dashboard Layout

**Miller Columns** (3-column navigation):
1. **Repositories** (left) - All repos with active runs
2. **Runs** (middle) - Runs for selected repository  
3. **Details** (right) - Full run information

## Navigation

### Core Keys
- **Tab** - Cycle columns forward
- **↑↓/j/k** - Move up/down
- **←→/h/l** - Move between columns
- **Enter** - Select and advance
- **f** - Fuzzy search (FZF mode)
- **n** - New run
- **r** - Refresh
- **s** - Status overlay
- **q** - Back/quit
- **?** - Help

### Navigation Flow
```
Dashboard → Create/Details/List → Back to Dashboard
         ↓                      ↑
      (n key)              (q/ESC/b keys)
```

## Architecture

### Message-Based Navigation
Views emit navigation messages handled by central app router:
- `NavigateToCreateMsg` - Open create view
- `NavigateToDetailsMsg` - Open details view
- `NavigateBackMsg` - Go to previous view
- `NavigateToDashboardMsg` - Return to dashboard

### View Pattern
```go
// Minimal constructor (max 3 params)
NewView(client, cache, id)

// Self-loading in Init()
func (v *View) Init() tea.Cmd {
    return v.loadData()
}
```

### Shared Components
- **ScrollableList** - Multi-column lists (`internal/tui/components/scrollable_list.go`)
- **Form** - Input forms with validation (`internal/tui/components/form.go`)
- **CustomCreateForm** - Advanced form with vim-style modal editing (`internal/tui/views/create_custom_form.go`)
- **WindowLayout** - Global sizing system (`internal/tui/components/window_layout.go`)
- **FZFSelector** - Fuzzy search overlay (`internal/tui/components/fzf_selector.go`)

## Views

### Dashboard View
**Files:** Split across 11 files for maintainability
- `dashboard.go` - Core Update/View/Init
- `dash_navigation.go` - Key handling
- `dash_rendering.go` - Layout rendering
- `dash_data.go` - Data loading and cache
- `dash_fzf.go` - FZF integration

**Features:**
- Real-time status updates
- Repository filtering
- Run status color coding
- Clipboard operations (y/Y keys)

### Create Run View
**Form Fields:**
- Title, Repository, Source/Target Branch
- Issue number (optional)
- Prompt and Context

**Modes:**
- Insert mode (default) - Text input
- Normal mode (ESC) - Navigation
- FZF mode (Ctrl+F) - Repository selection

### Details View
**Features:**
- Full run information display
- Scrollable content
- PR URL and status
- Copy to clipboard support

### Bulk View
**Modes:**
1. File selection - Choose config file
2. Run list - Review and toggle runs
3. Progress - Real-time submission tracking
4. Results - Summary with success/failure

## Caching

**Hybrid Cache Architecture:**
- Terminal runs persisted to disk
- Active runs cached in memory (5min TTL)
- User-isolated storage
- 90% API call reduction

**Location:** `~/.config/repobird/cache/users/{hash}/`

## Key Handling

### Centralized Keymap System
All keys processed through `App.processKeyWithFiltering()`:

1. Check if key disabled for view
2. Try custom handler
3. Process global actions (quit)
4. Convert to navigation messages
5. Delegate to view Update()

### View Customization
```go
// Disable keys
func (v *View) IsKeyDisabled(key string) bool

// Custom handling
func (v *View) HandleKey(msg tea.KeyMsg) (handled bool, model, cmd)
```

## FZF Integration

### Activation
- Press **f** on any dashboard column
- **Ctrl+F** in create view (insert mode)

### Features
- Real-time fuzzy filtering
- Dropdown overlay at cursor
- Repository/run/detail filtering
- Smart icon indicators

## WindowLayout System

Ensures consistent sizing across views:

```go
layout := components.NewWindowLayout(width, height)
boxW, boxH := layout.GetBoxDimensions()
viewportW, viewportH := layout.GetViewportDimensions()
```

**Critical:** Initialize layout only on WindowSizeMsg, not in constructor.

## Status Line

Unified status bar showing:
- Loading state
- Data counts
- Last refresh time
- Error messages
- Help text

## Debug Mode

```bash
REPOBIRD_DEBUG_LOG=1 repobird tui
tail -f /tmp/repobird_debug.log
```

## Performance

- Loads 1000+ runs efficiently
- Batch cache updates
- Lazy loading for large datasets
- Minimal API calls via caching

## Best Practices

1. Use navigation messages, never create views directly
2. Implement WindowLayout for consistent borders
3. Use shared components for common UI elements
4. Keep views self-loading via Init()
5. Test with `XDG_CONFIG_HOME` for isolation