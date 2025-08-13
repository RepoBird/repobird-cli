# Terminal User Interface (TUI) Guide

The RepoBird CLI includes a rich terminal user interface for managing runs, viewing status, and creating new tasks interactively.

## Launching the TUI

```bash
repobird tui
```

## Dashboard Overview

The dashboard uses a Miller Columns layout with three main sections:

1. **Repositories Column** (Left)
   - Lists all repositories with active runs
   - Shows status indicators for each repository
   - Displays run counts

2. **Runs Column** (Middle)
   - Shows runs for the selected repository
   - Displays run status, title, and timing
   - Color-coded status indicators

3. **Details Column** (Right)
   - Shows detailed information for selected run
   - Displays prompt, context, and output
   - Scrollable content for long outputs


## Navigation

The TUI uses a message-based navigation architecture that provides consistent behavior across all views.

### Basic Navigation
- **Tab** - Cycle through columns
- **↑↓** or **j/k** - Move up/down in current column
- **←→** or **h/l** - Move between columns
- **Enter** - Select item and move to next column
- **Backspace** - Move to previous column

### Navigation Architecture

The TUI implements a clean navigation system with the following components:

#### App Router
- Central navigation controller handling all view transitions
- Maintains navigation history stack for back navigation
- Manages shared context between views

#### Navigation Messages
All navigation is handled through type-safe messages rather than direct view creation:

- `NavigateToCreateMsg` - Navigate to create run view
- `NavigateToDetailsMsg` - Navigate to run details view  
- `NavigateToListMsg` - Navigate to run list view
- `NavigateToBulkMsg` - Navigate to bulk operations view
- `NavigateBackMsg` - Navigate to previous view in stack
- `NavigateToDashboardMsg` - Navigate to dashboard (clears stack)

#### View Constructor Pattern
All views use minimal constructors with shared cache:

```go
// Clean minimal constructor (max 3 params)
NewView(client APIClient, cache *cache.SimpleCache, id string)

// Views are self-loading in Init()
func (v *View) Init() tea.Cmd {
    return v.loadOwnData() // No parent state coupling
}
```

#### Shared Components
The TUI uses reusable components across views:
- `ScrollableList` - Multi-column scrollable lists with navigation
- `Form` - Input forms with validation and navigation  
- `StatusLine` - Unified status bar with loading and data info
- `KeyMap` - Standard keybindings for consistent behavior
- `WindowLayout` - Global layout system for consistent sizing and borders

#### View History
- **Back navigation** - Press `q`, `ESC`, or `b` to go back
- **Navigation stack** - Views are pushed onto a stack for history
- **Dashboard reset** - Pressing `d` clears history and returns to dashboard
- **Error recovery** - Recoverable errors allow going back, non-recoverable clear history

### Fuzzy Search (FZF Mode)

The TUI includes powerful fuzzy search capabilities for quick navigation:

#### Activating FZF Mode
- Press **f** on any column in the dashboard
- The FZF dropdown appears at the current cursor position
- Start typing to filter items in real-time

#### FZF Navigation
- **↑↓** or **Ctrl+j/k** - Navigate filtered items
- **Enter** - Select item and move to next column
- **ESC** - Cancel FZF mode
- Type to filter - Fuzzy matching on item text

#### FZF Behavior by Column
- **Repository Column**: Filters repositories, selecting moves to runs
- **Runs Column**: Filters runs, selecting moves to details
- **Details Column**: Filters detail lines for quick navigation

### View Controls
- **n** - Create new run
- **s** - Show status/user info overlay
- **r** - Refresh data
- **?** - Toggle help
- **q** - Go back to parent view (or quit from dashboard)
- **Q** - Force quit from anywhere

### Navigation Hierarchy
The TUI follows a consistent navigation pattern:
- **q** (lowercase) - Always goes back to the parent view
  - From Details → Dashboard/List
  - From Create Run → Dashboard
  - From Status View → Dashboard
  - From Dashboard → Quit (top-level)
- **Q** (uppercase/Shift+Q) - Force quit from any view
- **ESC** or **b** - Alternative ways to go back

### Clipboard Operations
- **y** - Copy current selection/field value to clipboard
- **Y** - Copy all content (in details view)
- Navigation works on all selectable fields

## Create Run View

The create run view provides an interactive form for submitting new tasks.

### Form Fields
1. **Title** - Brief description of the task
2. **Repository** - Target repository (org/repo format)
3. **Source Branch** - Base branch for changes
4. **Target Branch** - Branch for the pull request
5. **Issue** - Related issue number (optional)
6. **Prompt** - Detailed task description
7. **Context** - Additional context (optional)

### Input Modes

#### Insert Mode (Default)
- Active text input in fields
- **Tab/Shift+Tab** - Navigate between fields
- **ESC** - Switch to normal mode
- **Ctrl+S** - Submit run

#### Normal Mode
- Vim-style navigation
- **i** or **Enter** - Enter insert mode
- **j/k** - Navigate fields
- **ESC** - Exit (press twice to return to dashboard)

### Repository Selection with FZF

The repository field supports fuzzy search for quick selection:

#### In Insert Mode
- **Ctrl+F** - Activate FZF when on repository field
- **Ctrl+R** - Browse repository history (alternative selector)

#### In Normal Mode
- **f** - Activate FZF when repository field is focused

#### Repository FZF Features
- Shows current git repository (📁 icon)
- Lists recently used repositories (🔄 icon)
- Displays manually entered repositories (✏️ icon)
- Real-time fuzzy filtering
- Auto-saves selection to history

### Keyboard Shortcuts

#### Global
- **Ctrl+S** - Submit run
- **Ctrl+L** - Clear all fields
- **Ctrl+X** - Clear current field
- **?** - Toggle help

#### File Input Mode
- **Ctrl+F** - Toggle between file input and form input
- Allows loading task configuration from JSON file

### Duplicate Run Detection

When you load a task file, RepoBird automatically detects duplicates using file hashing:

#### Visual Indicators
- ✓ **Green checkmark** next to Submit button: Ready to submit (unique task)
- ⚠️ **Yellow warning** next to Submit button: Duplicate detected

#### Submission Behavior
When submitting a duplicate task, instead of showing an error:
1. **Friendly Prompt**: Yellow status bar appears: `[DUPLICATE] ⚠️ DUPLICATE RUN DETECTED (ID: 123) - Override? [y] yes [n] no`
2. **Easy Override**: Press `y` to automatically retry with override, or `n` to cancel
3. **No Error Page**: Clean user experience without confusing error messages

#### File Type Support
Works with any file type:
- **JSON** task files (`.json`)
- **YAML** configuration files (`.yaml`, `.yml`) 
- **Markdown** documentation (`.md`)
- **Any file type** - calculates SHA-256 hash of content

## Run Details View

The details view provides comprehensive information about a selected run with enhanced navigation:

### Field Navigation
- **j/k** or **↑↓** - Navigate between selectable fields
- **g** - Jump to first field
- **G** - Jump to last field
- **y** - Copy selected field value to clipboard
- **Y** - Copy all content

### Multi-line Field Handling
- Multi-line fields (Plan, Prompt, Context, Error) are treated as single selectable units
- The entire field is highlighted when selected
- Pressing **y** copies the complete multi-line content
- Navigation automatically scrolls to keep selected fields visible

### Features
- Row-based navigation for all selectable fields
- Smart highlighting that spans multiple lines for multi-line content
- Visual feedback when copying (green flash animation)
- Automatic scrolling to keep selections in view
- **q** returns to dashboard, **Q** force quits

## Status Info Overlay

Press **s** in the dashboard to view:
- User information
- API endpoint
- Usage statistics
- Rate limits
- Account details

### Status Info Navigation
- **j/k** - Navigate between fields
- **g/G** - Jump to first/last field
- **y** - Copy selected field value
- **s/q/ESC** - Close overlay
- **Q** - Force quit from overlay

## Bulk Operations View

Press **b** in the dashboard to access bulk operations for processing multiple configuration files.

### Bulk View Modes

The bulk view operates in several sequential modes:

#### 1. File Selection Mode (Default)
- Browse and select configuration files (JSON, YAML, JSONL, Markdown)
- Fuzzy search through available files in project directory
- Multi-file selection with checkboxes
- Preview file contents before selection

**Navigation:**
- **↑↓** or **j/k** - Navigate through files
- **Space** - Toggle file selection
- **Enter** - Confirm selection and proceed to run list
- **Ctrl+A** - Select all visible files
- **Ctrl+D** - Deselect all files
- **q** - Return to dashboard

#### 2. Run List Mode
- Review all runs loaded from selected configuration files
- Toggle individual runs for submission
- Edit run parameters before submission
- View repository and branch information

**Navigation:**
- **↑↓** or **j/k** - Navigate through runs
- **Space** - Toggle run selection
- **F** - Return to file selection
- **Ctrl+S** - Submit selected runs
- **q** - Return to dashboard

#### 3. Progress Mode
- Real-time progress tracking for submitted bulk runs
- View completion statistics and individual run status
- Option to cancel batch operation

#### 4. Results Mode
- Summary of completed bulk operation
- Success/failure status for each run
- Direct links to created runs
- Error details for failed runs

### Bulk View Architecture

The bulk view follows the navigation pattern with proper separation of concerns:

```go
// Clean navigation messages instead of direct view creation
func (v *BulkView) handleRunListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    case key.Matches(msg, v.keys.Quit):
        return v, func() tea.Msg {
            return messages.NavigateBackMsg{}  // Back to dashboard
        }
}

// Uses global layout system for consistent styling
func (v *BulkView) renderRunList() string {
    if v.layout == nil {
        v.layout = components.NewWindowLayout(v.width, v.height)
    }
    // ... consistent rendering with status line
}
```

### Bulk View Features

- **Split File Architecture**: Organized into focused modules (`bulk_commands.go`, `bulk_messages.go`, etc.)
- **Global Layout System**: Uses `WindowLayout` for consistent borders and sizing
- **Message-Based Navigation**: All navigation through type-safe messages
- **Status Line Integration**: Context-aware help text and loading indicators
- **Error Recovery**: Graceful error handling with navigation options

## Tips and Tricks

### Quick Repository Selection
1. Press **n** to create new run
2. Tab to repository field
3. Press **Ctrl+F** to activate FZF
4. Type partial repository name
5. Press Enter to select

### Efficient Navigation
- Use **f** for fuzzy search instead of scrolling
- Press **Enter** to drill down through columns
- Use **y** variants for quick copying

### Keyboard-Only Workflow
1. Launch TUI: `repobird tui`
2. Press **f** to find repository
3. Press **Enter** to see runs
4. Press **f** again to find specific run
5. Press **Enter** to view details
6. Press **y** to copy information

## Configuration

The TUI respects the following configuration:

```yaml
# ~/.repobird/config.yaml
tui:
  refresh_interval: 30s
  default_layout: triple_column
  show_help_on_start: false
  
colors:
  success: green
  running: yellow
  failed: red
  pending: blue
```

## Troubleshooting

### FZF Mode Not Working
- Ensure terminal supports Unicode characters
- Check terminal width (minimum 80 columns recommended)
- Try resizing terminal window

### Display Issues
- Set `TERM=xterm-256color` for better color support
- Ensure terminal font supports emoji/Unicode
- Try different terminal emulator if issues persist

### Performance
- Large repositories/runs lists may cause slowdown
- Use FZF search to filter large lists
- Adjust refresh interval if needed

## Layouts

The TUI supports multiple layout modes:

1. **Triple Column** (default) - Full dashboard view
2. **All Runs** - Timeline view of all runs
3. **Repositories Only** - Focus on repository overview

Switch layouts using:
- **1** - Triple column
- **2** - All runs
- **3** - Repositories only
- **l** - Cycle through layouts

## Shared Components

The TUI uses reusable components for consistency across views:

### WindowLayout Component
**Global layout system for consistent sizing and borders across all views.**

#### Purpose
Eliminates inconsistent view sizing by providing centralized layout calculations. All views except the Dashboard (which uses multi-column layout) should use WindowLayout for consistent borders and dimensions.

#### Key Features
- **Automatic Border Calculations**: Accounts for lipgloss border expansion (2px wider than set width)
- **Consistent Margins**: Proper top/bottom spacing to prevent border cutoffs
- **Viewport Sizing**: Calculates content area dimensions for bubble tea viewports
- **Standard Styling**: Provides consistent box, title, and content styles
- **Responsive Design**: Handles terminal resizing and small screen fallbacks

#### Usage Pattern
```go
// In view constructor
layout: components.NewWindowLayout(width, height)

// In View() method
boxStyle := v.layout.CreateStandardBox()
titleStyle := v.layout.CreateTitleStyle()
contentStyle := v.layout.CreateContentStyle()

// For viewports
viewportWidth, viewportHeight := v.layout.GetViewportDimensions()
```

#### Methods
- `GetBoxDimensions()` - Returns width/height for lipgloss containers
- `GetViewportDimensions()` - Returns content area for bubble tea viewports
- `CreateStandardBox()` - Standard box style with borders
- `CreateTitleStyle()` - Consistent title formatting
- `CreateContentStyle()` - Content area styling
- `IsValidDimensions()` - Checks if terminal is large enough
- `Update(width, height)` - Recalculates on terminal resize

#### When to Use
- ✅ **Details View**: Single-box layout with content
- ✅ **Status View**: Single-box info display
- ✅ **Create Run View**: Form-based views
- ✅ **Error View**: Error display views
- ✅ **List View**: Single-column list views
- ✅ **Bulk View**: All modes except complex multi-step flows
- ❌ **Dashboard**: Uses custom 3-column Miller Columns layout (renderTripleColumnLayout)

### ScrollableList Component
- Multi-column scrollable lists with keyboard navigation
- Supports both row and column navigation
- Consistent selection highlighting and keyboard shortcuts
- Used in Dashboard, List View, and other list-based views

### Form Component
- Input forms with validation and field management
- Support for text input, text area, and select fields
- Real-time validation with error messages
- Consistent styling and keyboard navigation
- Used in Create Run view and configuration forms

### ErrorView Component
- Consistent error display with recovery options
- Recoverable errors allow going back to previous view
- Non-recoverable errors clear navigation history
- User-friendly error messages with suggested actions

### Navigation Context
Views can share temporary data through navigation context:
- Form data preservation during navigation
- Selected repository/run information
- User preferences and temporary state
- Automatically cleared when returning to dashboard

## Advanced Features

### Run Following
When viewing run details:
- Auto-updates for running tasks
- Shows real-time output
- Progress indicators

### Smart Defaults
- Auto-detects current git repository
- Remembers last used values
- Suggests branch names based on task

### Context Preservation
- Form data persists between views
- Repository history maintained
- Session state preserved

## Best Practices

1. **Use FZF liberally** - It's the fastest way to navigate
2. **Learn the shortcuts** - Especially f, n, and y
3. **Customize your config** - Adjust refresh rates and colors
4. **Use normal mode** - For vim users, it's more efficient
5. **Master the columns** - Think of them as a drill-down interface

## See Also

- [CLI Commands](./cli-reference.md)
- [Configuration Guide](./configuration-guide.md)
- [API Reference](./api-reference.md)
- [Troubleshooting](./troubleshooting.md)