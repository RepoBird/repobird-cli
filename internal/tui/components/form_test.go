package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewForm(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		form := NewForm()
		
		assert.NotNil(t, form)
		assert.Empty(t, form.fields)
		assert.Equal(t, 0, form.focusIndex)
		assert.False(t, form.insertMode)
		assert.NotNil(t, form.errors)
	})
	
	t.Run("With fields", func(t *testing.T) {
		fields := []FormField{
			{
				Name:        "username",
				Label:       "Username",
				Type:        TextInput,
				Required:    true,
				Placeholder: "Enter username",
			},
			{
				Name:        "description",
				Label:       "Description",
				Type:        TextArea,
				Required:    false,
				Placeholder: "Enter description",
			},
		}
		
		form := NewForm(WithFields(fields))
		
		assert.Len(t, form.fields, 2)
		assert.Equal(t, "username", form.fields[0].Name)
		assert.Equal(t, "description", form.fields[1].Name)
		assert.NotNil(t, form.fields[0].textInput)
		assert.NotNil(t, form.fields[1].textArea)
	})
	
	t.Run("With custom keymaps", func(t *testing.T) {
		customKeys := FormKeyMap{
			NextField:  "ctrl+n",
			PrevField:  "ctrl+p",
			InsertMode: "a",
			NormalMode: "ctrl+[",
			Submit:     "ctrl+enter",
		}
		
		form := NewForm(WithFormKeymaps(customKeys))
		
		assert.Equal(t, "ctrl+n", form.keymaps.NextField)
		assert.Equal(t, "ctrl+p", form.keymaps.PrevField)
		assert.Equal(t, "a", form.keymaps.InsertMode)
	})
}

func TestFormNavigation(t *testing.T) {
	fields := []FormField{
		{Name: "field1", Label: "Field 1", Type: TextInput, Required: true},
		{Name: "field2", Label: "Field 2", Type: TextInput, Required: false},
		{Name: "field3", Label: "Field 3", Type: TextArea, Required: false},
	}
	
	form := NewForm(WithFields(fields))
	
	t.Run("Next field navigation", func(t *testing.T) {
		assert.Equal(t, 0, form.focusIndex)
		
		// Tab to next field
		msg := tea.KeyMsg{Type: tea.KeyTab}
		model, _ := form.Update(msg)
		updatedForm := model.(*FormComponent)
		
		assert.Equal(t, 1, updatedForm.focusIndex)
		
		// Tab again
		model, _ = updatedForm.Update(msg)
		updatedForm = model.(*FormComponent)
		
		assert.Equal(t, 2, updatedForm.focusIndex)
		
		// Tab wraps around
		model, _ = updatedForm.Update(msg)
		updatedForm = model.(*FormComponent)
		
		assert.Equal(t, 0, updatedForm.focusIndex)
	})
	
	t.Run("Previous field navigation", func(t *testing.T) {
		form.focusIndex = 0
		
		// Shift+Tab to previous field (wraps to last)
		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		model, _ := form.Update(msg)
		updatedForm := model.(*FormComponent)
		
		assert.Equal(t, 2, updatedForm.focusIndex)
		
		// Shift+Tab again
		model, _ = updatedForm.Update(msg)
		updatedForm = model.(*FormComponent)
		
		assert.Equal(t, 1, updatedForm.focusIndex)
	})
	
	t.Run("Navigation with j/k in normal mode", func(t *testing.T) {
		form.focusIndex = 0
		form.insertMode = false
		
		// j to move down
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		model, _ := form.Update(msg)
		updatedForm := model.(*FormComponent)
		
		assert.Equal(t, 1, updatedForm.focusIndex)
		
		// k to move up
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		model, _ = updatedForm.Update(msg)
		updatedForm = model.(*FormComponent)
		
		assert.Equal(t, 0, updatedForm.focusIndex)
	})
}

func TestFormModes(t *testing.T) {
	fields := []FormField{
		{Name: "field1", Label: "Field 1", Type: TextInput},
	}
	
	form := NewForm(WithFields(fields))
	
	t.Run("Enter insert mode", func(t *testing.T) {
		assert.False(t, form.insertMode)
		
		// Press 'i' to enter insert mode
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
		model, _ := form.Update(msg)
		updatedForm := model.(*FormComponent)
		
		assert.True(t, updatedForm.insertMode)
		assert.True(t, updatedForm.IsInsertMode())
	})
	
	t.Run("Exit insert mode", func(t *testing.T) {
		form.insertMode = true
		
		// Press ESC to exit insert mode
		msg := tea.KeyMsg{Type: tea.KeyEsc}
		model, _ := form.Update(msg)
		updatedForm := model.(*FormComponent)
		
		assert.False(t, updatedForm.insertMode)
		assert.False(t, updatedForm.IsInsertMode())
	})
	
	t.Run("SetInsertMode method", func(t *testing.T) {
		form.SetInsertMode(true)
		assert.True(t, form.insertMode)
		
		form.SetInsertMode(false)
		assert.False(t, form.insertMode)
	})
}

func TestFormValues(t *testing.T) {
	fields := []FormField{
		{Name: "username", Label: "Username", Type: TextInput, Value: "john"},
		{Name: "email", Label: "Email", Type: TextInput, Value: "john@example.com"},
		{Name: "bio", Label: "Bio", Type: TextArea, Value: "Software developer"},
	}
	
	form := NewForm(WithFields(fields))
	
	t.Run("Get values", func(t *testing.T) {
		values := form.GetValues()
		
		assert.Equal(t, "john", values["username"])
		assert.Equal(t, "john@example.com", values["email"])
		assert.Equal(t, "Software developer", values["bio"])
	})
	
	t.Run("Set value", func(t *testing.T) {
		form.SetValue("username", "jane")
		
		values := form.GetValues()
		assert.Equal(t, "jane", values["username"])
		
		// Verify the underlying text input was updated
		assert.Equal(t, "jane", form.fields[0].Value)
	})
	
	t.Run("Set value for non-existent field", func(t *testing.T) {
		form.SetValue("nonexistent", "value")
		
		values := form.GetValues()
		_, exists := values["nonexistent"]
		assert.False(t, exists)
	})
}

func TestFormValidation(t *testing.T) {
	fields := []FormField{
		{Name: "required1", Label: "Required 1", Type: TextInput, Required: true, Value: ""},
		{Name: "required2", Label: "Required 2", Type: TextInput, Required: true, Value: "filled"},
		{Name: "optional", Label: "Optional", Type: TextInput, Required: false, Value: ""},
	}
	
	form := NewForm(WithFields(fields))
	
	t.Run("Validation fails for empty required fields", func(t *testing.T) {
		// Try to submit
		msg := tea.KeyMsg{Type: tea.KeyCtrlS}
		model, cmd := form.Update(msg)
		updatedForm := model.(*FormComponent)
		
		// Should have error for required1
		assert.NotEmpty(t, updatedForm.errors["required1"])
		assert.Empty(t, updatedForm.errors["required2"])
		assert.Empty(t, updatedForm.errors["optional"])
		
		// Should not submit (no command returned)
		assert.Nil(t, cmd)
	})
	
	t.Run("Validation passes when required fields are filled", func(t *testing.T) {
		form.SetValue("required1", "now filled")
		form.ClearErrors()
		
		// Try to submit again
		msg := tea.KeyMsg{Type: tea.KeyCtrlS}
		_, cmd := form.Update(msg)
		
		// Should submit (command returned)
		assert.NotNil(t, cmd)
		
		// Execute the command to get the message
		submitMsg := cmd()
		formSubmitMsg, ok := submitMsg.(FormSubmitMsg)
		assert.True(t, ok)
		assert.Equal(t, "now filled", formSubmitMsg.Values["required1"])
		assert.Equal(t, "filled", formSubmitMsg.Values["required2"])
	})
}

func TestFormErrors(t *testing.T) {
	fields := []FormField{
		{Name: "field1", Label: "Field 1", Type: TextInput},
		{Name: "field2", Label: "Field 2", Type: TextInput},
	}
	
	form := NewForm(WithFields(fields))
	
	t.Run("Set and clear errors", func(t *testing.T) {
		form.SetError("field1", "This field has an error")
		form.SetError("field2", "Another error")
		
		assert.Equal(t, "This field has an error", form.errors["field1"])
		assert.Equal(t, "Another error", form.errors["field2"])
		
		form.ClearErrors()
		
		assert.Empty(t, form.errors)
	})
}

func TestFormRendering(t *testing.T) {
	fields := []FormField{
		{Name: "username", Label: "Username", Type: TextInput, Required: true, Value: "john"},
		{Name: "bio", Label: "Bio", Type: TextArea, Required: false, Value: ""},
	}
	
	form := NewForm(
		WithFields(fields),
		WithFormDimensions(80, 24),
	)
	
	t.Run("Render form", func(t *testing.T) {
		view := form.View()
		
		assert.Contains(t, view, "Username")
		assert.Contains(t, view, "*") // Required indicator
		assert.Contains(t, view, "Bio")
		assert.Contains(t, view, "Normal Mode")
		assert.Contains(t, view, "i to edit")
	})
	
	t.Run("Render in insert mode", func(t *testing.T) {
		form.insertMode = true
		view := form.View()
		
		assert.Contains(t, view, "Insert Mode")
		assert.Contains(t, view, "ESC to exit")
	})
	
	t.Run("Render with errors", func(t *testing.T) {
		form.SetError("username", "Username is taken")
		
		renderedField := form.renderField(form.fields[0], true)
		assert.Contains(t, renderedField, "Username is taken")
		assert.Contains(t, renderedField, "↳") // Error indicator
	})
}

func TestFormWindowResize(t *testing.T) {
	form := NewForm()
	
	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 30,
	}
	
	model, _ := form.Update(msg)
	updatedForm := model.(*FormComponent)
	
	assert.Equal(t, 100, updatedForm.width)
	assert.Equal(t, 30, updatedForm.height)
}

func TestFormInit(t *testing.T) {
	form := NewForm()
	cmd := form.Init()
	
	assert.NotNil(t, cmd) // Should return textinput.Blink
}

func TestFormFieldTypes(t *testing.T) {
	t.Run("TextInput field", func(t *testing.T) {
		field := FormField{
			Name:        "text",
			Label:       "Text Field",
			Type:        TextInput,
			Placeholder: "Enter text",
		}
		
		form := NewForm(WithFields([]FormField{field}))
		
		assert.NotNil(t, form.fields[0].textInput)
		assert.Equal(t, "Enter text", form.fields[0].textInput.Placeholder)
	})
	
	t.Run("TextArea field", func(t *testing.T) {
		field := FormField{
			Name:        "area",
			Label:       "Text Area",
			Type:        TextArea,
			Placeholder: "Enter long text",
		}
		
		form := NewForm(WithFields([]FormField{field}))
		
		assert.NotNil(t, form.fields[0].textArea)
		assert.Equal(t, "Enter long text", form.fields[0].textArea.Placeholder)
	})
	
	t.Run("Select field", func(t *testing.T) {
		field := FormField{
			Name:    "select",
			Label:   "Select Field",
			Type:    Select,
			Options: []string{"Option 1", "Option 2", "Option 3"},
			Value:   "Option 1",
		}
		
		form := NewForm(WithFields([]FormField{field}))
		
		// Select is not fully implemented yet
		assert.Equal(t, "Option 1", form.fields[0].Value)
		assert.Equal(t, []string{"Option 1", "Option 2", "Option 3"}, form.fields[0].Options)
	})
}

func TestFormTabNavigation(t *testing.T) {
	fields := []FormField{
		{Name: "field1", Label: "Field 1", Type: TextInput},
		{Name: "field2", Label: "Field 2", Type: TextInput},
	}
	
	form := NewForm(WithFields(fields))
	form.insertMode = true
	
	t.Run("Tab in insert mode", func(t *testing.T) {
		assert.Equal(t, 0, form.focusIndex)
		
		// Tab should move to next field but stay in insert mode
		msg := tea.KeyMsg{Type: tea.KeyTab}
		model, _ := form.Update(msg)
		updatedForm := model.(*FormComponent)
		
		assert.Equal(t, 1, updatedForm.focusIndex)
		assert.True(t, updatedForm.insertMode)
	})
	
	t.Run("Shift+Tab in insert mode", func(t *testing.T) {
		form.focusIndex = 1
		form.insertMode = true
		
		msg := tea.KeyMsg{Type: tea.KeyShiftTab}
		model, _ := form.Update(msg)
		updatedForm := model.(*FormComponent)
		
		assert.Equal(t, 0, updatedForm.focusIndex)
		assert.True(t, updatedForm.insertMode)
	})
}

func TestFormFieldFocus(t *testing.T) {
	fields := []FormField{
		{Name: "field1", Label: "Field 1", Type: TextInput},
		{Name: "field2", Label: "Field 2", Type: TextArea},
	}
	
	form := NewForm(WithFields(fields))
	
	t.Run("Focus current field", func(t *testing.T) {
		form.focusCurrentField()
		assert.True(t, form.fields[0].textInput.Focused())
		
		form.focusIndex = 1
		form.focusCurrentField()
		assert.True(t, form.fields[1].textArea.Focused())
	})
	
	t.Run("Blur current field", func(t *testing.T) {
		form.focusIndex = 0
		form.fields[0].textInput.Focus()
		
		form.blurCurrentField()
		assert.False(t, form.fields[0].textInput.Focused())
	})
}

func TestDefaultFormKeyMap(t *testing.T) {
	keymap := DefaultFormKeyMap()
	
	assert.Equal(t, "tab", keymap.NextField)
	assert.Equal(t, "shift+tab", keymap.PrevField)
	assert.Equal(t, "i", keymap.InsertMode)
	assert.Equal(t, "esc", keymap.NormalMode)
	assert.Equal(t, "ctrl+s", keymap.Submit)
}