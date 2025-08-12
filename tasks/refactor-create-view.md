# Refactor Create View - Code Splitting Plan

## Problem Statement
The `internal/tui/views/create.go` file has grown to 2,477 lines, making it difficult to:
- Navigate and understand the code
- Maintain and debug issues
- Add new features without increasing complexity
- Follow single responsibility principle
- Test individual components

## Goal
Split `create.go` into multiple focused files following the established patterns in the codebase (similar to dashboard and details views), improving:
- Code organization and readability
- Maintainability and testability
- Developer experience
- Adherence to Go best practices

## Current State Analysis

### File Statistics
- **Total Lines**: 2,477
- **Functions**: 65+ methods
- **Struct Fields**: 40+ fields in CreateRunView
- **Responsibilities**: Mixed (UI, input handling, API calls, config management, validation)

### Major Components Identified
1. **Core Structure & Lifecycle**: Struct definition, constructors, Init/Update
2. **Input Handling**: Multiple input modes (Normal, Insert, Error), field navigation
3. **Rendering**: Complex UI layouts, status bars, overlays, error displays
4. **Task Submission**: API communication, validation, duplicate detection
5. **Configuration**: File loading/saving, form persistence
6. **Repository Management**: Selection, FZF integration, auto-detection
7. **Helper Functions**: Message types, utilities, animations

## Proposed File Structure

### 1. `create.go` (Core Orchestrator)
**Target Size**: ~400 lines  
**Responsibilities**: Main controller and state management

**Contents**:
- `CreateRunView` struct definition (all fields)
- Constructor functions:
  - `NewCreateRunView(client APIClient)`
  - `NewCreateRunViewWithConfig(cfg CreateRunViewConfig)`
  - `NewCreateRunViewWithCache(...)`
- `Init() tea.Cmd` - initialization logic
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)` - main message dispatcher
- `handleWindowSizeMsg(msg tea.WindowSizeMsg)` - window resize handling
- Core state initialization methods

**Key Decisions**:
- Keep all struct fields here for single source of truth
- Update() remains here as the central orchestrator
- Delegates to other files for specific handling

### 2. `create_input.go` (Input Management)
**Target Size**: ~500 lines  
**Responsibilities**: All user input handling and field management

**Contents**:
- Input mode handlers:
  - `handleInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd)`
  - `handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd)`
  - `handleErrorMode(msg tea.KeyMsg) (tea.Model, tea.Cmd)`
- Field management:
  - `updateFields(msg tea.KeyMsg) []tea.Cmd`
  - `nextField()`
  - `prevField()`
  - `updateFocus()`
  - `blurAllFields()`
- Field manipulation:
  - `clearAllFields()`
  - `clearCurrentField()`
  - `initializeInputFields()`
- Focus state management:
  - `initErrorFocus()`
  - `restorePreviousFocus()`

**Dependencies**: Requires access to textinput, textarea models

### 3. `create_rendering.go` (UI Rendering)
**Target Size**: ~700 lines  
**Responsibilities**: All visual output and layout management

**Contents**:
- Main view:
  - `View() string` - primary render method
- Layout renderers:
  - `renderSinglePanelLayout(availableHeight int) string`
  - `renderCompactForm(width, height int) string`
  - `renderErrorLayout(availableHeight int) string`
- Modal/Overlay renderers:
  - `renderFileSelectionModal(statusBar string) string`
  - `renderWithFZFOverlay(baseView string) string`
  - `renderOverlayDropdown(baseView, overlayView string, yOffset, xOffset int) string`
- Component renderers:
  - `renderFieldIndicator() string`
  - `renderFileInputMode() string`
  - `renderStatusBar() string`

**Dependencies**: Heavy use of lipgloss for styling

### 4. `create_submission.go` (Task Processing)
**Target Size**: ~400 lines  
**Responsibilities**: Task preparation, validation, and API submission

**Contents**:
- Submission methods:
  - `submitRun() tea.Cmd`
  - `submitWithForce() tea.Cmd`
  - `submitToAPI(task models.RunRequest) (models.RunResponse, error)`
  - `submitToAPIWithForce(task models.RunRequest) (models.RunResponse, error)`
- Task preparation:
  - `prepareTask() (models.RunRequest, error)`
  - `prepareTaskFromForm() models.RunRequest`
  - `prepareTaskFromFile(filePath string) (models.RunRequest, error)`
- Validation:
  - `validateTask(task *models.RunRequest) error`
  - `validateForm() (bool, string)`
- Git integration:
  - `autoDetectGitInfo(task *models.RunRequest)`
- Duplicate detection:
  - `loadFileHashCache() tea.Cmd`
- Message handler:
  - `handleRunCreated(msg runCreatedMsg) (tea.Model, tea.Cmd)`

**Dependencies**: API client, models, git utilities

### 5. `create_config.go` (Configuration Management)
**Target Size**: ~250 lines  
**Responsibilities**: Config file operations and form data persistence

**Contents**:
- Config file operations:
  - `loadConfigFromFile(filePath string) tea.Cmd`
  - `populateFormFromConfig(config *models.RunRequest, filePath string)`
  - `activateConfigFileSelector() tea.Cmd`
- Form data persistence:
  - `loadFormData()`
  - `saveFormData()`
- Cache management:
  - Form data caching logic
  - Config file history

**Dependencies**: config.ConfigLoader, cache, file I/O

### 6. `create_repository.go` (Repository Selection)
**Target Size**: ~150 lines  
**Responsibilities**: Repository selection and FZF integration

**Contents**:
- Repository operations:
  - `selectRepository() tea.Cmd`
  - `autofillRepository()`
  - `handleRepositorySelected(msg repositorySelectedMsg) (tea.Model, tea.Cmd)`
- FZF mode:
  - `activateFZFMode()`
  - FZF-specific repository selection logic

**Dependencies**: components.RepositorySelector, FZFMode, git utilities

### 7. `create_messages.go` (Messages & Helpers)
**Target Size**: ~80 lines  
**Responsibilities**: Message types and utility functions

**Contents**:
- Message type definitions:
  - `type runCreatedMsg struct`
  - `type repositorySelectedMsg struct`
  - `type clipboardResultMsg struct`
  - `type configLoadedMsg struct`
  - `type configLoadErrorMsg struct`
  - `type fileSelectorActivatedMsg struct`
  - `type configSelectorTickMsg time.Time`
- Helper functions:
  - `min(a, b int) int`
  - `max(a, b int) int`
- Animation/Timer commands:
  - `startYankBlinkAnimation() tea.Cmd`
  - `startClearStatusTimer() tea.Cmd`
  - `tickCmd() tea.Cmd`
- Clipboard:
  - `copyToClipboard(text string) tea.Cmd`

**Dependencies**: Minimal - mostly pure functions

## Implementation Strategy

### Phase 1: Setup (Day 1)
1. Create all new empty files with package declaration and imports
2. Add file headers with clear documentation of purpose
3. Set up basic structure in each file

### Phase 2: Move Helper Components (Day 1)
1. Start with `create_messages.go` - move all message types and helpers
2. Move repository logic to `create_repository.go`
3. Ensure no compilation errors after each move

### Phase 3: Extract Major Components (Day 2-3)
1. Move config management to `create_config.go`
2. Extract submission logic to `create_submission.go`
3. Move rendering methods to `create_rendering.go`
4. Extract input handling to `create_input.go`

### Phase 4: Cleanup & Testing (Day 4)
1. Clean up `create.go` to contain only core logic
2. Remove any duplicate code
3. Optimize imports in each file
4. Run all tests to ensure nothing broke
5. Add/update tests for new structure

### Phase 5: Documentation & Polish (Day 5)
1. Update code comments
2. Ensure consistent naming
3. Add package-level documentation
4. Run linting and formatting

## Testing Plan

### Unit Tests
- Ensure existing tests continue to pass
- Update test imports as needed
- Consider adding focused tests for each new file

### Integration Tests
- Test complete create flow works end-to-end
- Verify all keyboard shortcuts function correctly
- Test error handling paths

### Manual Testing Checklist
- [ ] Create new run via form
- [ ] Create new run via file
- [ ] Repository selection with FZF
- [ ] Config file loading
- [ ] Form validation
- [ ] Error display and recovery
- [ ] All keyboard shortcuts (vim and standard)
- [ ] Submit and force submit
- [ ] Field navigation
- [ ] Clipboard operations

## Migration Steps

### Step-by-step Commands
```bash
# 1. Create new files
touch internal/tui/views/create_messages.go
touch internal/tui/views/create_repository.go
touch internal/tui/views/create_config.go
touch internal/tui/views/create_submission.go
touch internal/tui/views/create_rendering.go
touch internal/tui/views/create_input.go

# 2. After moving code, verify compilation
go build ./...

# 3. Run tests
go test ./internal/tui/views/...

# 4. Check for any circular dependencies
go list -f '{{.ImportPath}} -> {{join .Imports " "}}' ./internal/tui/views/... | grep create

# 5. Run linting
make lint-fix

# 6. Format code
make fmt
```

## Success Criteria

### Quantitative
- [ ] No file exceeds 700 lines
- [ ] create.go reduced to under 400 lines
- [ ] All tests pass without modification
- [ ] No performance regression
- [ ] Zero circular dependencies

### Qualitative
- [ ] Each file has a single, clear responsibility
- [ ] Code is easier to navigate and understand
- [ ] New developers can quickly find relevant code
- [ ] Easier to add new features
- [ ] Improved testability of individual components

## Risks & Mitigations

### Risk 1: Breaking Existing Functionality
**Mitigation**: 
- Incremental moves with testing after each step
- Keep backup of original file
- Use git commits for each major move

### Risk 2: Circular Dependencies
**Mitigation**:
- Plan file dependencies carefully
- Keep message types in separate file
- Use interfaces where needed

### Risk 3: Import Management Complexity
**Mitigation**:
- Use goimports to manage imports automatically
- Group imports logically
- Remove unused imports immediately

## Future Improvements

After this refactoring, consider:
1. Extract common TUI patterns into shared components
2. Create interfaces for better testability
3. Consider using more Bubble Tea components (viewport, spinner)
4. Implement more granular caching strategies
5. Add performance metrics for slow operations

## Notes

- Follow existing patterns from dashboard and details views
- Maintain consistency with codebase conventions
- Keep methods on CreateRunView receiver for state access
- Avoid creating unnecessary abstractions
- Focus on clarity over cleverness

## Timeline

**Estimated Duration**: 1 week (5 working days)
- Day 1: Setup and helper components
- Day 2-3: Major component extraction
- Day 4: Testing and bug fixes
- Day 5: Documentation and polish

## References

- Existing split views: `dashboard*.go`, `details*.go`
- Go best practices: https://go.dev/doc/effective_go
- Bubble Tea patterns: https://github.com/charmbracelet/bubbletea/tree/master/examples