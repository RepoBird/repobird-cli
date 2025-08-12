# Refactor Details View - Code Duplication Analysis

## Overview
The `internal/tui/views/details.go` file is 1297 lines long with significant code duplication and opportunities for refactoring. This document outlines the issues and proposes a refactoring strategy.

## Current Issues

### 1. File Length
- **Problem**: 1297 lines in a single file makes it hard to navigate and maintain
- **Impact**: Difficult to find specific functionality, increased cognitive load

### 2. Constructor Proliferation
- **Lines**: 70-255
- **Issue**: 6 different constructor functions with overlapping functionality
  - `NewRunDetailsView`
  - `NewRunDetailsViewWithCache` 
  - `NewRunDetailsViewWithConfig`
  - `NewRunDetailsViewWithDashboardState`
  - `NewRunDetailsViewWithCacheAndDimensions`
  - `NewRunDetailsViewWithCache` (backward compatibility)
- **Solution**: Use builder pattern or single constructor with options

### 3. Duplicate Clipboard Logic
- **Lines**: 458-538
- **Duplication**:
  - Copy selected field (464-495)
  - Copy current line (478-494)
  - Both have identical truncation, error handling, and status feedback
- **Solution**: Extract `copyWithFeedback(text string, description string)` helper

### 4. Repeated Highlight Styling
- **Occurrences**:
  - Lines 809-826: Highlight during rendering
  - Lines 1094-1109: createHighlightedContent method
- **Solution**: Create shared highlight style constants or methods

### 5. Complex Cache Retry Logic
- **Lines**: 279-301
- **Issue**: Deeply nested if-else for cache retry attempts
- **Solution**: Extract to `attemptCacheLoad()` method

## Proposed Refactoring Strategy

### Phase 1: Extract Helper Methods (Within File)
```go
// Helper methods to add:
- copyWithFeedback(text, description string) error
- applyHighlightStyle(line string, isYankBlink bool) string  
- attemptCacheLoad(runID string) (*models.RunResponse, bool)
- truncateForDisplay(text string, maxLen int) string
- addFieldToContent(label, value string) 
```

### Phase 2: Split Into Multiple Files

#### Option A: Feature-Based Split
```
internal/tui/views/
├── details.go                 // Core struct, Update, View
├── details_constructors.go    // All constructor functions
├── details_clipboard.go       // Clipboard operations
├── details_navigation.go      // Row/field navigation
├── details_rendering.go       // Content rendering helpers
├── details_polling.go         // Polling and status updates
└── details_test.go           // Keep tests together
```

#### Option B: Component Extraction
```
internal/tui/components/
├── clipboard_handler.go       // Reusable clipboard component
├── field_navigator.go         // Reusable field navigation
└── highlight_renderer.go      // Shared highlighting logic

internal/tui/views/
├── details.go                // Simplified, using components
└── details_test.go
```

### Phase 3: Specific Refactorings

#### 1. Consolidate Constructors
```go
// Single constructor with functional options
type DetailsOption func(*RunDetailsView)

func WithCache(cache *cache.SimpleCache) DetailsOption {
    return func(v *RunDetailsView) {
        v.cache = cache
    }
}

func WithDimensions(width, height int) DetailsOption {
    return func(v *RunDetailsView) {
        v.width = width
        v.height = height
    }
}

func NewRunDetailsView(client APIClient, run models.RunResponse, opts ...DetailsOption) *RunDetailsView {
    // Single implementation
}
```

#### 2. Extract Clipboard Handler
```go
type ClipboardHandler struct {
    statusLine *components.StatusLine
}

func (c *ClipboardHandler) CopyWithFeedback(text, description string) tea.Cmd {
    // Unified clipboard logic with feedback
}
```

#### 3. Simplify Field Navigation
```go
type FieldNavigator struct {
    selectedRow  int
    fieldValues  []string
    fieldRanges  [][2]int
}

func (f *FieldNavigator) HandleNavigation(key string) (updated bool) {
    // Centralized navigation logic
}
```

#### 4. Extract Rendering Helpers
```go
// Move to details_rendering.go
func (v *RunDetailsView) renderField(label, value string) string
func (v *RunDetailsView) renderMultilineField(label, value string) string  
func (v *RunDetailsView) renderSeparator(text string) string
func (v *RunDetailsView) renderStatusHistory() string
```

## Implementation Plan

### Week 1: Internal Refactoring
- [ ] Extract helper methods within the file
- [ ] Consolidate duplicate clipboard logic
- [ ] Unify highlight styling
- [ ] Simplify constructor pattern
- [ ] Add unit tests for extracted methods

### Week 2: File Splitting
- [ ] Create new files for each concern
- [ ] Move methods to appropriate files
- [ ] Update imports and dependencies
- [ ] Ensure all tests pass
- [ ] Update documentation

### Week 3: Component Extraction (Optional)
- [ ] Identify patterns used in other views
- [ ] Create reusable components
- [ ] Refactor other views to use components
- [ ] Add component tests

## Success Metrics
- [ ] No file exceeds 500 lines
- [ ] Code duplication reduced by >50%
- [ ] All existing tests pass
- [ ] Test coverage maintained or improved
- [ ] Performance unchanged or improved
- [ ] Backward compatibility maintained

## Testing Strategy
1. Run existing tests after each refactoring step
2. Add tests for new helper methods
3. Benchmark performance before/after
4. Manual testing of TUI interactions
5. Verify clipboard operations on different platforms

## Risks and Mitigations
- **Risk**: Breaking existing functionality
  - **Mitigation**: Small incremental changes with tests
- **Risk**: Performance regression
  - **Mitigation**: Benchmark critical paths
- **Risk**: Making code harder to understand
  - **Mitigation**: Clear naming, good documentation

## Alternative Approach: Minimal Refactoring
If time is limited, focus on:
1. Extract clipboard duplication (biggest win)
2. Consolidate constructors to 2-3 variants
3. Move helper methods to bottom of file
4. Add TODO comments for future refactoring

## Code Smell Summary
- **Long Method**: View() at 120 lines, updateContent() at 160 lines
- **Long Parameter List**: Multiple constructors with 6+ parameters
- **Duplicate Code**: Clipboard operations, highlighting, truncation
- **Complex Conditionals**: Cache retry logic, navigation handling
- **Feature Envy**: Direct manipulation of viewport, spinner, statusLine