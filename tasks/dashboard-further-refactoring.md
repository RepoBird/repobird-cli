# Dashboard.go Further Refactoring Task

## Current Status

The dashboard.go file has been reduced from 3,423 lines to 2,738 lines (20% reduction) by extracting functionality into focused files. However, **2,738 lines is still too large** for maintainability.

## Current File Organization

- **`dashboard.go`** (2,745 lines) - Core Update/View/Init methods and state management
- **`dash_rendering.go`** (584 lines) - All layout rendering 
- **`dash_data.go`** (461 lines) - Data loading and repository operations
- **`dash_status_info.go`** (254 lines) - Status/user info overlay
- **`dash_formatting.go`** (222 lines) - Text formatting utilities
- **`dash_fzf.go`** (124 lines) - FZF overlay logic
- **`dash_messages.go`** (42 lines) - Custom message types
- **`dash_clipboard.go`** (38 lines) - Clipboard operations

**Current Total**: 4,470 lines across 8 files (main file still 61% of total)

## Phase 2 Refactoring Goals

**Target**: Reduce main `dashboard.go` to **under 1,500 lines** (45% further reduction)

### Priority 1: Extract Navigation Logic

**File**: `dash_navigation.go` (~400-500 lines)

Extract all navigation-related methods (with line references):
- `handleMillerColumnsNavigation()` (dashboard.go:698 - large method ~276 lines)
- `cycleLayout()` (dashboard.go:974 - ~14 lines)
- `scrollToSelected()` (dashboard.go:1634 - ~38 lines)
- `findNextNonEmptyLine()` (dashboard.go:665 - ~33 lines)
- Navigation key handling logic from Update method
- Column focus management logic
- Selection index management helpers

### Priority 2: Extract Update Logic 

**File**: `dash_updates.go` (~400-500 lines)

Extract viewport and content update methods (with line references):
- `updateDetailLines()` (dashboard.go:1112 - ~118 lines)
- `updateAllRunsListData()` (dashboard.go:1230 - ~29 lines)  
- `updateViewportSizes()` (dashboard.go:1259 - ~34 lines)
- `updateViewportContent()` (dashboard.go:1293 - ~12 lines)
- `updateRepoViewportContent()` (dashboard.go:1305 - ~70 lines)
- `updateRunsViewportContent()` (dashboard.go:1375 - ~133 lines)
- `updateDetailsViewportContent()` (dashboard.go:1508 - ~126 lines)
- Related viewport helper methods and update coordination logic

### Priority 3: Extract State Management

**File**: `dash_state.go` (~150-200 lines)

Extract state management and helper methods (with line references):
- `getAPIRepositoryForRepo()` (dashboard.go:2713 - ~20 lines)
- `getRepositoryByName()` (dashboard.go:2733 - ~11 lines)
- `isEmptyLine()` (dashboard.go:660 - ~5 lines)
- `hasCurrentSelectionURL()` (dashboard.go:2668 - ~45 lines)
- State validation methods
- Index bounds checking helpers
- Selection state management utilities

### Priority 4: Consolidate Help/Docs Rendering

**Target**: Move remaining rendering methods to `dash_status_info.go` (~150-200 lines)

Extract remaining rendering and overlay methods (with line references):
- `handleStatusInfoNavigation()` (dashboard.go:1850 - ~108 lines)
- `handleHelpNavigation()` (dashboard.go:1958 - ~21 lines)  
- `renderHelp()` (dashboard.go:2391 - ~10 lines)
- `renderDocsOld()` (dashboard.go:2401 - ~158 lines)
- `getDocsPages()` (dashboard.go:2559 - ~109 lines)
- `initializeStatusInfoFields()` (dashboard.go:1672 - ~178 lines)

**Note**: These should go to `dash_status_info.go` rather than a new file since they're all overlay-related.

## Implementation Strategy

### Step 1: Navigation Logic Extraction
```bash
# Target methods to extract:
- handleMillerColumnsNavigation()
- cycleLayout() 
- scrollToSelected()
- Column navigation helpers
- Focus management
```

### Step 2: Update Logic Extraction  
```bash
# Target methods to extract:
- updateDetailLines()
- updateAllRunsListData()  
- updateViewportSizes()
- updateViewportContent()
- All viewport update methods
```

### Step 3: State Management Extraction
```bash
# Target methods to extract:
- getAPIRepositoryForRepo()
- getRepositoryByName()
- State validation helpers
- Index management helpers
```

### Step 4: Event Handling Extraction
```bash
# Target methods to extract:
- Specific key handling (non-nav)
- Window resize logic
- Overlay event handling
- Message processing helpers
```

## Final Target Structure

After Phase 2 refactoring:

```
dashboard.go           (~1,200 lines) - Core Update/View/Init only
├── dash_navigation.go (~450 lines)   - Navigation logic 
├── dash_updates.go    (~500 lines)   - Viewport/content updates  
├── dash_state.go      (~180 lines)   - State management helpers
├── dash_rendering.go  (~584 lines)   - Layout rendering
├── dash_data.go       (~461 lines)   - Data loading
├── dash_status_info.go (~840 lines)  - Status overlay + help/docs
├── dash_formatting.go (~222 lines)   - Text utilities
├── dash_fzf.go        (~124 lines)   - FZF logic
├── dash_messages.go   (~42 lines)    - Message types
└── dash_clipboard.go  (~38 lines)    - Clipboard ops
```

**Total**: ~4,640 lines across 10 focused files  
**Main file**: 1,200 lines (56% reduction from current 2,745 lines)
**Largest file**: dash_status_info.go at 840 lines (was 254, now includes all overlay logic)

## Benefits

1. **Maintainability**: Each file has a single, clear responsibility
2. **Readability**: Much easier to understand and modify specific functionality  
3. **Testing**: Easier to write focused unit tests for each component
4. **Collaboration**: Multiple developers can work on different aspects simultaneously
5. **Code Review**: Smaller, focused changes are easier to review

## Implementation Notes

- Maintain all existing functionality
- Preserve method signatures for internal calls
- Keep all public interfaces unchanged
- Add clear documentation to each extracted file
- Ensure proper import organization
- Run full test suite after each extraction

## Success Criteria

- [x] Phase 1: Reduce from 3,423 to 2,745 lines (20% reduction) ✅
- [ ] Phase 2: Reduce from 2,745 to ~1,200 lines (56% reduction)
- [ ] All tests pass
- [ ] No functionality regression  
- [ ] Clear file organization with single responsibilities
- [ ] No single file over 850 lines (dash_status_info.go at limit)
- [ ] Documentation updated to reflect new structure

## Estimated Effort

- **Navigation Logic**: 3-4 hours (large `handleMillerColumnsNavigation` method)
- **Update Logic**: 3-4 hours (multiple viewport methods)  
- **State Management**: 1-2 hours (smaller, simpler methods)
- **Help/Docs Consolidation**: 2-3 hours (move to dash_status_info.go)
- **Testing & Validation**: 2-3 hours (ensure no regressions)

**Total**: 11-16 hours of focused development work

## Additional Considerations

### Method Dependencies
Several methods have interdependencies that must be preserved:
- `handleMillerColumnsNavigation()` calls `scrollToSelected()` 
- Update methods call each other in sequence
- State helpers are used throughout navigation logic

### Testing Strategy
- Run `dashboard_*_test.go` after each extraction
- Verify TUI functionality manually in each layout mode
- Test navigation, FZF, status overlay, and help screens
- Ensure clipboard operations still work correctly

### Alternative: Larger State File
If dash_status_info.go becomes too large (>850 lines), consider creating:
- `dash_overlays.go` for help/docs rendering (~350 lines)
- Keep status info methods in original file (~500 lines)