// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FieldType represents the type of form field
type FieldType int

const (
	// TextInput is a single-line text input field
	TextInput FieldType = iota
	// TextArea is a multi-line text input field
	TextArea
	// Select is a dropdown selection field
	Select
)

// FormField represents a single field in the form
type FormField struct {
	Name        string
	Label       string
	Type        FieldType
	Value       string
	Placeholder string
	Required    bool
	Options     []string // For Select type
	textInput   textinput.Model
	textArea    textarea.Model
}

// FormComponent is a reusable form component
type FormComponent struct {
	fields     []FormField
	focusIndex int
	insertMode bool
	keymaps    FormKeyMap
	width      int
	height     int
	errors     map[string]string

	// Styling
	labelStyle    lipgloss.Style
	focusedStyle  lipgloss.Style
	errorStyle    lipgloss.Style
	requiredStyle lipgloss.Style
}

// FormKeyMap contains key bindings specific to forms
type FormKeyMap struct {
	KeyMap     // Embed standard keymaps
	NextField  string
	PrevField  string
	InsertMode string
	NormalMode string
	Submit     string
}

// DefaultFormKeyMap returns the default form key bindings
func DefaultFormKeyMap() FormKeyMap {
	return FormKeyMap{
		KeyMap:     DefaultKeyMap,
		NextField:  "tab",
		PrevField:  "shift+tab",
		InsertMode: "i",
		NormalMode: "esc",
		Submit:     "ctrl+s",
	}
}

// FormOption is a functional option for configuring FormComponent
type FormOption func(*FormComponent)

// NewForm creates a new form with the given options
func NewForm(opts ...FormOption) *FormComponent {
	f := &FormComponent{
		fields:        []FormField{},
		focusIndex:    0,
		insertMode:    false,
		keymaps:       DefaultFormKeyMap(),
		errors:        make(map[string]string),
		labelStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		focusedStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("205")),
		errorStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		requiredStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
	}

	// Apply options
	for _, opt := range opts {
		opt(f)
	}

	// Initialize text inputs and areas
	for i := range f.fields {
		switch f.fields[i].Type {
		case TextInput:
			ti := textinput.New()
			ti.Placeholder = f.fields[i].Placeholder
			ti.SetValue(f.fields[i].Value)
			if i == 0 {
				ti.Focus()
			}
			f.fields[i].textInput = ti

		case TextArea:
			ta := textarea.New()
			ta.Placeholder = f.fields[i].Placeholder
			ta.SetValue(f.fields[i].Value)
			if i == 0 {
				ta.Focus()
			}
			f.fields[i].textArea = ta
		case Select:
			// Select fields don't need initialization
		}
	}

	return f
}

// WithFields sets the form fields
func WithFields(fields []FormField) FormOption {
	return func(f *FormComponent) {
		f.fields = fields
	}
}

// WithFormKeymaps sets custom keymaps for the form
func WithFormKeymaps(km FormKeyMap) FormOption {
	return func(f *FormComponent) {
		f.keymaps = km
	}
}

// WithFormDimensions sets the form dimensions
func WithFormDimensions(width, height int) FormOption {
	return func(f *FormComponent) {
		f.width = width
		f.height = height
	}
}

// Init initializes the form
func (f *FormComponent) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the form state
func (f *FormComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height

	case tea.KeyMsg:
		// Handle form navigation in normal mode
		if !f.insertMode {
			switch msg.String() {
			case f.keymaps.InsertMode, "i":
				f.insertMode = true
				f.focusCurrentField()
				return f, textinput.Blink

			case f.keymaps.NextField, "tab", "j":
				f.nextField()
				return f, nil

			case f.keymaps.PrevField, "shift+tab", "k":
				f.prevField()
				return f, nil

			case f.keymaps.Submit, "ctrl+s":
				// Validate and submit
				if f.validate() {
					// Return submit command
					return f, f.submitCmd()
				}
				return f, nil
			}
		} else {
			// In insert mode, handle text input
			switch msg.String() {
			case f.keymaps.NormalMode, "esc":
				f.insertMode = false
				f.blurCurrentField()
				return f, nil

			case f.keymaps.NextField, "tab":
				f.insertMode = false
				f.nextField()
				f.insertMode = true
				f.focusCurrentField()
				return f, nil

			case f.keymaps.PrevField, "shift+tab":
				f.insertMode = false
				f.prevField()
				f.insertMode = true
				f.focusCurrentField()
				return f, nil

			default:
				// Pass to the focused field
				if f.focusIndex < len(f.fields) {
					field := &f.fields[f.focusIndex]
					var cmd tea.Cmd

					switch field.Type {
					case TextInput:
						field.textInput, cmd = field.textInput.Update(msg)
						field.Value = field.textInput.Value()
						cmds = append(cmds, cmd)

					case TextArea:
						field.textArea, cmd = field.textArea.Update(msg)
						field.Value = field.textArea.Value()
						cmds = append(cmds, cmd)
					
					case Select:
						// Select fields handle updates differently
						// No-op for now
					}
				}
			}
		}
	}

	return f, tea.Batch(cmds...)
}

// View renders the form
func (f *FormComponent) View() string {
	if f.width == 0 || f.height == 0 {
		return ""
	}

	var sections []string

	for i, field := range f.fields {
		sections = append(sections, f.renderField(field, i == f.focusIndex))
	}

	// Add mode indicator
	modeStr := "Normal Mode (i to edit)"
	if f.insertMode {
		modeStr = "Insert Mode (ESC to exit)"
	}
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242")).
		Italic(true)
	sections = append(sections, "", modeStyle.Render(modeStr))

	// Add submit hint
	if !f.insertMode {
		submitHint := "Ctrl+S to submit"
		sections = append(sections, modeStyle.Render(submitHint))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderField renders a single form field
func (f *FormComponent) renderField(field FormField, focused bool) string {
	var label string
	if field.Required {
		label = f.labelStyle.Render(field.Label) + f.requiredStyle.Render(" *")
	} else {
		label = f.labelStyle.Render(field.Label)
	}

	var input string
	switch field.Type {
	case TextInput:
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

	case TextArea:
		if focused && f.insertMode {
			input = field.textArea.View()
		} else {
			value := field.Value
			if value == "" {
				value = lipgloss.NewStyle().
					Foreground(lipgloss.Color("239")).
					Render(field.Placeholder)
			}
			if focused {
				lines := strings.Split(value, "\n")
				if len(lines) > 0 {
					lines[0] = f.focusedStyle.Render("▶ " + lines[0])
					for i := 1; i < len(lines); i++ {
						lines[i] = "  " + lines[i]
					}
				}
				input = strings.Join(lines, "\n")
			} else {
				input = "  " + value
			}
		}

	case Select:
		// TODO: Implement select field rendering
		input = "  " + field.Value
	}

	// Add error message if present
	if errMsg, ok := f.errors[field.Name]; ok {
		input += "\n" + f.errorStyle.Render("  ↳ "+errMsg)
	}

	return label + "\n" + input
}

// GetValues returns the current form values
func (f *FormComponent) GetValues() map[string]string {
	values := make(map[string]string)
	for _, field := range f.fields {
		values[field.Name] = field.Value
	}
	return values
}

// SetValue sets the value of a specific field
func (f *FormComponent) SetValue(fieldName, value string) {
	for i := range f.fields {
		if f.fields[i].Name == fieldName {
			f.fields[i].Value = value
			switch f.fields[i].Type {
			case TextInput:
				f.fields[i].textInput.SetValue(value)
			case TextArea:
				f.fields[i].textArea.SetValue(value)
			case Select:
				// Select value is already set above
			}
			break
		}
	}
}

// SetError sets an error message for a specific field
func (f *FormComponent) SetError(fieldName, errorMsg string) {
	f.errors[fieldName] = errorMsg
}

// ClearErrors clears all error messages
func (f *FormComponent) ClearErrors() {
	f.errors = make(map[string]string)
}

// validate checks if all required fields are filled
func (f *FormComponent) validate() bool {
	valid := true
	f.ClearErrors()

	for _, field := range f.fields {
		if field.Required && strings.TrimSpace(field.Value) == "" {
			f.SetError(field.Name, "This field is required")
			valid = false
		}
	}

	return valid
}

// nextField moves focus to the next field
func (f *FormComponent) nextField() {
	f.blurCurrentField()
	f.focusIndex = (f.focusIndex + 1) % len(f.fields)
	if f.insertMode {
		f.focusCurrentField()
	}
}

// prevField moves focus to the previous field
func (f *FormComponent) prevField() {
	f.blurCurrentField()
	f.focusIndex--
	if f.focusIndex < 0 {
		f.focusIndex = len(f.fields) - 1
	}
	if f.insertMode {
		f.focusCurrentField()
	}
}

// focusCurrentField focuses the current field
func (f *FormComponent) focusCurrentField() {
	if f.focusIndex < len(f.fields) {
		field := &f.fields[f.focusIndex]
		switch field.Type {
		case TextInput:
			field.textInput.Focus()
		case TextArea:
			field.textArea.Focus()
		case Select:
			// Select fields don't have focus in the same way
		}
	}
}

// blurCurrentField blurs the current field
func (f *FormComponent) blurCurrentField() {
	if f.focusIndex < len(f.fields) {
		field := &f.fields[f.focusIndex]
		switch field.Type {
		case TextInput:
			field.textInput.Blur()
		case TextArea:
			field.textArea.Blur()
		case Select:
			// Select fields don't have blur in the same way
		}
	}
}

// submitCmd returns a command that can be used to handle form submission
func (f *FormComponent) submitCmd() tea.Cmd {
	return func() tea.Msg {
		return FormSubmitMsg{Values: f.GetValues()}
	}
}

// FormSubmitMsg is sent when the form is submitted
type FormSubmitMsg struct {
	Values map[string]string
}

// IsInsertMode returns whether the form is in insert mode
func (f *FormComponent) IsInsertMode() bool {
	return f.insertMode
}

// SetInsertMode sets the insert mode state
func (f *FormComponent) SetInsertMode(mode bool) {
	f.insertMode = mode
	if mode {
		f.focusCurrentField()
	} else {
		f.blurCurrentField()
	}
}

// GetFocusIndex returns the currently focused field index
func (f *FormComponent) GetFocusIndex() int {
	return f.focusIndex
}

// SetFocusIndex sets the focused field index
func (f *FormComponent) SetFocusIndex(index int) {
	if index >= 0 && index < len(f.fields) {
		f.blurCurrentField()
		f.focusIndex = index
		if f.insertMode {
			f.focusCurrentField()
		}
	}
}

// ClearCurrentField clears the value of the currently focused field
func (f *FormComponent) ClearCurrentField() {
	if f.focusIndex < len(f.fields) {
		field := &f.fields[f.focusIndex]
		field.Value = ""
		switch field.Type {
		case TextInput:
			field.textInput.SetValue("")
		case TextArea:
			field.textArea.SetValue("")
		case Select:
			// Select field value is already cleared above
		}
	}
}

// GetCurrentFieldName returns the name of the currently focused field
func (f *FormComponent) GetCurrentFieldName() string {
	if f.focusIndex < len(f.fields) {
		return f.fields[f.focusIndex].Name
	}
	return ""
}
