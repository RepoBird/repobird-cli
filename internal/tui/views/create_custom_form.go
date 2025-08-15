package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// CustomFormField represents a field in the custom form
type CustomFormField struct {
	Name        string
	Label       string
	Type        string // "text", "textarea", "toggle", "button"
	Value       string
	Placeholder string
	Required    bool
	Options     []string // For toggle fields
	Icon        string   // Emoji icon for the field
	textInput   textinput.Model
	textArea    textarea.Model
}

// CustomCreateForm is a specialized form for the Create Run view
type CustomCreateForm struct {
	fields       []CustomFormField
	focusIndex   int
	insertMode   bool
	width        int
	height       int
	errors       map[string]string
	runTypeIndex int // 0 for "run", 1 for "plan"

	// Styling
	labelStyle    lipgloss.Style
	focusedStyle  lipgloss.Style
	errorStyle    lipgloss.Style
	requiredStyle lipgloss.Style
	buttonStyle   lipgloss.Style
	toggleStyle   lipgloss.Style
}

// NewCustomCreateForm creates a new custom form for Create Run view
func NewCustomCreateForm() *CustomCreateForm {
	f := &CustomCreateForm{
		errors:        make(map[string]string),
		runTypeIndex:  0, // Default to "run"
		labelStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		focusedStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		requiredStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		buttonStyle:   lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230")).Padding(0, 2),
		toggleStyle:   lipgloss.NewStyle().Background(lipgloss.Color("237")).Foreground(lipgloss.Color("252")).Padding(0, 1),
	}

	// Initialize fields with emojis
	f.fields = []CustomFormField{
		{
			Name:        "title",
			Label:       "Title",
			Type:        "text",
			Placeholder: "Brief description of the task",
			Required:    false,  // Not required per user request
			Icon:        "📝",
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        "text",
			Placeholder: "org/repo",
			Required:    true,
			Icon:        "📦",
		},
		{
			Name:        "source",
			Label:       "Source Branch",
			Type:        "text",
			Placeholder: "main",
			Value:       "",  // Keep empty by default - no git detection
			Required:    false,  // Not required per user request
			Icon:        "🌿",
		},
		{
			Name:        "target",
			Label:       "Target Branch",
			Type:        "text",
			Placeholder: "feature/new-feature",
			Required:    false,
			Icon:        "🎯",
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        "textarea",
			Placeholder: "Describe what you want the AI to do...",
			Required:    true,
			Icon:        "💭",
		},
		{
			Name:        "context",
			Label:       "Additional Context",
			Type:        "textarea",
			Placeholder: "Any additional context or requirements...",
			Required:    false,
			Icon:        "📋",
		},
		{
			Name:    "runtype",
			Label:   "Run Type",
			Type:    "toggle",
			Options: []string{"run", "plan"},
			Value:   "run",
			Icon:    "⚙️",
		},
		{
			Name:  "submit",
			Label: "Submit",
			Type:  "button",
			Value: "[CTRL+S] Submit Run",
			Icon:  "🚀",
		},
	}

	// Initialize text inputs and areas
	for i := range f.fields {
		switch f.fields[i].Type {
		case "text":
			ti := textinput.New()
			ti.Placeholder = f.fields[i].Placeholder
			ti.SetValue(f.fields[i].Value)
			f.fields[i].textInput = ti

		case "textarea":
			ta := textarea.New()
			ta.Placeholder = f.fields[i].Placeholder
			ta.SetValue(f.fields[i].Value)
			ta.SetHeight(3)
			f.fields[i].textArea = ta
		}
	}

	return f
}

// Init initializes the form
func (f *CustomCreateForm) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the custom form
func (f *CustomCreateForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height

	case tea.KeyMsg:
		return f.handleKeyMsg(msg)
	}

	return f, tea.Batch(cmds...)
}

// handleKeyMsg processes keyboard input for the form
func (f *CustomCreateForm) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	keyString := msg.String()
	debug.LogToFilef("🎨 CUSTOM FORM handleKeyMsg: key='%s', insertMode=%v, focusIndex=%d", keyString, f.insertMode, f.focusIndex)

	currentField := &f.fields[f.focusIndex]

	// Handle insert mode
	if f.insertMode {
		debug.LogToFilef("📝 CUSTOM FORM: Processing key '%s' in INSERT mode", keyString)
		switch keyString {
		case "esc":
			// ESC should not reach here if HandleKey is working correctly
			// But if it does, we should handle it to prevent issues
			debug.LogToFilef("⚠️ CUSTOM FORM: ESC in insert mode reached form (should have been handled by HandleKey)")
			// Exit insert mode as a fallback
			f.insertMode = false
			f.blurCurrentField()
			return f, nil

		case "tab":
			f.insertMode = false
			f.nextField()
			// Re-enter insert mode if on text field
			if f.fields[f.focusIndex].Type == "text" || f.fields[f.focusIndex].Type == "textarea" {
				f.insertMode = true
				f.focusCurrentField()
			}
			return f, nil

		case "shift+tab":
			f.insertMode = false
			f.prevField()
			// Re-enter insert mode if on text field
			if f.fields[f.focusIndex].Type == "text" || f.fields[f.focusIndex].Type == "textarea" {
				f.insertMode = true
				f.focusCurrentField()
			}
			return f, nil

		case "ctrl+s":
			// Submit form
			if f.validate() {
				return f, f.submitCmd()
			}
			return f, nil

		default:
			// Pass to the focused field if it's a text input
			if currentField.Type == "text" || currentField.Type == "textarea" {
				var cmd tea.Cmd
				switch currentField.Type {
				case "text":
					currentField.textInput, cmd = currentField.textInput.Update(msg)
					currentField.Value = currentField.textInput.Value()
				case "textarea":
					currentField.textArea, cmd = currentField.textArea.Update(msg)
					currentField.Value = currentField.textArea.Value()
				}
				return f, cmd
			}
		}
	} else {
		// Normal mode navigation
		debug.LogToFilef("🔑 CUSTOM FORM: Normal mode key: '%s'", msg.String())

		switch msg.String() {
		case "q", "b":
			// These should be handled by the keymap registry for navigation
			// If they reach here, just pass them through
			debug.LogToFilef("🔑 CUSTOM FORM: Navigation key '%s' in normal mode - passing through", msg.String())
			// Don't process - let keymap registry handle navigation
			return f, nil

		case "i":
			// Enter insert mode for text fields
			if currentField.Type == "text" || currentField.Type == "textarea" {
				f.insertMode = true
				f.focusCurrentField()
				debug.LogToFilef("✏️ CUSTOM FORM: Entering insert mode for field '%s'", currentField.Name)
				return f, textinput.Blink
			}
			return f, nil

		case "d":
			// Delete current field's text (vim-like)
			if currentField.Type == "text" || currentField.Type == "textarea" {
				f.ClearCurrentField()
				debug.LogToFilef("✂️ CUSTOM FORM: Deleted field '%s' content", currentField.Name)
			}
			return f, nil

		case "c":
			// Change - delete current field and enter insert mode (vim-like)
			if currentField.Type == "text" || currentField.Type == "textarea" {
				f.ClearCurrentField()
				f.insertMode = true
				f.focusCurrentField()
				debug.LogToFilef("✏️ CUSTOM FORM: Change mode - cleared field '%s' and entering insert", currentField.Name)
				return f, textinput.Blink
			}
			return f, nil

		case "j", "down":
			f.nextField()
			debug.LogToFilef("⬇️ CUSTOM FORM: Moving to next field")
			return f, nil

		case "k", "up":
			f.prevField()
			debug.LogToFilef("⬆️ CUSTOM FORM: Moving to previous field")
			return f, nil

		case "enter", " ":
			// Handle special field types
			switch currentField.Type {
			case "toggle":
				// Toggle the runtype
				if currentField.Name == "runtype" {
					f.runTypeIndex = (f.runTypeIndex + 1) % len(currentField.Options)
					currentField.Value = currentField.Options[f.runTypeIndex]
					debug.LogToFilef("🔄 CUSTOM FORM: Toggled runtype to %s", currentField.Value)
				}
			case "button":
				// Submit the form
				if currentField.Name == "submit" && f.validate() {
					debug.LogToFilef("🚀 CUSTOM FORM: Submit button pressed")
					return f, f.submitCmd()
				}
			default:
				// Enter insert mode for text fields
				f.insertMode = true
				f.focusCurrentField()
				debug.LogToFilef("✏️ CUSTOM FORM: ENTER pressed - entering insert mode for field '%s'", currentField.Name)
				return f, textinput.Blink
			}
			return f, nil

		case "ctrl+s":
			// Submit from any field
			if f.validate() {
				debug.LogToFilef("💾 CUSTOM FORM: CTRL+S pressed - submitting form")
				return f, f.submitCmd()
			}
			debug.LogToFilef("⚠️ CUSTOM FORM: CTRL+S pressed but validation failed")
			return f, nil

		default:
			debug.LogToFilef("❓ CUSTOM FORM: Unhandled key in normal mode: '%s'", msg.String())
		}
	}

	return f, nil
}

// View renders the custom form
func (f *CustomCreateForm) View() string {
	if f.width == 0 || f.height == 0 {
		return ""
	}

	var sections []string

	for i, field := range f.fields {
		sections = append(sections, f.renderField(field, i == f.focusIndex))
	}

	// Add mode indicator at the bottom
	modeStr := "Normal Mode (i to edit, j/k to navigate)"
	if f.insertMode {
		modeStr = "Insert Mode (ESC to exit, TAB to next field)"
	}
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242")).
		Italic(true)
	sections = append(sections, "", modeStyle.Render(modeStr))

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderField renders a single form field with custom styling
func (f *CustomCreateForm) renderField(field CustomFormField, focused bool) string {
	// Build label with emoji
	var label string
	if field.Icon != "" {
		label = field.Icon + " "
	}
	label += field.Label

	if field.Required {
		label = f.labelStyle.Render(label) + f.requiredStyle.Render(" *")
	} else {
		label = f.labelStyle.Render(label)
	}

	var input string

	switch field.Type {
	case "text":
		if focused && f.insertMode {
			input = field.textInput.View()
		} else {
			value := field.Value
			if value == "" {
				value = lipgloss.NewStyle().
					Foreground(lipgloss.Color("239")).
					Render(field.Placeholder)
			}
			if focused {
				input = f.focusedStyle.Render("▶ " + value)
			} else {
				input = "  " + value
			}
		}

	case "textarea":
		if focused && f.insertMode {
			input = field.textArea.View()
		} else {
			value := field.Value
			if value == "" {
				value = lipgloss.NewStyle().
					Foreground(lipgloss.Color("239")).
					Render(field.Placeholder)
			} else {
				// Compact multi-line text to single line when not in insert mode
				// This prevents overflow issues
				lines := strings.Split(value, "\n")
				if len(lines) > 1 {
					// Show first line with ellipsis to indicate more content
					firstLine := strings.TrimSpace(lines[0])
					if firstLine == "" && len(lines) > 1 {
						// If first line is empty, try to find first non-empty line
						for _, line := range lines {
							if trimmed := strings.TrimSpace(line); trimmed != "" {
								firstLine = trimmed
								break
							}
						}
					}
					if firstLine == "" {
						firstLine = "[empty]"
					}
					value = firstLine + " ..."
				} else {
					value = strings.TrimSpace(value)
				}
			}
			if focused {
				input = f.focusedStyle.Render("▶ " + value)
			} else {
				input = "  " + value
			}
		}

	case "toggle":
		// Render toggle button
		options := []string{}
		for i, opt := range field.Options {
			style := f.toggleStyle
			if i == f.runTypeIndex {
				style = style.Background(lipgloss.Color("62"))
			}
			options = append(options, style.Render(opt))
		}
		toggleStr := strings.Join(options, " ")

		if focused {
			input = f.focusedStyle.Render("▶ ") + toggleStr + lipgloss.NewStyle().Foreground(lipgloss.Color("239")).Render(" (ENTER to toggle)")
		} else {
			input = "  " + toggleStr
		}

	case "button":
		// Render submit button
		buttonStyle := f.buttonStyle
		if focused {
			buttonStyle = buttonStyle.Background(lipgloss.Color("205"))
		}
		button := buttonStyle.Render(field.Value)

		if focused {
			input = f.focusedStyle.Render("▶ ") + button
		} else {
			input = "  " + button
		}

		// Don't show label for button
		return input
	}

	// Add error message if present
	if errMsg, ok := f.errors[field.Name]; ok {
		input += "\n" + f.errorStyle.Render("  ↳ "+errMsg)
	}

	return label + "\n" + input
}

// Helper methods

func (f *CustomCreateForm) nextField() {
	f.blurCurrentField()
	f.focusIndex = (f.focusIndex + 1) % len(f.fields)
	if f.insertMode {
		f.focusCurrentField()
	}
}

func (f *CustomCreateForm) prevField() {
	f.blurCurrentField()
	f.focusIndex--
	if f.focusIndex < 0 {
		f.focusIndex = len(f.fields) - 1
	}
	if f.insertMode {
		f.focusCurrentField()
	}
}

func (f *CustomCreateForm) focusCurrentField() {
	if f.focusIndex < len(f.fields) {
		field := &f.fields[f.focusIndex]
		switch field.Type {
		case "text":
			field.textInput.Focus()
		case "textarea":
			field.textArea.Focus()
		}
	}
}

func (f *CustomCreateForm) blurCurrentField() {
	if f.focusIndex < len(f.fields) {
		field := &f.fields[f.focusIndex]
		switch field.Type {
		case "text":
			field.textInput.Blur()
		case "textarea":
			field.textArea.Blur()
		}
	}
}

func (f *CustomCreateForm) validate() bool {
	valid := true
	f.errors = make(map[string]string)

	for _, field := range f.fields {
		if field.Required && field.Type != "button" && field.Type != "toggle" {
			if strings.TrimSpace(field.Value) == "" {
				f.errors[field.Name] = "This field is required"
				valid = false
			}
		}
	}

	return valid
}

func (f *CustomCreateForm) submitCmd() tea.Cmd {
	values := f.GetValues()
	return func() tea.Msg {
		return CustomFormSubmitMsg{Values: values}
	}
}

// CustomFormSubmitMsg is sent when the custom form is submitted
type CustomFormSubmitMsg struct {
	Values map[string]string
}

// CustomFormNavigateBackMsg signals that the user wants to navigate back
type CustomFormNavigateBackMsg struct{}

// Public methods for external access

func (f *CustomCreateForm) GetValues() map[string]string {
	values := make(map[string]string)
	for _, field := range f.fields {
		if field.Type != "button" {
			values[field.Name] = field.Value
		}
	}
	return values
}

func (f *CustomCreateForm) SetValue(fieldName, value string) {
	for i := range f.fields {
		if f.fields[i].Name == fieldName {
			f.fields[i].Value = value
			switch f.fields[i].Type {
			case "text":
				f.fields[i].textInput.SetValue(value)
			case "textarea":
				f.fields[i].textArea.SetValue(value)
			case "toggle":
				// Update runTypeIndex based on value
				for j, opt := range f.fields[i].Options {
					if opt == value {
						f.runTypeIndex = j
						break
					}
				}
			}
			break
		}
	}
}

func (f *CustomCreateForm) IsInsertMode() bool {
	return f.insertMode
}

func (f *CustomCreateForm) SetInsertMode(mode bool) {
	f.insertMode = mode
	if mode {
		f.focusCurrentField()
	} else {
		f.blurCurrentField()
	}
}

func (f *CustomCreateForm) GetFocusIndex() int {
	return f.focusIndex
}

func (f *CustomCreateForm) SetFocusIndex(index int) {
	if index >= 0 && index < len(f.fields) {
		f.blurCurrentField()
		f.focusIndex = index
		if f.insertMode {
			f.focusCurrentField()
		}
	}
}

func (f *CustomCreateForm) ClearCurrentField() {
	if f.focusIndex < len(f.fields) {
		field := &f.fields[f.focusIndex]
		// Only clear text-based fields
		switch field.Type {
		case "text":
			field.Value = ""
			field.textInput.SetValue("")
			field.textInput.Reset() // Ensure cursor is reset
		case "textarea":
			field.Value = ""
			field.textArea.SetValue("")
			field.textArea.Reset() // Ensure cursor is reset
		}
	}
}

func (f *CustomCreateForm) GetCurrentFieldName() string {
	if f.focusIndex < len(f.fields) {
		return f.fields[f.focusIndex].Name
	}
	return ""
}

// GetRunType returns the current runtype value
func (f *CustomCreateForm) GetRunType() string {
	for _, field := range f.fields {
		if field.Name == "runtype" {
			return field.Value
		}
	}
	return "run" // Default
}
