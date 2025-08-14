# Bubble Tea Forms Best Practices Guide

## Overview

This guide documents best practices for implementing forms in Bubble Tea terminal applications, based on research of the ecosystem, analysis of our current implementation, and common patterns from the community.

## Table of Contents
- [Current Architecture Analysis](#current-architecture-analysis)
- [Available Form Solutions](#available-form-solutions)
- [Best Practices & Patterns](#best-practices--patterns)
- [Our Implementation Assessment](#our-implementation-assessment)
- [Recommendations](#recommendations)

## Current Architecture Analysis

### What We Have: Custom Form Implementation

Our `CreateRunView` uses a custom form implementation (`CustomCreateForm`) with the following characteristics:

**Strengths:**
- ✅ **Vim-style modal editing**: Proper insert/normal mode separation
- ✅ **Field navigation**: Tab/Shift+Tab navigation between fields
- ✅ **Mixed field types**: Supports text, textarea, toggle, and button types
- ✅ **Visual feedback**: Clear focus indicators and emojis for field identification
- ✅ **Form persistence**: Auto-saves form state to cache
- ✅ **Validation**: Basic required field validation
- ✅ **Integration**: Works well with our navigation system and keymap architecture

**Areas for Improvement:**
- ⚠️ **Not using standard Bubbles components fully**: We have a `FormComponent` in `internal/tui/components/form.go` that's unused
- ⚠️ **Manual focus management**: Could be abstracted into reusable patterns
- ⚠️ **Limited scrolling**: No viewport for long forms
- ⚠️ **Validation feedback**: Could be more immediate/inline

### Existing Reusable Components

We have several modular components that could be leveraged:

1. **`components/FormComponent`** (`form.go`)
   - Generic form implementation with field abstraction
   - Supports TextInput, TextArea, and Select types
   - Has proper keymap configuration
   - **Currently unused** - could replace custom implementation

2. **`components/ScrollableList`** (`scrollable_list.go`)
   - Handles viewport and scrolling
   - Multi-column support
   - Could be adapted for form field lists

3. **`components/WindowLayout`** (`window_layout.go`)
   - Consistent sizing and borders
   - Already used by CreateRunView
   - Provides proper viewport dimensions

## Available Form Solutions

### 1. Huh Library (by Charm)
**Best for:** Quick, standard forms with minimal customization

```go
form := huh.NewForm(
    huh.NewInput().Title("Name").Value(&name),
    huh.NewSelect[string]().
        Title("Type").
        Options(
            huh.NewOption("Run", "run"),
            huh.NewOption("Plan", "plan"),
        ).Value(&runType),
)
```

**Pros:**
- Declarative, minimal boilerplate
- Built-in validation
- Handles all navigation/focus automatically

**Cons:**
- Less control over custom behaviors
- May not fit complex UI requirements
- Another dependency to manage

### 2. Bubbles Components (Low-level)
**Best for:** Custom UIs with specific requirements

```go
type model struct {
    inputs []textinput.Model
    focus  int
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Manual focus management
    if key == "tab" {
        m.focus = (m.focus + 1) % len(m.inputs)
    }
    // Update only focused input
    m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
}
```

**Pros:**
- Complete control
- Can build exactly what you need
- Integrates naturally with Bubble Tea

**Cons:**
- More boilerplate
- Must implement navigation, validation, etc.

### 3. Custom Implementation (Current Approach)
**Best for:** Domain-specific requirements with unique behaviors

Our current approach combines Bubbles components with custom logic for our specific needs.

## Best Practices & Patterns

### 1. Modal Editing (Vim-style)

```go
type Mode int
const (
    NormalMode Mode = iota
    InsertMode
)

type FormModel struct {
    mode   Mode
    fields []Field
    focus  int
}

// Key handling based on mode
func (m FormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if key, ok := msg.(tea.KeyMsg); ok {
        switch m.mode {
        case NormalMode:
            return m.handleNormalMode(key)
        case InsertMode:
            return m.handleInsertMode(key)
        }
    }
}
```

### 2. Focus Management Pattern

```go
// Centralized focus management
type FocusManager struct {
    items []Focusable
    index int
}

func (fm *FocusManager) Next() {
    fm.items[fm.index].Blur()
    fm.index = (fm.index + 1) % len(fm.items)
    fm.items[fm.index].Focus()
}

func (fm *FocusManager) Current() Focusable {
    return fm.items[fm.index]
}
```

### 3. Validation Pattern

```go
type Validator func(string) error

type Field struct {
    Name       string
    Value      string
    Validators []Validator
    Error      string
}

func (f *Field) Validate() bool {
    for _, v := range f.Validators {
        if err := v(f.Value); err != nil {
            f.Error = err.Error()
            return false
        }
    }
    f.Error = ""
    return true
}

// Usage
field := Field{
    Name: "email",
    Validators: []Validator{
        RequiredValidator,
        EmailValidator,
    },
}
```

### 4. Scrolling in Forms

```go
type ScrollableForm struct {
    viewport viewport.Model
    fields   []Field
    offset   int
    visible  int // Number of visible fields
}

func (f ScrollableForm) View() string {
    // Render only visible fields
    start := f.offset
    end := min(f.offset + f.visible, len(f.fields))
    
    content := ""
    for i := start; i < end; i++ {
        content += f.renderField(f.fields[i])
    }
    
    f.viewport.SetContent(content)
    return f.viewport.View()
}
```

### 5. Form State Persistence

```go
type FormState struct {
    Values map[string]string
    Focus  int
    Mode   Mode
}

func (f *Form) Save() FormState {
    state := FormState{
        Values: make(map[string]string),
        Focus:  f.focusIndex,
        Mode:   f.mode,
    }
    for _, field := range f.fields {
        state.Values[field.Name] = field.Value
    }
    return state
}

func (f *Form) Load(state FormState) {
    for name, value := range state.Values {
        f.SetFieldValue(name, value)
    }
    f.focusIndex = state.Focus
    f.mode = state.Mode
}
```

## Our Implementation Assessment

### What We're Doing Right

1. **Modal Editing**: Our implementation correctly separates insert/normal modes
2. **Navigation Context**: Smart use of cache for form persistence
3. **Field Types**: Good mix of input types (text, textarea, toggle)
4. **Visual Design**: Clear with emojis and styling
5. **Integration**: Works well with our keymap system

### What Could Be Improved

1. **Component Reuse**: We have `FormComponent` but aren't using it
2. **Scrolling**: No viewport for long forms
3. **Validation UX**: Could show inline errors as user types
4. **Field Dependencies**: No reactive field updates (e.g., auto-generate target branch)

## Recommendations

### Short Term (Keep Current Implementation)

Our current `CustomCreateForm` is actually quite good and follows many best practices. To improve it:

1. **Add Viewport for Scrolling**:
```go
// In CustomCreateForm
type CustomCreateForm struct {
    viewport viewport.Model
    // ... existing fields
}

func (f *CustomCreateForm) View() string {
    if f.height > 20 && len(f.fields) > 10 {
        // Use viewport for long forms
        f.viewport.SetContent(f.renderFields())
        return f.viewport.View()
    }
    return f.renderFields()
}
```

2. **Improve Validation Feedback**:
```go
// Add inline validation
func (f *CustomCreateForm) validateField(field *CustomFormField) {
    if field.Required && field.Value == "" {
        f.errors[field.Name] = "This field is required"
    } else {
        delete(f.errors, field.Name)
    }
}

// Call on each keystroke in insert mode
func (f *CustomCreateForm) handleInsertMode(msg tea.KeyMsg) {
    // ... update field
    f.validateField(currentField)
}
```

3. **Add Field Dependencies**:
```go
// Auto-generate target branch from title
func (f *CustomCreateForm) updateDependentFields(changedField string) {
    if changedField == "title" {
        title := f.GetValue("title")
        if f.GetValue("target") == "" {
            // Auto-generate branch name
            branch := "fix/" + strings.ToLower(strings.ReplaceAll(title, " ", "-"))
            f.SetValue("target", branch)
        }
    }
}
```

### Long Term (Consider Refactoring)

If we need more complex forms in the future:

1. **Extract Common Patterns**: Create a base form struct that handles:
   - Focus management
   - Mode switching
   - Navigation
   - Validation

2. **Use FormComponent**: Refactor to use our existing `FormComponent` as a base:
```go
// Extend FormComponent for specific needs
type CreateRunForm struct {
    *components.FormComponent
    // Add custom fields/methods
}
```

3. **Consider Huh**: For simple forms (config, settings), use Huh library to reduce maintenance

## Migration Path (If Desired)

If we want to move to a more standard approach:

### Option 1: Use Our FormComponent
```go
// Replace CustomCreateForm with configured FormComponent
form := components.NewForm(
    components.WithFields([]components.FormField{
        {Name: "title", Label: "Title", Type: components.TextInput, Required: true},
        {Name: "repository", Label: "Repository", Type: components.TextInput, Required: true},
        // ...
    }),
    components.WithFormKeymaps(customKeymaps),
)
```

### Option 2: Adopt Huh for Simple Forms
```go
// For configuration or simple input forms
form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().Title("Title").Value(&title),
        huh.NewInput().Title("Repository").Value(&repo),
    ),
)
```

### Option 3: Keep Current (Recommended)
Our current implementation is working well and is maintainable. The customizations we have (vim modes, field types, navigation integration) justify the custom approach.

## Conclusion

Our current form implementation in `CreateRunView` is actually following most Bubble Tea best practices. It's not "wonky" but rather a legitimate custom implementation that fits our specific needs. The main improvements would be:

1. Adding viewport support for long forms
2. Improving validation feedback
3. Potentially extracting common patterns for reuse

The Bubble Tea ecosystem doesn't have a single "correct" way to handle forms - it provides the primitives and patterns, and applications build what they need. Our approach is valid and working well for our use case.