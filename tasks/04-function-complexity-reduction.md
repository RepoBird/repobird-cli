# Task: Function Complexity Reduction ✅ COMPLETED

## ⚠️ IMPORTANT: Parallel Agent Coordination
**Note to Agent:** Other agents may be working on different tasks in parallel. To avoid conflicts:
- Only fix linting/test issues in the functions YOU are refactoring
- Do NOT fix linting issues in unrelated functions in the same file
- Do NOT fix test failures unrelated to your function decomposition
- Focus solely on breaking down the complex functions listed in this document
- When extracting helper functions, only fix linting in those new functions
- If you encounter merge conflicts, prioritize completing your complexity reduction

## Executive Summary

Analysis of the RepoBird CLI codebase has identified significant function complexity issues that violate clean code principles and hinder maintainability. This document provides a comprehensive roadmap for breaking down complex functions into smaller, more focused units.

### Key Findings
- **11 functions** exceed complexity thresholds (>50 lines or >15 cyclomatic complexity)
- **2 critical functions** with 170+ lines requiring immediate attention
- **Multiple functions** with >4 levels of nesting
- **Mixed abstraction levels** throughout TUI event handlers
- **High parameter counts** in constructor functions

## Detailed Function Analysis

### Critical Priority (Immediate Action Required)

#### 1. RunListView.Update() - CRITICAL
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/list.go`  
**Lines:** 151-407 (256 lines)  
**Metrics:**
- Cyclomatic Complexity: ~38 (Target: <15)
- Nesting Depth: 5 levels (Target: <4)
- Parameter Count: 1 (Acceptable)

**Issues:**
- Handles 8+ different message types in single function
- Mixes UI event handling with business logic (caching, preloading)
- Deep nesting in key handling switch statements
- Debug logging scattered throughout

**Refactoring Strategy:**
```go
// Extract into focused event handlers
func (v *RunListView) handleWindowSizeMsg(msg tea.WindowSizeMsg) []tea.Cmd
func (v *RunListView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (v *RunListView) handleSearchMode(msg tea.KeyMsg) tea.Cmd
func (v *RunListView) handleRunsLoaded(msg runsLoadedMsg) []tea.Cmd
func (v *RunListView) handleRunDetailsPreloaded(msg runDetailsPreloadedMsg)
func (v *RunListView) handleRetryNavigation(msg retryNavigationMsg) (tea.Model, tea.Cmd)
func (v *RunListView) handlePolling(msg pollTickMsg) []tea.Cmd
```

#### 2. CreateRunView.Update() - CRITICAL  
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/create.go`  
**Lines:** 160-331 (171 lines)  
**Metrics:**
- Cyclomatic Complexity: ~28 (Target: <15)
- Nesting Depth: 5 levels (Target: <4)
- Parameter Count: 1 (Acceptable)

**Issues:**
- Complex modal input logic with nested switches
- Mixes insert/normal mode handling
- Debug logging throughout event handling
- Form validation mixed with navigation logic

**Refactoring Strategy:**
```go
// Extract mode-specific handlers
func (v *CreateRunView) handleInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (v *CreateRunView) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd)
func (v *CreateRunView) handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd)
func (v *CreateRunView) processTextInput(msg tea.KeyMsg) []tea.Cmd
func (v *CreateRunView) handleNavigation(key string) (tea.Model, tea.Cmd)
```

### High Priority

#### 3. CreateRunView.submitRun() - HIGH
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/create.go`  
**Lines:** 584-691 (107 lines)  
**Metrics:**
- Cyclomatic Complexity: ~18 (Target: <15)
- Nesting Depth: 4 levels (Target: <4)
- Parameter Count: 0 (Good)

**Issues:**
- Mixes file I/O, validation, and API calls
- Git auto-detection logic embedded
- Extensive debug logging mixed with business logic
- Complex validation chains

**Refactoring Strategy:**
```go
// Extract validation and processing steps
func (v *CreateRunView) prepareTaskFromFile(filePath string) (models.RunRequest, error)
func (v *CreateRunView) prepareTaskFromForm() (models.RunRequest, error)
func (v *CreateRunView) validateTask(task *models.RunRequest) error
func (v *CreateRunView) autoDetectGitInfo(task *models.RunRequest) error
func (v *CreateRunView) submitToAPI(task models.RunRequest) tea.Cmd
```

#### 4. RunListView.preloadRunDetails() - HIGH
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/list.go`  
**Lines:** 624-715 (91 lines)  
**Metrics:**
- Cyclomatic Complexity: ~12 (Acceptable but complex)
- Nesting Depth: 4 levels (Target: <4)
- Parameter Count: 0 (Good)

**Issues:**
- Complex caching logic mixed with command generation
- Debug logging throughout business logic
- Multiple nested loops for run selection

**Refactoring Strategy:**
```go
// Extract preloading logic
func (v *RunListView) selectRunsToPreload() []string
func (v *RunListView) createPreloadCommands(runIDs []string) []tea.Cmd
func (v *RunListView) prioritizeSelectedRun(runIDs []string) []string
func (v *RunListView) createSinglePreloadCmd(runID string) tea.Cmd
```

#### 5. NewCreateRunViewWithCache() - HIGH
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/create.go`  
**Lines:** 56-143 (87 lines)  
**Metrics:**
- Cyclomatic Complexity: ~8 (Acceptable)
- Nesting Depth: 3 levels (Acceptable)
- Parameter Count: 5 (Target: <5)

**Issues:**
- Too many parameters (violates clean code principles)
- Mixed concerns (UI setup, caching, debugging)
- Long initialization sequence

**Refactoring Strategy:**
```go
// Use config struct pattern
type CreateRunViewConfig struct {
    Client             *api.Client
    ParentRuns         []models.RunResponse
    ParentCached       bool
    ParentCachedAt     time.Time
    ParentDetailsCache map[string]*models.RunResponse
}

func NewCreateRunViewWithConfig(config CreateRunViewConfig) *CreateRunView
func (v *CreateRunView) initializeInputFields() 
func (v *CreateRunView) setupCacheContext(config CreateRunViewConfig)
func (v *CreateRunView) loadFormData()
```

### Medium Priority

#### 6. RunDetailsView.Update() - MEDIUM
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/details.go`  
**Lines:** 187-282 (95 lines)  
**Metrics:**
- Cyclomatic Complexity: ~22 (Target: <15)
- Nesting Depth: 4 levels (Target: <4)
- Parameter Count: 1 (Acceptable)

**Issues:**
- Large key handling switch statement
- Mixed clipboard and navigation logic

**Refactoring Strategy:**
```go
func (v *RunDetailsView) handleKeyInput(msg tea.KeyMsg) []tea.Cmd
func (v *RunDetailsView) handleClipboardOperations(key string) error
func (v *RunDetailsView) handleViewportNavigation(key string)
```

#### 7. NewRunDetailsViewWithCache() - MEDIUM
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/details.go`  
**Lines:** 61-135 (74 lines)  
**Metrics:**
- Parameter Count: 5 (Target: <5)
- Mixed initialization and caching logic

#### 8. Client.doRequestWithRetry() - MEDIUM
**File:** `/home/ari/repos/repobird-cli/internal/api/client.go`  
**Lines:** 266-293 (27 lines)  
**Metrics:**
- Complex nested error handling
- Mixed retry and circuit breaker logic

### Lower Priority

#### 9. RunListView.renderStatusBar() - LOW
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/list.go`  
**Lines:** 492-527 (35 lines)  
**Issues:** Multiple string concatenations and logic branches

#### 10. CreateRunView.View() - LOW  
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/create.go`  
**Lines:** 455-567 (112 lines)  
**Issues:** Long rendering logic with mixed concerns

#### 11. RunDetailsView.updateContent() - LOW
**File:** `/home/ari/repos/repobird-cli/internal/tui/views/details.go`  
**Lines:** 360-408 (48 lines)  
**Issues:** Mixed content generation logic

## Implementation Roadmap

### Phase 1: Critical Functions (Week 1-2)
1. **RunListView.Update()** - Break into 7 focused methods
2. **CreateRunView.Update()** - Break into 5 focused methods

### Phase 2: High Priority Functions (Week 3-4)  
3. **CreateRunView.submitRun()** - Extract validation and API logic
4. **RunListView.preloadRunDetails()** - Separate caching from command logic
5. **NewCreateRunViewWithCache()** - Implement config struct pattern

### Phase 3: Medium Priority Functions (Week 5-6)
6. **RunDetailsView.Update()** - Extract key and clipboard handlers
7. **NewRunDetailsViewWithCache()** - Apply config struct pattern
8. **Client.doRequestWithRetry()** - Separate retry from error handling

### Phase 4: Lower Priority Functions (Week 7-8)
9. **Remaining functions** - Apply consistent patterns from earlier phases

## Specific Refactoring Steps

### Step-by-Step Breakdown for RunListView.Update()

1. **Create message type handlers** (Day 1)
   ```go
   func (v *RunListView) handleWindowSizeMsg(msg tea.WindowSizeMsg) []tea.Cmd {
       v.width = msg.Width
       v.height = msg.Height
       v.table.SetDimensions(msg.Width, msg.Height-4)
       v.help.Width = msg.Width
       return nil
   }
   ```

2. **Extract search mode logic** (Day 2)
   ```go
   func (v *RunListView) handleSearchMode(msg tea.KeyMsg) tea.Cmd {
       switch msg.String() {
       case "enter":
           v.searchMode = false
           v.filterRuns()
       case "esc":
           v.searchMode = false
           v.searchQuery = ""
           v.filterRuns()
       case "backspace":
           if len(v.searchQuery) > 0 {
               v.searchQuery = v.searchQuery[:len(v.searchQuery)-1]
           }
       default:
           if len(msg.String()) == 1 {
               v.searchQuery += msg.String()
           }
       }
       return nil
   }
   ```

3. **Create key handler dispatcher** (Day 3)
   ```go
   func (v *RunListView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
       if v.searchMode {
           return v, v.handleSearchMode(msg)
       }
       
       return v.handleNormalModeKeys(msg)
   }
   ```

4. **Extract navigation logic** (Day 4)
   ```go
   func (v *RunListView) handleNavigationKeys(msg tea.KeyMsg) []tea.Cmd {
       var cmds []tea.Cmd
       
       switch {
       case key.Matches(msg, v.keys.Up):
           v.table.MoveUp()
           cmds = append(cmds, v.preloadSelectedRun())
       case key.Matches(msg, v.keys.Down):
           v.table.MoveDown()
           cmds = append(cmds, v.preloadSelectedRun())
       // ... other navigation cases
       }
       
       return cmds
   }
   ```

5. **Update main Update() method** (Day 5)
   ```go
   func (v *RunListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
       var cmds []tea.Cmd

       switch msg := msg.(type) {
       case tea.WindowSizeMsg:
           cmds = append(cmds, v.handleWindowSizeMsg(msg)...)
       case tea.KeyMsg:
           return v.handleKeyMsg(msg)
       case runsLoadedMsg:
           cmds = append(cmds, v.handleRunsLoaded(msg)...)
       case runDetailsPreloadedMsg:
           v.handleRunDetailsPreloaded(msg)
       case retryNavigationMsg:
           return v.handleRetryNavigation(msg)
       case pollTickMsg:
           cmds = append(cmds, v.handlePolling(msg)...)
       case spinner.TickMsg:
           if v.loading {
               var cmd tea.Cmd
               v.spinner, cmd = v.spinner.Update(msg)
               cmds = append(cmds, cmd)
           }
       }

       return v, tea.Batch(cmds...)
   }
   ```

## Testing Requirements

### Unit Tests Required for Each Refactored Function

1. **Event Handler Tests**
   ```go
   func TestRunListView_HandleWindowSizeMsg(t *testing.T)
   func TestRunListView_HandleKeyMsg_SearchMode(t *testing.T)  
   func TestRunListView_HandleKeyMsg_NormalMode(t *testing.T)
   func TestRunListView_HandleNavigationKeys(t *testing.T)
   ```

2. **Business Logic Tests**
   ```go
   func TestCreateRunView_PrepareTaskFromFile(t *testing.T)
   func TestCreateRunView_PrepareTaskFromForm(t *testing.T)
   func TestCreateRunView_ValidateTask(t *testing.T)
   func TestCreateRunView_AutoDetectGitInfo(t *testing.T)
   ```

3. **Integration Tests**
   ```go
   func TestRunListView_Update_MessageFlow(t *testing.T)
   func TestCreateRunView_Update_ModeTransitions(t *testing.T)
   ```

### Test Coverage Targets
- **Critical functions**: 95% coverage
- **High priority functions**: 90% coverage  
- **Medium priority functions**: 85% coverage
- **Lower priority functions**: 80% coverage

## Code Quality Metrics (Post-Refactoring)

### Target Metrics
- **Function Length**: <50 lines per function
- **Cyclomatic Complexity**: <15 per function
- **Nesting Depth**: <4 levels
- **Parameter Count**: <5 parameters
- **Single Responsibility**: Each function should have one clear purpose

### Validation Checklist
- [ ] No function exceeds 50 lines
- [ ] No function has cyclomatic complexity >15
- [ ] No function has >4 levels of nesting
- [ ] No function has >5 parameters
- [ ] All functions have single, clear responsibility
- [ ] Side effects are indicated by function names
- [ ] Boolean parameters are eliminated or well-documented
- [ ] Debug logging is separated from business logic

## Success Criteria

### Code Quality Improvements
1. **Reduced Complexity**: All functions under complexity thresholds
2. **Improved Testability**: Each function can be unit tested in isolation
3. **Enhanced Readability**: Clear function names and single responsibilities
4. **Better Maintainability**: Changes isolated to specific function areas

### Performance Considerations
- Refactoring should not impact runtime performance
- Memory allocation patterns should remain consistent
- TUI responsiveness must be maintained

## Final Implementation Task

After completing all refactoring phases, run the following validation:

```bash
make lint-fix fmt
go test ./... -v
go test ./... -race
golangci-lint run
```

All tests must pass and linting must be clean before considering the task complete.