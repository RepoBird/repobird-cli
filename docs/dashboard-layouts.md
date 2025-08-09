# Dashboard Layout Design Options

## Overview

This document outlines various layout options for the RepoBird CLI dashboard that will display both issue runs and repositories. The dashboard will support dynamic layout switching via keyboard shortcuts (Shift+L) to provide maximum flexibility for different user preferences and screen configurations.

## Top Bar Design

The top bar will display essential information across all layout modes:

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
│                                                                                   │
```

**Top Bar Elements:**
- **Left**: Application title and current view
- **Center**: Status indicators (connection, current operation)  
- **Right**: Runs remaining counter with plan type (Free: 10/10, Pro: 47/100, Enterprise: Unlimited)
- **Far Right**: Layout switch hint

## Layout Options

### 1. Vertical Split Layout (Default)
**Best for**: Side-by-side comparison, wide screens, detailed information

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
├─ Issue Runs ─────────────┬─ Repositories ────────────────────────────────────────┤
│ ● #1234 Fix auth bug     │ 📁 acme-corp/backend                                  │
│   Status: Running        │    12 runs • Last: 2h ago                            │
│   Branch: fix/auth       │                                                       │
│   Started: 5m ago        │ 📁 acme-corp/frontend                                 │
│                          │    8 runs • Last: 1d ago                             │
│ ● #1235 Add feature X    │                                                       │
│   Status: Completed      │ 📁 acme-corp/mobile-app                              │
│   Branch: feat/x         │    3 runs • Last: 3d ago                             │
│   Completed: 1h ago      │                                                       │
│                          │ 📁 acme-corp/docs                                     │
│ ● #1236 Bug fix Y        │    1 run • Last: 1w ago                              │
│   Status: Failed         │                                                       │
│   Branch: fix/y          │                                                       │
│   Failed: 30m ago        │                                                       │
└──────────────────────────┴───────────────────────────────────────────────────────┘
```

**Key Features:**
- Equal width panes (50/50 split)
- Resize with Ctrl+Left/Right arrows
- Focus switching with Tab/Shift+Tab
- Clear visual separation

### 2. Horizontal Split Layout
**Best for**: Narrow screens, detailed run logs, stacked information

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
├─ Issue Runs ──────────────────────────────────────────────────────────────────────┤
│ ● #1234 Fix auth bug        ● #1235 Add feature X      ● #1236 Bug fix Y         │
│   Running • fix/auth • 5m     Completed • feat/x • 1h    Failed • fix/y • 30m    │
├─ Repositories ────────────────────────────────────────────────────────────────────┤
│ 📁 acme-corp/backend (12 runs, 2h ago)  📁 acme-corp/frontend (8 runs, 1d ago)   │
│ 📁 acme-corp/mobile-app (3 runs, 3d ago)  📁 acme-corp/docs (1 run, 1w ago)      │
└────────────────────────────────────────────────────────────────────────────────────┘
```

**Key Features:**
- Top pane: Issue runs (compact horizontal cards)
- Bottom pane: Repositories (grid layout)
- Resize with Ctrl+Up/Down arrows
- Better for showing more items horizontally

### 3. Tabbed Interface Layout
**Best for**: Clean interface, single focus, context switching

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
├─ [Issue Runs] [Repositories] [All Items] ─────────────────────────────────────────┤
│                                                                                   │
│ ● #1234 Fix authentication bug in login flow                                     │
│   Status: Running         Branch: fix/auth-bug        Started: 5 minutes ago    │
│   Repository: acme-corp/backend                                                  │
│                                                                                   │
│ ● #1235 Add user dashboard feature                                               │
│   Status: Completed       Branch: feat/dashboard      Completed: 1 hour ago     │
│   Repository: acme-corp/frontend                                                 │
│                                                                                   │
│ ● #1236 Fix mobile app crash on startup                                          │
│   Status: Failed          Branch: fix/startup-crash   Failed: 30 minutes ago    │
│   Repository: acme-corp/mobile-app                                               │
│                                                                                   │
│                                                                                   │
└────────────────────────────────────────────────────────────────────────────────────┘
```

**Key Features:**
- Single pane with tabs
- Switch tabs with Ctrl+Tab or 1/2/3 number keys
- More detailed information per item
- Clean, uncluttered interface

### 4. Master-Detail Layout
**Best for**: Detailed inspection, focused workflows

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
├─ Items ──────────────────┬─ Details ──────────────────────────────────────────────┤
│ ● #1234 Fix auth bug     │ Run Details: #1234                                    │
│ ● #1235 Add feature X    │ Title: Fix authentication bug in login flow           │
│ ● #1236 Bug fix Y        │ Repository: acme-corp/backend                          │
│ 📁 acme-corp/backend     │ Branch: fix/auth-bug → fix/auth-bug-resolved          │
│ 📁 acme-corp/frontend    │ Status: Running (Step 3/7)                            │
│ 📁 acme-corp/mobile-app  │ Started: 2024-08-09 14:23:15                          │
│ 📁 acme-corp/docs        │ Estimated completion: ~8 minutes                      │
│                          │                                                        │
│                          │ Progress:                                              │
│                          │ ✅ Analyze codebase                                   │
│                          │ ✅ Identify bug location                              │
│                          │ ⏳ Generate fix                                       │
│                          │ ⏳ Apply changes                                      │
│                          │ ⚪ Run tests                                          │
│                          │ ⚪ Create pull request                                │
│                          │ ⚪ Notify completion                                  │
└──────────────────────────┴────────────────────────────────────────────────────────┘
```

**Key Features:**
- Mixed list on left (runs + repos)
- Detailed view on right shows selected item
- Perfect for inspecting individual runs or repo stats
- Arrow keys navigate list, Enter shows details

### 5. Grid/Card Layout
**Best for**: Visual overview, quick status scanning

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
├─ Issue Runs ──────────────────────────────────────────────────────────────────────┤
│ ┌─ #1234 ──────────────┐ ┌─ #1235 ──────────────┐ ┌─ #1236 ──────────────┐      │
│ │ Fix auth bug         │ │ Add feature X        │ │ Bug fix Y            │      │
│ │ 🔄 Running           │ │ ✅ Completed         │ │ ❌ Failed            │      │
│ │ fix/auth • 5m ago    │ │ feat/x • 1h ago      │ │ fix/y • 30m ago      │      │
│ └──────────────────────┘ └──────────────────────┘ └──────────────────────┘      │
├─ Repositories ────────────────────────────────────────────────────────────────────┤
│ ┌─ backend ────────────┐ ┌─ frontend ───────────┐ ┌─ mobile-app ─────────┐      │
│ │ 📁 acme-corp/backend │ │ 📁 acme-corp/frontend│ │ 📁 acme-corp/mobile  │      │
│ │ 12 runs • 2h ago     │ │ 8 runs • 1d ago      │ │ 3 runs • 3d ago      │      │
│ │ ⭐ 234 stars         │ │ ⭐ 89 stars          │ │ ⭐ 45 stars          │      │
│ └──────────────────────┘ └──────────────────────┘ └──────────────────────┘      │
└────────────────────────────────────────────────────────────────────────────────────┘
```

**Key Features:**
- Card-based layout for visual appeal
- Color-coded status indicators
- Responsive grid (adjusts to terminal width)
- Good for status overview at a glance

### 6. Tree/Hierarchical Layout
**Best for**: Organized navigation, repository-centric view

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
│ 📁 acme-corp/backend (12 runs)                                                   │
│ ├── ● #1234 Fix auth bug (Running, 5m ago)                                       │
│ ├── ● #1230 Add logging (Completed, 2h ago)                                      │
│ └── ● #1228 Database fix (Failed, 1d ago)                                        │
│                                                                                   │
│ 📁 acme-corp/frontend (8 runs)                                                   │
│ ├── ● #1235 Add feature X (Completed, 1h ago)                                    │
│ ├── ● #1232 UI improvements (Running, 45m ago)                                   │
│ └── ● #1229 Responsive design (Completed, 2d ago)                                │
│                                                                                   │
│ 📁 acme-corp/mobile-app (3 runs)                                                 │
│ ├── ● #1236 Bug fix Y (Failed, 30m ago)                                          │
│ └── ● #1231 Performance opt (Completed, 3d ago)                                  │
│                                                                                   │
│ 📁 acme-corp/docs (1 run)                                                        │
│ └── ● #1233 Update API docs (Completed, 1w ago)                                  │
└────────────────────────────────────────────────────────────────────────────────────┘
```

**Key Features:**
- Hierarchical tree structure
- Repository-centric organization
- Expandable/collapsible sections
- Arrow keys for navigation
- Space/Enter to expand/collapse

## Layout Switching Mechanism

### Keyboard Shortcuts

| Shortcut | Action | Notes |
|----------|--------|-------|
| `Shift+L` | Cycle through layouts | Primary layout switching |
| `Ctrl+1-6` | Jump to specific layout | Direct layout selection |
| `Ctrl+←/→` | Resize vertical splits | When applicable |
| `Ctrl+↑/↓` | Resize horizontal splits | When applicable |
| `Tab` | Switch pane focus | In multi-pane layouts |
| `Shift+Tab` | Reverse pane focus | In multi-pane layouts |

### Layout Memory
- Remember user's preferred layout per session
- Save layout preference to config file (`~/.repobird/config.yaml`)
- Support per-repository layout preferences

### Transition Animation
- Smooth transitions between layouts (fade/slide)
- Preserve focus and selection when switching
- Visual feedback during layout change

## Implementation Considerations

### Bubble Tea Components
Each layout will be implemented as a separate Bubble Tea model:

```go
type LayoutType int

const (
    LayoutVerticalSplit LayoutType = iota
    LayoutHorizontalSplit
    LayoutTabbed
    LayoutMasterDetail
    LayoutGrid
    LayoutTree
)

type DashboardModel struct {
    currentLayout LayoutType
    layouts       map[LayoutType]tea.Model
    // ... other fields
}
```

### Responsive Design
- Adapt layouts based on terminal dimensions
- Minimum width/height requirements for each layout
- Graceful fallback to simpler layouts on small terminals

### Data Model
```go
type DashboardData struct {
    Runs         []RunItem
    Repositories []RepoItem
    CurrentUser  UserInfo
    Quotas       QuotaInfo
}

type RunItem struct {
    ID          string
    Title       string
    Status      RunStatus
    Repository  string
    Branch      string
    CreatedAt   time.Time
    CompletedAt *time.Time
}

type RepoItem struct {
    Name        string
    RunCount    int
    LastRunTime time.Time
    Stars       int
}
```

### Performance Considerations
- Lazy loading for large lists
- Virtual scrolling for repositories list
- Efficient re-rendering on layout switch
- Debounced resize handling

## Configuration Options

Users can customize layouts via config file:

```yaml
dashboard:
  default_layout: "vertical_split"
  remember_layout: true
  animations: true
  compact_mode: false
  show_icons: true
  
layouts:
  vertical_split:
    pane_ratio: 0.5
    show_borders: true
  
  horizontal_split:
    top_pane_height: 0.6
    
  grid:
    cards_per_row: 3
    card_height: 6
```

## Accessibility Features
- High contrast mode support
- Screen reader friendly text
- Keyboard-only navigation
- Focus indicators
- Color-blind friendly status colors

## Future Enhancements
- Custom layout builder
- Plugin system for additional layouts
- Layout templates for different workflows
- Export/import layout configurations
- Team shared layout preferences

---

This design provides maximum flexibility while maintaining usability and follows established CLI/TUI patterns from popular tools like tmux, vim, and file managers.