# TUI Key Management Architecture

## Overview

RepoBird CLI uses a centralized key processing system that provides consistent, extensible key handling across all views. This system was implemented to solve the problem of scattered key handling and provide a clean way for any view to disable or customize specific keys.

## Architecture

### Core Components

1. **CoreKeyRegistry** (`internal/tui/keymap/core.go`)
   - Central registry of all keys and their default actions
   - Maps key strings to actions (navigation, global, view-specific)
   - Extensible system for registering new keys

2. **CoreViewKeymap Interface**
   - Optional interface that views can implement
   - Provides `IsKeyDisabled()` for disabling keys
   - Provides `HandleKey()` for custom key handling

3. **Centralized Processor** (`App.processKeyWithFiltering()`)
   - Single entry point for all key processing
   - Checks view keymaps before executing actions
   - Routes actions to appropriate handlers

### Key Processing Flow

```
┌─────────────┐    ┌──────────────────────┐    ┌─────────────────┐
│ Key Press   │ -> │ App.processKey...()  │ -> │ Action Handler  │
└─────────────┘    └──────────────────────┘    └─────────────────┘
                            │
                            ▼
                   ┌─────────────────────┐
                   │ View Keymap Check   │
                   │ IsKeyDisabled()?    │
                   │ HandleKey()?        │
                   └─────────────────────┘
```

### Action Types

**Navigation Actions**
- `b` - Back navigation
- `B` - Bulk operations  
- `n` - New item
- `r` - Refresh
- `q` - Quit
- `?` - Help

**Global Actions**
- `Q` - Force quit (always works)
- `ctrl+c` - Force quit (always works)

**View-Specific Actions**
- `s` - Status/info
- `f` - Filter/search
- `enter` - Select
- `tab` - Next field
- Arrow keys, etc.

## Implementation Guide

### For Views That Need Key Customization

1. **Implement the Interface**
```go
type MyView struct {
    disabledKeys map[string]bool
}

func (v *MyView) IsKeyDisabled(keyString string) bool {
    return v.disabledKeys[keyString]
}

func (v *MyView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
    // Custom key handling logic
    return false, v, nil // Let system handle if no custom logic
}
```

2. **Disable Unwanted Keys**
```go
func NewMyView() *MyView {
    return &MyView{
        disabledKeys: map[string]bool{
            "b":   true, // Disable back navigation
            "esc": true, // Disable escape
        },
    }
}
```

3. **Custom Key Behaviors**
```go
func (v *MyView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
    switch keyMsg.String() {
    case "ctrl+s":
        if v.canSave() {
            return true, v, v.saveCommand()
        }
    case "ctrl+r":
        return true, v, v.customRefreshCommand()
    }
    return false, v, nil // Let system handle other keys
}
```

### Key Processing Priority

1. **Disabled Check**: If `IsKeyDisabled(key)` returns `true` → ignore completely
2. **Custom Handler**: If `HandleKey()` returns `handled=true` → use custom result
3. **Global Actions**: Force quit, etc. → handled regardless of view state  
4. **Navigation Actions**: Back, bulk, etc. → converted to navigation messages
5. **View-Specific**: Other keys → delegated to view's `Update()` method

### Examples

#### Dashboard (Disables Back Navigation)
```go
type DashboardView struct {
    disabledKeys map[string]bool
}

func NewDashboardView(client APIClient) *DashboardView {
    return &DashboardView{
        disabledKeys: map[string]bool{
            "b":   true, // No back navigation from dashboard
            "esc": true, // No escape from dashboard
        },
    }
}

func (d *DashboardView) IsKeyDisabled(keyString string) bool {
    return d.disabledKeys[keyString]
}

func (d *DashboardView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
    return false, d, nil // No custom handling needed
}
```

#### Create View (Custom Save Shortcut)
```go
func (c *CreateView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
    if keyMsg.String() == "ctrl+s" && c.isFormValid() {
        // Custom save with validation
        return true, c, c.submitFormCommand()
    }
    return false, c, nil
}
```

## Benefits

- **Consistent Behavior**: Same key handling logic across all views
- **Extensible**: Easy to add new keys or customize existing ones
- **Maintainable**: Single place to understand key processing
- **Debuggable**: Clear flow for tracing key handling
- **Backward Compatible**: Views without keymap interface work unchanged

## Debugging Key Issues

1. **Add Debug Logging**
```go
func (a *App) processKeyWithFiltering(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
    keyString := keyMsg.String()
    debug.LogToFilef("Key pressed: %s", keyString)
    
    if viewKeymap, hasKeymap := a.current.(keymap.CoreViewKeymap); hasKeymap {
        if viewKeymap.IsKeyDisabled(keyString) {
            debug.LogToFilef("Key %s disabled by view", keyString)
            return true, a, nil
        }
    }
    // ...
}
```

2. **Check Key Registry**
```go
action := a.keyRegistry.GetAction(keyString)
debug.LogToFilef("Key %s maps to action %v", keyString, action)
```

3. **Verify Interface Implementation**
```go
if viewKeymap, hasKeymap := a.current.(keymap.CoreViewKeymap); hasKeymap {
    debug.LogToFilef("View implements CoreViewKeymap: %T", a.current)
} else {
    debug.LogToFilef("View does not implement CoreViewKeymap: %T", a.current)
}
```

## Migration Guide

### Existing Views
1. Views without `CoreViewKeymap` continue to work unchanged
2. All keys are enabled by default
3. No migration required unless custom behavior is needed

### Adding Key Customization
1. Add `CoreViewKeymap` interface to view struct
2. Implement `IsKeyDisabled()` method
3. Implement `HandleKey()` method (can return `false, view, nil` if no custom handling)
4. Remove any existing scattered key handling from `Update()` method

This architecture provides a clean, extensible foundation for key handling that scales with the application's complexity while maintaining simplicity for basic use cases.