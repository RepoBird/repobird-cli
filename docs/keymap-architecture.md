# Keymap Architecture

## Overview

Centralized key processing system providing consistent, extensible key handling across all TUI views.

## Related Documentation
- **[TUI Guide](tui-guide.md)** - Complete TUI navigation and usage
- **[Architecture Overview](architecture.md)** - System design patterns
- **[Dashboard Layouts](dashboard-layouts.md)** - View-specific key handling

## Architecture

### Core Components

**1. CoreKeyRegistry** (`internal/tui/keymap/core.go`)
- Central registry mapping keys to actions
- Defines navigation, global, and view-specific actions

**2. CoreViewKeymap Interface**
```go
type CoreViewKeymap interface {
    IsKeyDisabled(keyString string) bool
    HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd)
}
```

**3. Processing Flow**
```
Key Press → App.processKeyWithFiltering() → View Check → Action
                    ↓
            1. Check IsKeyDisabled()
            2. Try HandleKey()
            3. Process global actions
            4. Convert to navigation
            5. Delegate to view
```

## Key Categories

### Navigation Keys
- `b` - Back navigation
- `B` - Bulk operations
- `n` - New item
- `r` - Refresh
- `q` - Quit/back
- `?` - Help

### Global Keys (Always Active)
- `Q` / `ctrl+c` - Force quit from anywhere

### View-Specific Keys
- `s` - Status overlay
- `f` - Filter/search
- `enter` - Select
- `tab` - Next field
- Arrow keys - Navigation

## Implementation

### Disable Keys in a View
```go
type MyView struct {
    disabledKeys map[string]bool
}

func NewMyView() *MyView {
    return &MyView{
        disabledKeys: map[string]bool{
            "b": true,   // Disable back
            "esc": true, // Disable escape
        },
    }
}

func (v *MyView) IsKeyDisabled(key string) bool {
    return v.disabledKeys[key]
}
```

### Custom Key Handling
```go
func (v *MyView) HandleKey(msg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
    switch msg.String() {
    case "ctrl+s":
        if v.canSave() {
            return true, v, v.saveCommand()
        }
    case "ctrl+r":
        return true, v, v.customRefresh()
    }
    return false, v, nil // System handles other keys
}
```

## Processing Priority

1. **Disabled Check** - Ignored if disabled
2. **Custom Handler** - View's custom logic
3. **Global Actions** - Force quit always works
4. **Navigation** - Converted to messages
5. **View Default** - Delegated to Update()

## Real Examples

### Dashboard (No Back Navigation)
```go
func NewDashboardView() *DashboardView {
    return &DashboardView{
        disabledKeys: map[string]bool{
            "b": true,   // Top level, no back
            "esc": true,
        },
    }
}
```

### Details View (Custom Copy)
```go
func (d *DetailsView) HandleKey(msg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
    if msg.String() == "y" {
        // Custom yank behavior
        return true, d, d.copyToClipboard()
    }
    return false, d, nil
}
```

### Bulk View (Mode-Specific Keys)
```go
func (b *BulkView) IsKeyDisabled(key string) bool {
    switch b.mode {
    case FileSelectMode:
        return key == "enter" && b.selectedFile == ""
    case ProgressMode:
        return key == "q" // Can't quit during progress
    }
    return false
}
```

## Benefits

✅ **Consistent** - Same key behavior across views  
✅ **Extensible** - Easy to add new keys/actions  
✅ **Maintainable** - Single processing location  
✅ **Debuggable** - Clear processing flow  
✅ **Flexible** - Per-view customization

## Best Practices

1. **Minimal Disabling** - Only disable when necessary
2. **Clear Feedback** - Show why keys are disabled
3. **Document Changes** - Note non-standard behavior
4. **Test Thoroughly** - Verify key interactions
5. **Consistent Patterns** - Follow established conventions

## Adding New Keys

1. Register in `CoreKeyRegistry`
2. Define action type
3. Add to processing logic
4. Update help text
5. Test across views

## Debugging

```go
// In processKeyWithFiltering()
debug.LogToFilef("Key pressed: %s, Disabled: %v", 
    keyString, view.IsKeyDisabled(keyString))
```

Enable debug logging:
```bash
REPOBIRD_DEBUG_LOG=1 repobird tui
tail -f /tmp/repobird_debug.log
```