# Dashboard Layout Design Options

## Overview

This document outlines the optimal layout system for the RepoBird CLI dashboard, inspired by Ranger's proven Miller Columns interface. The primary layout uses a triple-column design that provides hierarchical navigation: Repositories → Issue Runs → Run Details. Additional simplified layouts support different workflows, with seamless switching via Shift+L.

## Top Bar Design

The top bar displays essential information across all layout modes:

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

### 1. Triple-Column Layout (Default) - Ranger-Inspired
**Best for**: Hierarchical navigation, efficient workflow, detailed inspection

Based on Ranger's Miller Columns interface, this layout provides the optimal user experience for navigating repositories, runs, and details.

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
├─ Repositories ───────┬─ Issue Runs ─────────────┬─ Run Details ─────────────────────┤
│ 📁 acme-corp/backend │ ● #1234 Fix auth bug    │ Run #1234: Fix auth bug           │
│    12 runs • 2h ago  │   🔄 Running • 5m ago   │                                   │
│ ► 📁 acme-corp/web   │   fix/auth-bug          │ Repository: acme-corp/backend     │
│    8 runs • 1d ago   │                         │ Branch: fix/auth-bug              │
│ 📁 acme-corp/mobile  │ ● #1230 Add logging     │ Status: Running (Step 3/7)       │
│    3 runs • 3d ago   │   ✅ Completed • 2h ago │ Started: 2024-08-09 14:23:15     │
│ 📁 acme-corp/docs    │   feat/logging          │ Est. completion: ~8 minutes       │
│    1 run • 1w ago    │                         │                                   │
│                      │ ● #1228 Database fix    │ Progress:                         │
│                      │   ❌ Failed • 1d ago    │ ✅ Analyze codebase              │
│                      │   fix/db-conn           │ ✅ Identify bug location         │
│                      │                         │ ⏳ Generate fix                  │
│                      │                         │ ⚪ Apply changes                 │
│                      │                         │ ⚪ Run tests                     │
│                      │                         │ ⚪ Create pull request           │
│                      │                         │                                   │
│                      │                         │ Recent Activity:                  │
│                      │                         │ 14:28 - Analyzing auth module    │
│                      │                         │ 14:25 - Found security issue     │
│                      │                         │ 14:23 - Started code analysis    │
└──────────────────────┴─────────────────────────┴───────────────────────────────────┘
```

**Column Widths:**
- Left (Repositories): 25% - Repository list with icons, run counts, last activity
- Center (Issue Runs): 35% - Runs for selected repository with status and timing
- Right (Run Details): 40% - Detailed information about selected run

**Key Features:**
- **Miller Columns Navigation**: Selection in left column updates center, selection in center updates right
- **Ranger-style Controls**: `h`/`l` moves between columns, `j`/`k` navigates within columns
- **Context Awareness**: Each column shows relevant information based on selection in previous column
- **Visual Hierarchy**: Clear indication of current selection and focus
- **Rich Information**: Progress tracking, logs, metadata in detail pane

**Navigation Shortcuts:**
- `h` / `Left Arrow`: Move focus to left column (repositories)
- `l` / `Right Arrow`: Move focus to right column (run details) 
- `j` / `Down Arrow`: Move selection down within current column
- `k` / `Up Arrow`: Move selection up within current column
- `Tab`: Cycle focus between columns (alternative to h/l)
- `Enter`: Action on selected item (view details, follow run, etc.)
- `Space`: Quick actions (toggle follow, mark favorite, etc.)

### 2. All Runs Timeline Layout
**Best for**: Temporal overview, cross-repository activity monitoring

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
├─ All Issue Runs (by recency) ────────────────────────────────────────────────────┤
│                                                                                   │
│ ● #1234 Fix authentication bug in login flow                                     │
│   🔄 Running • acme-corp/backend • fix/auth-bug • Started 5 minutes ago         │
│                                                                                   │
│ ● #1235 Add user dashboard feature with real-time updates                        │
│   ✅ Completed • acme-corp/frontend • feat/dashboard • Completed 1 hour ago      │
│                                                                                   │
│ ● #1236 Fix mobile app crash on startup - memory leak                           │
│   ❌ Failed • acme-corp/mobile-app • fix/startup-crash • Failed 30 minutes ago   │
│                                                                                   │
│ ● #1230 Add comprehensive logging system                                         │
│   ✅ Completed • acme-corp/backend • feat/logging • Completed 2 hours ago        │
│                                                                                   │
│ ● #1232 Improve UI responsiveness and animations                                 │
│   🔄 Running • acme-corp/frontend • perf/ui • Started 45 minutes ago            │
│                                                                                   │
│ ● #1229 Implement responsive design for tablet devices                           │
│   ✅ Completed • acme-corp/frontend • feat/responsive • Completed 2 days ago     │
└────────────────────────────────────────────────────────────────────────────────────┘
```

**Key Features:**
- **Chronological Ordering**: Most recent activity first
- **Cross-Repository View**: See activity across all repositories
- **Compact Format**: Each run shows essential info in 2 lines
- **Status Icons**: Visual indicators for run status
- **Repository Context**: Repository name included in each entry

### 3. Repositories Only Layout
**Best for**: Repository management, overview of project portfolio

```
┌─ RepoBird Dashboard ─────────────────── Runs: 47/100 (Pro) ── [Shift+L: Layout] ─┐
├─ Repositories ────────────────────────────────────────────────────────────────────┤
│                                                                                   │
│ 📁 acme-corp/backend                                      ⭐ 234 stars           │
│    Main repository for API services and business logic                           │
│    🔄 2 running • ✅ 8 completed • ❌ 2 failed • Last: 2 hours ago              │
│    Languages: Go, SQL • Size: 45.2 MB • Contributors: 12                        │
│                                                                                   │
│ 📁 acme-corp/frontend                                     ⭐ 89 stars            │
│    React-based web application with modern UI                                    │
│    🔄 1 running • ✅ 6 completed • ❌ 1 failed • Last: 1 day ago                │
│    Languages: TypeScript, CSS • Size: 23.1 MB • Contributors: 8                 │
│                                                                                   │
│ 📁 acme-corp/mobile-app                                   ⭐ 45 stars            │
│    React Native mobile application for iOS and Android                          │
│    ✅ 3 completed • Last: 3 days ago                                             │
│    Languages: TypeScript, Swift • Size: 18.7 MB • Contributors: 5               │
│                                                                                   │
│ 📁 acme-corp/docs                                         ⭐ 12 stars            │
│    Documentation and guides for the platform                                     │
│    ✅ 1 completed • Last: 1 week ago                                             │
│    Languages: Markdown • Size: 2.3 MB • Contributors: 4                         │
└────────────────────────────────────────────────────────────────────────────────────┘
```

**Key Features:**
- **Extended Repository Info**: Description, languages, size, contributors
- **Activity Summary**: Run statistics and last activity
- **Metadata Display**: Stars, repository size, team information
- **Full-Width Layout**: Maximize space for detailed information

## Layout Switching Mechanism

### Primary Controls
- **Shift+L**: Cycle through layouts (Triple-Column → All Runs → Repositories → repeat)
- **Ctrl+1**: Jump directly to Triple-Column layout
- **Ctrl+2**: Jump directly to All Runs layout  
- **Ctrl+3**: Jump directly to Repositories layout

### Advanced Navigation (Triple-Column Layout)
| Shortcut | Action | Context |
|----------|--------|---------|
| `h` / `Left` | Move to left column | Ranger-style navigation |
| `l` / `Right` | Move to right column | Ranger-style navigation |
| `j` / `Down` | Move selection down | Within current column |
| `k` / `Up` | Move selection up | Within current column |
| `Tab` | Cycle column focus | Alternative to h/l |
| `Shift+Tab` | Reverse cycle focus | Alternative navigation |
| `Enter` | Primary action | View details, follow run |
| `Space` | Secondary action | Toggle follow, favorite |
| `/` | Search within column | Filter items |
| `?` | Show keybindings | Help overlay |

### Layout Memory
- Remember user's preferred layout per session
- Save layout preference to config file (`~/.repobird/config.yaml`)
- Preserve focus and selection when switching layouts
- Support per-repository layout preferences (future)

### Transition Behavior
- Smooth transitions between layouts with visual feedback
- Maintain context: selected repository/run remains selected
- Preserve scroll positions where applicable
- Show brief layout name during transition

## Miller Columns Implementation Details

The triple-column layout follows Ranger's Miller Columns pattern:

### Column Interdependence
1. **Left → Center**: Selecting a repository filters runs to show only that repo's runs
2. **Center → Right**: Selecting a run shows detailed information for that specific run
3. **Navigation Flow**: Users naturally flow left-to-right through the hierarchy

### Visual Design Elements
- **Focus Indicators**: Highlighted borders around active column
- **Selection Highlighting**: Clear visual indication of selected items
- **Status Icons**: Consistent iconography for run states (🔄 ✅ ❌)
- **Column Separators**: Subtle borders to distinguish columns
- **Responsive Sizing**: Columns adjust based on terminal width

### Data Loading Strategy
- **Lazy Loading**: Only load run details when selected
- **Caching**: Cache repository and run data for smooth navigation  
- **Real-time Updates**: Live status updates for running tasks
- **Pagination**: Handle large lists efficiently

## Current Implementation Status

### Completed Features
- **Triple-Column Layout**: Fully implemented with Miller Columns navigation pattern
- **Repository List**: Shows repositories with run counts and last activity
- **Run List**: Displays runs for selected repository with status indicators
- **Run Details**: Shows detailed information about selected run
- **Keyboard Navigation**: h/j/k/l vim-style navigation between and within columns
- **Visual Indicators**: Color-coded status icons and borders
- **Column Width Management**: Automatic width calculation with proper border rendering
- **Statusline**: Shows current layout mode and navigation hints
- **Real-time Updates**: Live status updates for running tasks

### Known Issues (Fixed)
- ✅ Column width calculation now properly accounts for border rendering
- ✅ Column heights now render fully without bottom border cutoff
- ✅ Third column no longer gets cut off on the right side

## Implementation Architecture

### Bubble Tea Models
```go
type LayoutType int

const (
    LayoutTripleColumn LayoutType = iota
    LayoutAllRuns
    LayoutRepositoriesOnly
)

type DashboardModel struct {
    currentLayout     LayoutType
    tripleColumnModel *TripleColumnModel
    allRunsModel      *AllRunsModel
    reposModel        *RepositoriesModel
    
    // Shared state
    repositories []Repository
    runs         []Run
    selectedRepo *Repository
    selectedRun  *Run
}

type TripleColumnModel struct {
    leftColumn   *RepositoryListModel
    centerColumn *RunListModel
    rightColumn  *RunDetailModel
    activeColumn int // 0=left, 1=center, 2=right
    columnWidths [3]int
}
```

### Data Models
```go
type Repository struct {
    Name         string    `json:"name"`
    Description  string    `json:"description"`
    Stars        int       `json:"stars"`
    LastActivity time.Time `json:"last_activity"`
    RunCounts    RunStats  `json:"run_counts"`
    Languages    []string  `json:"languages"`
    Size         string    `json:"size"`
    Contributors int       `json:"contributors"`
}

type Run struct {
    ID          string    `json:"id"`
    Title       string    `json:"title"`
    Repository  string    `json:"repository"`
    Branch      string    `json:"branch"`
    Status      RunStatus `json:"status"`
    Progress    *Progress `json:"progress,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    Logs        []LogEntry `json:"logs,omitempty"`
}

type Progress struct {
    CurrentStep int           `json:"current_step"`
    TotalSteps  int           `json:"total_steps"`
    Steps       []ProgressStep `json:"steps"`
    ETA         *time.Time    `json:"eta,omitempty"`
}
```

## Configuration

Default configuration in `~/.repobird/config.yaml`:

```yaml
dashboard:
  default_layout: "triple_column"
  remember_layout: true
  animations: true
  auto_refresh: 30s
  
triple_column:
  column_widths: [25, 35, 40]  # percentages
  show_icons: true
  compact_mode: false
  
all_runs:
  max_items: 50
  show_repository: true
  
repositories:
  show_extended_info: true
  show_statistics: true
```

## Accessibility & Usability

- **Keyboard-First**: All functionality accessible via keyboard
- **Screen Reader Support**: Semantic markup for accessibility tools
- **High Contrast**: Support for high contrast terminal themes
- **Color-Blind Friendly**: Status indication via both color and symbols
- **Responsive**: Graceful degradation on narrow terminals
- **Help System**: Built-in help overlay (? key)

## Future Enhancements

1. **Custom Column Layouts**: User-defined column arrangements
2. **Saved Workspaces**: Named layout configurations for different projects
3. **Plugin System**: Third-party layout extensions
4. **Team Collaboration**: Shared dashboard configurations
5. **Theming**: Customizable color schemes and icons
6. **Search Integration**: Global search across all data
7. **Filtering**: Advanced filtering and sorting options
8. **Export**: Export dashboard data to various formats

---

This design leverages the proven UX patterns from Ranger while adapting them perfectly to RepoBird's workflow. The triple-column layout provides the most efficient navigation experience, while alternative layouts support different use cases and preferences.