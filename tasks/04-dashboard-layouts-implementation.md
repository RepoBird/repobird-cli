# Dashboard Layout System Implementation Plan

## Overview

This document outlines the comprehensive implementation plan for integrating the dashboard layout system described in `docs/dashboard-layouts.md` without breaking the current excellent width/height sizing calculations and performance optimizations.

## Current Architecture Analysis

### Strengths to Preserve
- **Sophisticated Dimension Handling**: `handleWindowSizeMsg()` in `list.go` (lines 200-237) with precise calculations for title, search, help, and table heights
- **Excellent Caching System**: Pre-loading, details cache, global cache in `cache/` package
- **Performance Optimizations**: Real-time polling, background preloading, sophisticated state management
- **Flexible Table Component**: Dynamic column width calculations with flex/fixed width support
- **Vim-style Navigation**: Existing h/j/k/l key bindings perfect for Miller Columns

### Current File Structure
```
internal/tui/
├── app.go                    # Entry point (simple, creates RunListView)
├── components/
│   ├── keys.go              # Excellent key mapping system
│   ├── table.go             # Sophisticated table with flex calculations
│   └── repository_selector.go
├── views/
│   ├── list.go              # Complex RunListView with caching/polling
│   ├── details.go           # RunDetailsView with viewport
│   └── create.go            # CreateRunView
├── styles/
│   └── theme.go
└── debug/
    └── logger.go
```

## Implementation Strategy

**Key Principle**: **Extend, don't replace** the existing excellent system. Create a dashboard wrapper that contains and coordinates existing views while adding new layout options.

## Phase 1: Foundation Models & Components

### 1.1 Create Dashboard Data Models

**File**: `internal/models/dashboard.go`
```go
type LayoutType int

const (
    LayoutTripleColumn LayoutType = iota
    LayoutAllRuns
    LayoutRepositoriesOnly
)

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

type RunStats struct {
    Running   int `json:"running"`
    Completed int `json:"completed"`
    Failed    int `json:"failed"`
    Total     int `json:"total"`
}
```

### 1.2 Create Repository Data Model

**File**: `internal/models/repository.go`
- Extend existing models to support repository metadata
- Add repository aggregation methods
- Support for repository filtering and grouping

### 1.3 Extend Key Mapping System

**File**: `internal/tui/components/keys.go` (extend existing)
```go
// Add new keys to existing KeyMap struct
LayoutSwitch    key.Binding  // Shift+L
LayoutTriple    key.Binding  // Ctrl+1  
LayoutAllRuns   key.Binding  // Ctrl+2
LayoutRepos     key.Binding  // Ctrl+3
```

### 1.4 Create Miller Columns Component

**File**: `internal/tui/components/column_layout.go`
```go
type ColumnLayout struct {
    columns      []Column
    activeColumn int
    widths       []int
    // Leverages existing Table component for each column
}
```

## Phase 2: New View Implementations

### 2.1 Create Dashboard Controller View

**File**: `internal/tui/views/dashboard.go`
```go
type DashboardView struct {
    client       *api.Client
    currentLayout LayoutType
    
    // Layout-specific models (reuse existing where possible)
    tripleColumn *TripleColumnView
    allRuns      *RunListView     // Reuse existing!
    repositories *RepositoriesView
    
    // Preserve all existing excellent features
    width        int
    height       int
    userInfo     *models.UserInfo
    
    // Shared state across layouts
    repositories []Repository
    runs         []models.RunResponse
    selectedRepo *Repository
    selectedRun  *models.RunResponse
}
```

**Key Features:**
- **Layout Switching**: Shift+L cycles through layouts
- **State Preservation**: Selected repository/run maintained across layout switches
- **Dimension Management**: Inherits and delegates to child views
- **Cache Integration**: Leverages existing caching system

### 2.2 Implement Triple-Column Miller Columns View

**File**: `internal/tui/views/triple_column.go`
```go
type TripleColumnView struct {
    leftColumn   *RepositoryListModel  // New
    centerColumn *RunListView          // Reuse existing!
    rightColumn  *RunDetailsView       // Reuse existing!
    
    activeColumn int // 0=left, 1=center, 2=right
    columnWidths [3]int
    
    // Preserve existing dimension handling
    width  int
    height int
}
```

**Implementation Strategy:**
- **Reuse Existing Views**: Center and right columns use existing `RunListView` and `RunDetailsView`
- **Miller Columns Navigation**: h/l moves between columns, j/k navigates within columns
- **Column Width Distribution**: 25%/35%/40% with responsive adjustments
- **State Synchronization**: Selection in left updates center, selection in center updates right

### 2.3 Create All Runs Timeline View

**File**: `internal/tui/views/all_runs.go`
```go
type AllRunsView struct {
    *RunListView  // Inherit existing excellent implementation
    // Override rendering for chronological display
    displayMode TimelineDisplayMode
}
```

**Implementation Strategy:**
- **Inherit Excellence**: Extends existing `RunListView` with its caching and performance
- **Modified Rendering**: Override `View()` method for timeline format
- **Preserve Features**: All existing search, filtering, polling functionality intact

### 2.4 Create Repositories-Only View

**File**: `internal/tui/views/repositories.go`
```go
type RepositoriesView struct {
    client     *api.Client
    table      *components.Table  // Reuse excellent table component
    repos      []Repository
    
    // Preserve existing dimension patterns
    width      int
    height     int
}
```

## Phase 3: Integration & Configuration

### 3.1 Modify App Entry Point

**File**: `internal/tui/app.go` (modify existing)
```go
func (a *App) Run() error {
    // Replace RunListView with DashboardView
    dashboardView := views.NewDashboardView(a.client)
    p := tea.NewProgram(dashboardView, tea.WithAltScreen(), tea.WithMouseCellMotion())
    _, err := p.Run()
    return err
}
```

### 3.2 Add Configuration Support

**File**: Extend existing config system
```yaml
# Add to ~/.repobird/config.yaml
dashboard:
  default_layout: "triple_column"
  remember_layout: true
  animations: true
  auto_refresh: 30s
  
triple_column:
  column_widths: [25, 35, 40]
  show_icons: true
```

### 3.3 API Client Extensions

**Files**: `internal/api/client.go` (extend existing)
- Add repository listing methods
- Add repository metadata fetching
- Extend existing caching to support repository data

## Phase 4: Advanced Features & Polish

### 4.1 Enhanced Navigation System

**Features:**
- **Layout Memory**: Remember user's preferred layout per session
- **Focus Preservation**: Maintain selected items when switching layouts
- **Smooth Transitions**: Visual feedback during layout switches
- **Help Integration**: Context-sensitive help for each layout

### 4.2 Repository Data Integration

**Implementation:**
- **Repository Discovery**: Extract from existing run data
- **Metadata Enrichment**: Add repository statistics and information
- **Grouping Logic**: Aggregate runs by repository
- **Filtering**: Repository-based filtering across layouts

### 4.3 Performance Optimizations

**Preserve and Extend:**
- **Lazy Loading**: Only load repository details when needed
- **Smart Caching**: Extend existing cache to repository metadata
- **Background Updates**: Real-time updates for repository statistics
- **Memory Management**: Efficient handling of large repository lists

## Implementation Sequence

### Sprint 1: Foundation (Files to Create/Modify)
1. **Create**: `internal/models/dashboard.go`
2. **Create**: `internal/models/repository.go` 
3. **Extend**: `internal/tui/components/keys.go`
4. **Create**: `internal/tui/components/column_layout.go`

### Sprint 2: Core Views (Files to Create/Modify)
1. **Create**: `internal/tui/views/dashboard.go`
2. **Create**: `internal/tui/views/triple_column.go`
3. **Create**: `internal/tui/views/all_runs.go`
4. **Create**: `internal/tui/views/repositories.go`

### Sprint 3: Integration (Files to Modify)
1. **Modify**: `internal/tui/app.go`
2. **Extend**: API client methods
3. **Extend**: Configuration system
4. **Create**: Repository data services

### Sprint 4: Polish & Testing
1. **Testing**: Comprehensive test coverage
2. **Performance**: Benchmarking and optimization
3. **Documentation**: Update help text and documentation
4. **Integration**: Final testing and refinement

## Key Design Decisions

### 1. Preservation Strategy
- **Keep Existing Views**: `RunListView`, `RunDetailsView`, `CreateRunView` remain unchanged
- **Wrapper Pattern**: `DashboardView` coordinates existing views
- **Dimension Delegation**: Pass width/height to existing views exactly as before
- **Cache Integration**: Extend existing cache rather than replacing it

### 2. Miller Columns Implementation
- **Component Reuse**: Left column new, center/right columns reuse existing
- **Navigation Enhancement**: h/l for column switching, existing j/k for navigation
- **State Management**: Shared state with clear ownership boundaries
- **Responsive Design**: Column widths adjust based on terminal size

### 3. Performance Considerations
- **Lazy Initialization**: Create layout views only when needed
- **Memory Sharing**: Share data between layouts rather than duplicating
- **Smart Updates**: Only refresh active layout
- **Background Processing**: Repository data loading in background

## Risk Mitigation

### 1. Preserve Existing Functionality
- **Extensive Testing**: Ensure existing features continue to work
- **Fallback Options**: Ability to revert to current list view
- **Progressive Enhancement**: Features added incrementally

### 2. Performance Maintenance
- **Benchmarking**: Compare performance before and after
- **Memory Profiling**: Ensure no memory leaks with new layouts
- **Load Testing**: Test with large numbers of repositories/runs

### 3. User Experience
- **Gradual Introduction**: Layout switching optional initially
- **Clear Documentation**: Help system updated for new features
- **Keyboard Consistency**: Maintain existing keyboard shortcuts

## Success Metrics

1. **Functionality**: All existing features continue to work perfectly
2. **Performance**: No degradation in startup time or memory usage
3. **Usability**: Layout switching feels natural and responsive
4. **Code Quality**: Clean integration without architectural compromises
5. **Maintainability**: Easy to extend with additional layouts in the future

## Testing Strategy

### 1. Unit Tests
- New model methods and data structures
- Layout switching logic
- Column width calculations
- Repository data aggregation

### 2. Integration Tests  
- Dashboard view coordination
- State preservation across layouts
- API client extensions
- Configuration management

### 3. End-to-End Tests
- Complete user workflows in each layout
- Layout switching scenarios
- Performance with large datasets
- Error handling and recovery

### 4. Manual Testing
- User experience validation
- Keyboard navigation flows
- Visual design and spacing
- Help system completeness

This implementation plan preserves all the excellent existing functionality while adding the sophisticated dashboard layout system described in the requirements. The key is extending rather than replacing the current system, ensuring no regression in the outstanding width/height calculations and performance optimizations that are already in place.