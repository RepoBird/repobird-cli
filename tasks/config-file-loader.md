# Config File Loader Implementation Plan

## Overview
Add comprehensive config file loading functionality to the RepoBird CLI's TUI create run view, allowing users to load run configurations from JSON files using an intuitive FZF-based file selector.

## Requirements Analysis

### User Requirements
- **Load from Config** button as the first row in create run form
- FZF autocompletion for .json files in current directory (with depth limit)
- Support for entering full paths or navigating deeper folders
- Ignore patterns for common folders (node_modules, build/, etc.)
- Focus on JSON format initially
- Maintain existing manual form functionality

### Technical Requirements
- Integrate with existing FZF infrastructure
- Support flexible JSON config format with required/optional fields
- Graceful error handling for malformed files
- Preserve form state and user experience
- Performance optimization for file discovery

## Config File Format Specification

### Supported JSON Structure
```json
{
  // Required fields (at least one repository identifier)
  "prompt": "Description of what the AI should do",
  
  // Repository identification (flexible - use any ONE of these)
  "repository": "owner/repo",           // Primary field (string)
  "repositoryName": "owner/repo",       // Alternative field name
  "repoId": 12345,                     // Alternative (integer ID)
  
  // Optional fields
  "source": "main",                    // Source branch
  "target": "feature/branch-name",     // Target branch  
  "runType": "run",                    // "run" | "plan" | "approval"
  "title": "Brief description",        // Run title
  "context": "Additional context",     // Extra context
  "files": ["src/file1.js", "src/file2.js"] // File list
}
```

### Field Validation Rules
1. **prompt**: Required string, non-empty
2. **Repository**: Must have at least one of:
   - `repository` (string format: "owner/repo")  
   - `repositoryName` (string format: "owner/repo")
   - `repoId` (positive integer)
3. **runType**: Optional enum ("run", "plan", "approval"), defaults to "run"
4. **source/target**: Optional strings
5. **files**: Optional array of strings
6. **title/context**: Optional strings

### Priority Order for Repository Fields
1. `repoId` (if present and valid integer)
2. `repository` (if present and valid string)
3. `repositoryName` (if present and valid string)

## Architecture Design

### New Components

#### 1. File Discovery System (`internal/utils/file_discovery.go`)
```go
type FileDiscoveryOptions struct {
    MaxDepth        int
    IgnorePatterns  []string
    FileExtensions  []string
    SortByModTime   bool
}

type FileDiscovery interface {
    FindFiles(rootPath string, opts FileDiscoveryOptions) ([]string, error)
    GetFileInfo(path string) (os.FileInfo, error)
}
```

**Features:**
- Recursive directory traversal with depth limiting (default: 3 levels)
- Configurable ignore patterns
- File extension filtering
- Sort by modification time or alphabetical
- Relative path resolution

**Default Ignore Patterns:**
```go
var DefaultIgnorePatterns = []string{
    "node_modules", ".git", "build", "dist", "target", 
    "bin", ".cache", ".next", ".vscode", ".idea", 
    "vendor", "__pycache__", ".nuxt", ".output"
}
```

#### 2. Config File Loader (`internal/config/loader.go`)
```go
type ConfigLoader interface {
    LoadConfig(filePath string) (*models.RunRequest, error)
    ValidateConfig(config *models.RunRequest) []error
    NormalizeRepository(config *models.RunRequest) error
}
```

**Features:**
- JSON parsing with detailed error messages
- Field validation and normalization
- Repository field resolution (repoId vs repository vs repositoryName)
- Partial loading support (populate valid fields, report errors)

#### 3. FZF File Selector (`internal/tui/components/file_selector.go`)
```go
type FileSelector struct {
    discovery    FileDiscovery
    fzfMode      *FZFMode
    currentPath  string
    recentFiles  []string
}

type FileSelectorOptions struct {
    MaxDepth       int
    IgnorePatterns []string
    ShowPreview    bool
    AllowManualEntry bool
}
```

**Features:**
- FZF-based file selection with fuzzy matching
- Recent files history
- Manual path entry support
- File preview (show first few lines)
- Keyboard navigation (arrows, Enter, Escape)

### Modified Components

#### 1. Create Run View (`internal/tui/views/create.go`)

**Focus Index Changes:**
- 0: Load from Config (new)
- 1: Run Type (was 0)
- 2: Repository (was 1)  
- 3: Prompt (was 2)
- 4+: Remaining fields (shifted down)

**New Fields/Methods:**
```go
type CreateRunView struct {
    // ... existing fields ...
    
    // Config loading
    configLoader    *ConfigLoader
    fileSelector    *FileSelector
    loadConfigFocused bool
    lastLoadedFile    string
    
    // Replace existing file input mode
    // useFileInput bool // Remove this
    // filePathInput textinput.Model // Remove this
}

// New methods
func (v *CreateRunView) activateFileSelector()
func (v *CreateRunView) loadConfigFromFile(filePath string) 
func (v *CreateRunView) populateFormFromConfig(config *models.RunRequest)
func (v *CreateRunView) handleConfigLoadError(err error)
```

## Implementation Phases

### Phase 1: Core Infrastructure
1. **File Discovery Implementation**
   - Create `internal/utils/file_discovery.go`
   - Implement recursive file search with ignore patterns
   - Add unit tests for various directory structures
   - Performance testing with large directories

2. **Config Loader Implementation**  
   - Create `internal/config/loader.go`
   - JSON parsing with validation
   - Repository field normalization logic
   - Comprehensive error handling
   - Unit tests for valid/invalid configs

3. **Basic Integration**
   - Add "Load from Config" field to create.go
   - Update focus navigation system
   - Basic file loading without FZF (manual path entry)
   - Update status bar help text

### Phase 2: FZF File Selection
1. **File Selector Component**
   - Create `internal/tui/components/file_selector.go`
   - Integrate with existing FZF framework
   - File list generation and filtering
   - Manual path entry support

2. **UI Integration**
   - Connect file selector to create view
   - FZF dropdown overlay positioning
   - Keyboard event handling
   - Visual feedback for file selection

3. **Form Population**
   - Load config data into form fields
   - Validation error display
   - Partial loading support
   - Success/error status messages

### Phase 3: Enhanced Features
1. **Recent Files History**
   - Cache recently loaded config files
   - Show recent files at top of FZF list
   - Persistent storage in user config

2. **File Preview**
   - Show file content preview in FZF
   - Validation status indicators
   - File size and modification time display

3. **Advanced Error Handling**
   - Detailed JSON parsing error messages
   - Field-level validation errors
   - Recovery suggestions for common issues

## User Experience Flow

### Primary Workflow
1. **Navigation**: User navigates to "Load from Config" field (focus index 0)
2. **Activation**: Press Enter → FZF file selector opens
3. **Selection**: User types to filter or uses arrows to navigate files
4. **Loading**: Select file → config loads and populates form fields  
5. **Editing**: User can edit any populated field as normal
6. **Submission**: Submit run with loaded + edited data

### Alternative Workflows
- **Manual Path**: In FZF, type full file path instead of selecting
- **Navigation**: Use FZF to browse into subdirectories
- **Error Handling**: If load fails, stay in form with error message
- **Partial Load**: If some fields invalid, load valid ones and show errors

## Error Handling Strategy

### File System Errors
- **File not found**: "Config file not found: {path}"
- **Permission denied**: "Cannot read config file: {path} (permission denied)"
- **Directory not accessible**: "Cannot access directory: {path}"

### JSON Parsing Errors  
- **Invalid JSON**: "Invalid JSON format at line {line}, column {col}: {error}"
- **Empty file**: "Config file is empty: {path}"
- **Wrong format**: "Expected JSON object, got {type}"

### Validation Errors
- **Missing prompt**: "Required field 'prompt' is missing or empty"
- **Missing repository**: "Must specify one of: 'repository', 'repositoryName', or 'repoId'"
- **Invalid runType**: "Invalid runType '{value}', must be one of: run, plan, approval"
- **Invalid repository format**: "Repository must be in format 'owner/repo'"

### Display Strategy
- Show errors in status line with error color
- Keep form populated with valid fields
- Allow user to correct errors and retry
- Provide "Copy error" functionality for troubleshooting

## Testing Strategy

### Unit Tests
1. **File Discovery**
   - Test ignore patterns effectiveness
   - Test depth limiting
   - Test file extension filtering
   - Test edge cases (empty dirs, symlinks)

2. **Config Loader**
   - Test valid JSON parsing
   - Test invalid JSON handling  
   - Test field validation rules
   - Test repository field priority

3. **File Selector**
   - Test FZF integration
   - Test manual path entry
   - Test recent files functionality
   - Test keyboard navigation

### Integration Tests
1. **Form Population**
   - Test config loading into form fields
   - Test partial loading with errors
   - Test focus navigation after loading

2. **End-to-End Workflows**
   - Test complete config load → edit → submit flow
   - Test error recovery workflows
   - Test FZF file selection workflows

### Test Data Structure
```
testdata/
├── valid/
│   ├── minimal.json          # Only required fields
│   ├── complete.json         # All fields populated
│   ├── alternative-names.json # repositoryName instead of repository
│   └── repo-id.json          # repoId instead of repository
├── invalid/
│   ├── malformed.json        # Invalid JSON syntax
│   ├── missing-prompt.json   # Missing required field
│   ├── missing-repo.json     # No repository identifier
│   └── invalid-runtype.json  # Invalid enum value
└── directories/
    ├── nested/
    │   └── deep/
    │       └── config.json
    ├── node_modules/         # Should be ignored
    └── build/               # Should be ignored
```

## Performance Considerations

### File Discovery Optimization
- **Depth Limiting**: Default max depth of 3 levels
- **Early Termination**: Stop searching when ignore pattern matches
- **Async Loading**: Use goroutines for directory traversal
- **Caching**: Cache file lists for recently accessed directories

### Memory Management  
- **Streaming**: Don't load large files entirely into memory
- **Lazy Loading**: Only load file content when selected
- **Cleanup**: Clean up FZF resources when not in use

### User Experience
- **Responsive UI**: Show loading indicators for slow operations
- **Interrupt Handling**: Allow cancellation of long file searches
- **Progressive Loading**: Show files as they're discovered

## Documentation Updates

### User Documentation
1. **CLAUDE.md Updates**
   - Add config file format section
   - Add "Load from Config" usage instructions
   - Update example JSON configs

2. **CLI Help Text**
   - Update create command help
   - Add config file examples
   - Add troubleshooting tips

### Developer Documentation
1. **Architecture docs**
   - Document new components
   - Update TUI flow diagrams
   - Add config loading sequence diagrams

2. **API Documentation**  
   - Document config file format
   - Add validation rules
   - Add error code references

## Migration Strategy

### Backward Compatibility
- Keep existing manual form functionality unchanged
- Maintain existing file input mode as fallback
- Preserve all existing keyboard shortcuts and workflows

### Gradual Rollout
1. **Phase 1**: Add basic config loading (no FZF)
2. **Phase 2**: Add FZF file selection
3. **Phase 3**: Add enhanced features (preview, recent files)
4. **Phase 4**: Remove old file input mode (optional)

### User Migration
- Add help text explaining new functionality
- Show example config files in documentation
- Provide migration guide from manual entry to config files

## Success Criteria

### Functional Requirements
- [ ] Users can load JSON config files via FZF selector
- [ ] Form populates correctly from valid config files
- [ ] Error handling works for invalid files/JSON
- [ ] File discovery respects ignore patterns and depth limits
- [ ] Integration preserves existing TUI functionality

### Performance Requirements
- [ ] File discovery completes in <2 seconds for typical projects
- [ ] FZF selector is responsive with 100+ files
- [ ] Config loading completes in <500ms for typical files
- [ ] Memory usage stays reasonable during file operations

### User Experience Requirements  
- [ ] Intuitive navigation and keyboard shortcuts
- [ ] Clear error messages and recovery options
- [ ] Visual feedback for loading operations
- [ ] Consistent with existing TUI patterns
- [ ] Comprehensive help text and documentation

## Risk Assessment

### Technical Risks
1. **Performance**: Large directories might slow down file discovery
   - *Mitigation*: Depth limits, ignore patterns, async loading

2. **Compatibility**: FZF integration might conflict with existing components
   - *Mitigation*: Reuse existing FZF framework, thorough testing

3. **Error Handling**: Complex JSON validation might be fragile
   - *Mitigation*: Comprehensive test coverage, graceful fallbacks

### User Experience Risks
1. **Discoverability**: Users might not notice new functionality
   - *Mitigation*: Clear UI placement, documentation, help text

2. **Complexity**: Too many options might confuse users  
   - *Mitigation*: Progressive disclosure, sensible defaults

3. **Migration**: Existing users might resist workflow changes
   - *Mitigation*: Keep existing workflows unchanged, additive approach

## Future Enhancements

### Short Term (Next Sprint)
- **Config File Creation**: Save current form as config file
- **Templates**: Built-in config templates for common use cases
- **Validation Preview**: Show validation status before loading

### Medium Term (Next Quarter)  
- **Multi-format Support**: YAML, TOML config file support
- **Remote Configs**: Load config files from URLs or git repos
- **Config Management**: Organize and manage multiple config files

### Long Term (Future)
- **Visual Config Editor**: GUI for creating/editing config files
- **Config Sharing**: Share config files with team members
- **Integration**: IDE plugins for config file management

---

## Implementation Checklist

### Core Infrastructure
- [ ] Create `internal/utils/file_discovery.go`
- [ ] Create `internal/config/loader.go`  
- [ ] Add unit tests for file discovery
- [ ] Add unit tests for config loading
- [ ] Add integration tests

### UI Integration
- [ ] Create `internal/tui/components/file_selector.go`
- [ ] Modify `internal/tui/views/create.go` 
- [ ] Update focus navigation system
- [ ] Add FZF file selection overlay
- [ ] Update status bar help text

### Features  
- [ ] Add form population from config
- [ ] Add error handling and display
- [ ] Add recent files functionality
- [ ] Add manual path entry support
- [ ] Add config validation feedback

### Testing & Documentation
- [ ] Create comprehensive test suite
- [ ] Add testdata directory structure  
- [ ] Update CLAUDE.md documentation
- [ ] Update CLI help text
- [ ] Add troubleshooting guide
- [ ] Performance testing and optimization

### Final Integration
- [ ] End-to-end testing
- [ ] User acceptance testing  
- [ ] Documentation review
- [ ] Code review and cleanup
- [ ] Final lint and format pass