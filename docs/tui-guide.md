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

### Basic Navigation
- **Tab** - Cycle through columns
- **↑↓** or **j/k** - Move up/down in current column
- **←→** or **h/l** - Move between columns
- **Enter** - Select item and move to next column
- **Backspace** - Move to previous column

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
- **q** - Quit

### Clipboard Operations
- **y** - Copy current selection to clipboard
- **yy** - Copy entire run details
- **yp** - Copy prompt
- **yc** - Copy context

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

## Status Info Overlay

Press **s** in the dashboard to view:
- User information
- API endpoint
- Usage statistics
- Rate limits
- Account details

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