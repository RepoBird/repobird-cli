# Bulk View Refactoring Plan

## Overview
The current `internal/tui/views/bulk.go` file is ~1300 lines and violates the single responsibility principle by handling multiple concerns. This document outlines a plan to refactor it into 8 focused files following the same pattern used for the create view.

## Current Issues
- **Large monolithic file**: 1300+ lines in a single file
- **Multiple responsibilities**: UI rendering, event handling, API commands, component logic
- **Maintenance difficulty**: Hard to navigate and modify specific functionality
- **Testing challenges**: Difficult to test individual components in isolation

## Proposed File Structure

### 1. `bulk_view.go` (Main orchestrator - ~150 lines)
**Responsibility**: Core view lifecycle and mode orchestration

**Contents**:
- `BulkView` struct definition (lines 34-69)
- `BulkMode`, `BulkRunItem`, `RunStatus` types (lines 23-102)  
- `bulkKeyMap` struct and `defaultBulkKeyMap()` (lines 104-233)
- `NewBulkView()` constructor (lines 134-152)
- `Init()` method (lines 235-241)
- `Update()` method with mode switching and delegation (lines 243-325)
- Main `View()` method with rendering delegation (lines 327-343)

**Dependencies**:
- Calls handler methods from `bulk_handlers.go`
- Calls render methods from `bulk_rendering.go`  
- Calls command functions from `bulk_commands.go`

### 2. `bulk_handlers.go` (Event handling - ~110 lines)
**Responsibility**: Key event handling for all modes

**Contents**:
- `handleFileSelectKeys()` (lines 478-495)
- `handleRunListKeys()` (lines 496-553)
- `handleRunEditKeys()` (lines 554-563)
- `handleProgressKeys()` (lines 564-575)
- `handleResultsKeys()` (lines 576-590)

**Methods**: All receiver methods on `*BulkView` for key handling

### 3. `bulk_rendering.go` (UI rendering - ~130 lines)
**Responsibility**: Rendering views for all modes

**Contents**:
- `renderFileSelect()` (lines 345-357)
- `renderRunList()` (lines 359-407)
- `renderRunEdit()` (lines 409-411)
- `renderProgress()` (lines 413-418)
- `renderResults()` (lines 420-470)
- `renderHelp()` (lines 472-476)

**Methods**: All receiver methods on `*BulkView` for rendering

### 4. `bulk_commands.go` (API operations - ~155 lines)
**Responsibility**: Command execution and API interactions

**Contents**:
- `loadFiles()` (lines 593-623)
- `submitBulkRuns()` (lines 625-711)
- `pollProgress()` (lines 713-736)
- `cancelBatch()` (lines 738-747)

**Methods**: All receiver methods on `*BulkView` returning `tea.Cmd`

### 5. `bulk_messages.go` (Message types - ~35 lines)
**Responsibility**: Message type definitions and related structs

**Contents**:
- `fileSelectedMsg` (lines 750-752)
- `bulkRunsLoadedMsg` (lines 754-761)
- `bulkSubmittedMsg` (lines 763-767)
- `bulkProgressMsg` (lines 769-774)
- `bulkCancelledMsg` (lines 776)
- `errMsg` (lines 778-780)
- `BulkRunResult` struct (lines 95-102)

### 6. `bulk_file_selector.go` (File selection component - ~270 lines)
**Responsibility**: File selection UI component

**Contents**:
- `BulkFileSelector` struct and related types (lines 783-821)
- `NewBulkFileSelector()` constructor (lines 822-833)
- `Init()`, `Update()`, `View()` methods (lines 835-930)
- Helper methods: `loadFiles()`, `applyFilter()`, `detectFileType()`, etc. (lines 932-1046)
- `filesLoadedMsg` type (lines 1048-1050)

### 7. `bulk_run_editor.go` (Run editing component - ~145 lines)
**Responsibility**: Individual run editing UI

**Contents**:
- `RunEditor` struct (lines 1053-1060)
- `NewRunEditor()` constructor (lines 1062-1087)
- `SetRun()`, `UpdateRunEditor()`, `View()` methods (lines 1089-1176)
- `updateFocus()` helper (lines 1178-1194)

### 8. `bulk_progress_view.go` (Progress tracking component - ~105 lines)
**Responsibility**: Bulk run progress display

**Contents**:
- `BulkProgressView` struct (lines 1197-1202)
- `NewBulkProgressView()` constructor (lines 1204-1212)
- `UpdateProgressView()`, `View()` methods (lines 1214-1262)
- Helper methods: `UpdateProgress()`, `makeProgressBar()`, `getStatusIcon()` (lines 1264-1300)

## Refactoring Steps

### Phase 1: Extract Components
1. Create `bulk_file_selector.go` with `BulkFileSelector` component
2. Create `bulk_run_editor.go` with `RunEditor` component  
3. Create `bulk_progress_view.go` with `BulkProgressView` component
4. Update imports in main file to reference new component files

### Phase 2: Extract Message Types
1. Create `bulk_messages.go` with all message type definitions
2. Update main file to import message types

### Phase 3: Extract Commands
1. Create `bulk_commands.go` with command functions
2. Move receiver methods that return `tea.Cmd`
3. Update main `Update()` method to call command functions

### Phase 4: Extract Handlers  
1. Create `bulk_handlers.go` with key event handlers
2. Move all `handle*Keys()` receiver methods
3. Update main `Update()` method to call handler methods

### Phase 5: Extract Rendering
1. Create `bulk_rendering.go` with rendering methods
2. Move all `render*()` receiver methods  
3. Update main `View()` method to call render methods

### Phase 6: Clean Up Main File
1. Remove extracted code from `bulk.go`
2. Update imports and ensure all references work
3. Add package documentation

## Benefits After Refactoring

### Maintainability
- Each file has a single, clear responsibility
- Easier to locate and modify specific functionality
- Reduced cognitive load when working on specific features

### Testability  
- Components can be tested in isolation
- Handlers can be unit tested separately from rendering
- Command functions can be mocked more easily

### Code Organization
- Follows established patterns in the codebase (like create view)
- Consistent with project architecture guidelines
- Better separation of concerns

### File Size Management
- No file exceeds ~300 lines (most are under 200)
- Each file fits comfortably on screen
- Easier code review and navigation

## Testing Considerations

After refactoring, ensure:
- All existing tests continue to pass
- Component integration still works correctly
- Message passing between components functions properly
- Mode transitions work as expected

## Follow-up Tasks

1. Update any documentation referencing the bulk view structure
2. Consider similar refactoring for other large view files
3. Add component-specific tests for newly extracted components
4. Review for any performance implications of the refactoring