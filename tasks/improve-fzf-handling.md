# Task: Improve FZF Integration in RepoBird CLI

## Overview
Enhance the RepoBird CLI with FZF (fuzzy finder) integration to provide interactive selection capabilities for various commands, improving user experience and workflow efficiency.

## Current State
- No FZF integration currently exists
- Users must manually type or copy/paste run IDs
- No interactive selection for tasks, branches, or repositories
- Command-line experience requires exact input

## Requirements

### 1. Core FZF Integration

#### Binary Detection
- Detect if FZF is installed on the system
- Provide graceful fallback when FZF is not available
- Option to use embedded FZF library or external binary
- Clear messaging when FZF features are unavailable

#### Integration Points
1. **Run ID Selection**
   - When no run ID provided to `status` command
   - Interactive selection from recent runs
   - Show run details in preview pane

2. **Task File Selection**
   - When no task file provided to `run` command
   - Browse and select from .json files
   - Preview task content before selection

3. **Branch Selection**
   - Interactive branch picker for source/target
   - Show commit counts and last activity
   - Filter local/remote branches

4. **Repository Selection**
   - Select from recently used repositories
   - Browse organization repositories
   - Search by name or description

### 2. Implementation Approach

#### External Binary Integration
```go
// internal/utils/fzf/fzf.go
package fzf

import (
    "os/exec"
    "strings"
)

type FZF struct {
    available bool
    path      string
}

func New() *FZF {
    path, err := exec.LookPath("fzf")
    return &FZF{
        available: err == nil,
        path:      path,
    }
}

func (f *FZF) IsAvailable() bool {
    return f.available
}

func (f *FZF) Select(items []string, opts ...Option) (string, error) {
    if !f.available {
        return "", ErrFZFNotAvailable
    }
    
    // Build fzf command with options
    cmd := exec.Command(f.path, buildArgs(opts)...)
    
    // Pipe items to stdin
    stdin, _ := cmd.StdinPipe()
    for _, item := range items {
        stdin.Write([]byte(item + "\n"))
    }
    stdin.Close()
    
    // Get selection from stdout
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    
    return strings.TrimSpace(string(output)), nil
}
```

#### Embedded Library Alternative
```go
// Use github.com/ktr0731/go-fuzzyfinder as fallback
import "github.com/ktr0731/go-fuzzyfinder"

func (f *FZF) SelectWithLibrary(items []interface{}, opts ...Option) (interface{}, error) {
    idx, err := fuzzyfinder.Find(
        items,
        func(i int) string {
            // Extract display string from item
            return formatItem(items[i])
        },
        fuzzyfinder.WithPreviewWindow(func(i, w, h int) string {
            // Generate preview content
            return generatePreview(items[i], w, h)
        }),
    )
    
    if err != nil {
        return nil, err
    }
    
    return items[idx], nil
}
```

### 3. Feature Implementations

#### Inline Run ID Selection
```go
// internal/commands/status.go
func getRunID(args []string) (string, error) {
    if len(args) > 0 {
        return args[0], nil
    }
    
    // Try FZF selection
    fzf := fzf.New()
    if fzf.IsAvailable() {
        runs, err := apiClient.ListRuns()
        if err != nil {
            return "", err
        }
        
        items := formatRunsForFZF(runs)
        selection, err := fzf.Select(items, 
            fzf.WithPrompt("Select a run: "),
            fzf.WithPreview(true),
            fzf.WithHeader("ID | Status | Repository | Created"),
        )
        
        if err != nil {
            return "", err
        }
        
        return extractRunID(selection), nil
    }
    
    // Fallback to listing runs
    return "", fmt.Errorf("no run ID provided, use --list to see available runs")
}
```

#### Task File Selection
```go
// internal/commands/run.go
func selectTaskFile() (string, error) {
    fzf := fzf.New()
    if !fzf.IsAvailable() {
        return "", fmt.Errorf("FZF not available for interactive selection")
    }
    
    // Find all JSON files
    files, err := findTaskFiles()
    if err != nil {
        return "", err
    }
    
    if len(files) == 0 {
        return "", fmt.Errorf("no task files found")
    }
    
    // FZF with preview
    selected, err := fzf.Select(files,
        fzf.WithPrompt("Select task file: "),
        fzf.WithPreviewCommand("cat {}"),
        fzf.WithPreviewWindow("right:60%"),
    )
    
    return selected, err
}

func findTaskFiles() ([]string, error) {
    var files []string
    
    // Search common locations
    patterns := []string{
        "*.json",
        "tasks/*.json",
        ".repobird/*.json",
    }
    
    for _, pattern := range patterns {
        matches, _ := filepath.Glob(pattern)
        files = append(files, matches...)
    }
    
    return files, nil
}
```

#### Branch Selection
```go
// internal/utils/git/branches.go
func SelectBranch(prompt string) (string, error) {
    fzf := fzf.New()
    if !fzf.IsAvailable() {
        return "", ErrFZFRequired
    }
    
    branches, err := listBranches()
    if err != nil {
        return "", err
    }
    
    // Format branches with additional info
    items := make([]string, len(branches))
    for i, branch := range branches {
        items[i] = formatBranchInfo(branch)
    }
    
    selected, err := fzf.Select(items,
        fzf.WithPrompt(prompt),
        fzf.WithPreviewCommand("git log --oneline -n 20 {1}"),
        fzf.WithHeader("Branch | Last Commit | Author | Age"),
    )
    
    if err != nil {
        return "", err
    }
    
    // Extract branch name from selection
    return parseBranchName(selected), nil
}
```

### 4. Configuration Options

Add FZF-specific configuration:

```yaml
# ~/.repobird/config.yaml
fzf:
  enabled: true
  use_external: true  # Use system fzf if available
  default_opts: "--height 40% --layout reverse --border"
  preview:
    enabled: true
    position: "right:50%"
  keybindings:
    select: "enter"
    abort: "esc"
    preview_up: "ctrl-u"
    preview_down: "ctrl-d"
```

### 5. CLI Command Integration

#### Status Command
```bash
# Without FZF (current behavior)
repobird status RUN_ID

# With FZF (when no ID provided)
repobird status
# Opens FZF selector with all runs

# Force FZF selection
repobird status --select
```

#### Run Command
```bash
# Without FZF (current behavior)
repobird run task.json

# With FZF (when no file provided)
repobird run
# Opens FZF file selector

# With FZF for branches
repobird run task.json --select-branch
# Opens FZF for source/target branch selection
```

#### New Selection Command
```bash
# Generic selection command
repobird select run       # Select from runs
repobird select task      # Select from task files
repobird select branch    # Select from branches
repobird select repo      # Select from repositories
```

### 6. Preview Pane Content

#### Run Preview
```
Run ID: abc-123-def
Status: running
Repository: org/repo
Branch: feature/new-feature
Started: 2024-01-15 10:30:00
Duration: 5m 23s

Prompt:
Fix the authentication bug in the login flow

Last Output:
Analyzing codebase...
Found 3 files to modify...
```

#### Task File Preview
```json
{
  "prompt": "Fix authentication bug",
  "repository": "org/repo",
  "source": "main",
  "target": "fix/auth-bug",
  "runType": "run",
  "context": "Users cannot login"
}
```

### 7. Error Handling

- Graceful degradation when FZF not available
- Clear error messages with installation instructions
- Fallback to non-interactive mode
- Handle FZF cancellation (ESC key)
- Validate selections before use

### 8. Installation Instructions

When FZF not detected, provide helpful message:

```
FZF not found. Interactive selection features are disabled.

To install FZF:
- macOS: brew install fzf
- Linux: sudo apt-get install fzf (Ubuntu/Debian)
         sudo dnf install fzf (Fedora)
- Windows: choco install fzf

Or download from: https://github.com/junegunn/fzf

Alternatively, use --no-fzf flag to suppress this message.
```

## Implementation Plan

### Phase 1: Core Infrastructure (Day 1)
- [ ] Create FZF detection and wrapper
- [ ] Implement basic selection functionality
- [ ] Add configuration support
- [ ] Handle missing FZF gracefully

### Phase 2: Run Selection (Day 2)
- [ ] Integrate with status command
- [ ] Add run preview formatting
- [ ] Implement run filtering
- [ ] Add tests

### Phase 3: File Selection (Day 3)
- [ ] Add task file discovery
- [ ] Integrate with run command
- [ ] Add JSON preview
- [ ] Implement file filtering

### Phase 4: Advanced Features (Day 4)
- [ ] Branch selection integration
- [ ] Repository selection
- [ ] Custom keybindings
- [ ] Performance optimization

### Phase 5: Polish (Day 5)
- [ ] Documentation
- [ ] Installation guide
- [ ] Examples and tutorials
- [ ] Integration tests

## Testing Strategy

### Unit Tests
- FZF availability detection
- Command building
- Output parsing
- Fallback behavior

### Integration Tests
- End-to-end selection flows
- Preview generation
- Error scenarios
- Performance with large lists

### Manual Testing
- Test on systems with/without FZF
- Various terminal emulators
- Different color schemes
- SSH sessions

## Success Criteria

- [ ] FZF integration works when available
- [ ] Graceful fallback when FZF missing
- [ ] Improved UX for selection tasks
- [ ] No breaking changes to existing commands
- [ ] Performance acceptable with 1000+ items
- [ ] Documentation complete
- [ ] Tests passing with good coverage

## Future Enhancements

1. **Multi-select Support**
   - Select multiple runs for batch operations
   - Choose multiple files for analysis

2. **Smart Sorting**
   - Most recently used items first
   - Frecency algorithm for better predictions

3. **Custom Actions**
   - Define actions for selected items
   - Keyboard shortcuts for common operations

4. **Integration with TUI**
   - Use FZF within TUI mode
   - Seamless switching between modes

5. **Theme Support**
   - Match FZF theme with terminal
   - Custom color schemes

## Dependencies

- Optional: `fzf` binary (>= 0.40.0 recommended)
- Optional: `github.com/ktr0731/go-fuzzyfinder` for embedded support
- Existing: Command execution capabilities

## Notes

- FZF should be optional, never required
- Maintain backward compatibility
- Consider Windows compatibility (use fzf.exe)
- Performance important for large result sets
- Preview content should be informative but concise