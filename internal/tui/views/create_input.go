package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/repobird/repobird-cli/internal/tui/components"
)

// Input handling and field management

func (m *CreateRunView) handleInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg.String() {
	case "esc":
		m.inputMode = components.NormalMode
		m.blurAllFields()
		return m, nil
	case "ctrl+f":
		// Activate FZF for repository field if focused on first field
		if m.focusIndex == 0 {
			m.activateFZFMode()
			return m, nil
		}
	case "tab":
		m.nextField()
		return m, nil
	case "shift+tab":
		m.prevField()
		return m, nil
	}

	// Update the currently focused field
	cmds = append(cmds, m.updateFields(msg)...)
	
	return m, tea.Batch(cmds...)
}

func (m *CreateRunView) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "i":
		m.inputMode = components.InsertMode
		m.updateFocus()
		return m, nil
	case "j", "down":
		m.nextField()
		return m, nil
	case "k", "up":
		m.prevField()
		return m, nil
	case "f":
		// Activate FZF for current field if it's repository
		if m.focusIndex == 0 {
			m.activateFZFMode()
			return m, nil
		}
	case "enter":
		if m.submitButtonFocused {
			return m, m.submitRun()
		}
	case "q":
		m.exitRequested = true
		return m, tea.Quit
	}

	return m, nil
}

func (m *CreateRunView) handleErrorMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		// Return to previous state
		m.inputMode = components.NormalMode
		m.error = nil
		m.restorePreviousFocus()
		return m, nil
	case "j", "down":
		m.errorRowFocused = true
		return m, nil
	case "k", "up":
		m.errorRowFocused = false
		return m, nil
	}

	return m, nil
}

func (m *CreateRunView) updateFields(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// Update text input fields
	for i := range m.fields {
		if i == m.focusIndex {
			var cmd tea.Cmd
			m.fields[i], cmd = m.fields[i].Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	// Update text areas
	if m.focusIndex == len(m.fields) {
		var cmd tea.Cmd
		m.promptArea, cmd = m.promptArea.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	} else if m.focusIndex == len(m.fields)+1 {
		var cmd tea.Cmd
		m.contextArea, cmd = m.contextArea.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return cmds
}

func (m *CreateRunView) nextField() {
	totalFields := len(m.fields) + 2 + 2 // fields + text areas + buttons
	m.focusIndex = (m.focusIndex + 1) % totalFields
	m.updateFocus()
}

func (m *CreateRunView) prevField() {
	totalFields := len(m.fields) + 2 + 2 // fields + text areas + buttons
	m.focusIndex = (m.focusIndex - 1 + totalFields) % totalFields
	m.updateFocus()
}

func (m *CreateRunView) updateFocus() {
	// Reset all focus states
	m.blurAllFields()
	m.backButtonFocused = false
	m.submitButtonFocused = false

	// Set focus based on current index
	if m.focusIndex < len(m.fields) {
		m.fields[m.focusIndex].Focus()
	} else if m.focusIndex == len(m.fields) {
		m.promptArea.Focus()
	} else if m.focusIndex == len(m.fields)+1 {
		m.contextArea.Focus()
	} else if m.focusIndex == len(m.fields)+2 {
		m.backButtonFocused = true
	} else if m.focusIndex == len(m.fields)+3 {
		m.submitButtonFocused = true
	}
}

func (m *CreateRunView) blurAllFields() {
	for i := range m.fields {
		m.fields[i].Blur()
	}
	m.promptArea.Blur()
	m.contextArea.Blur()
}

func (m *CreateRunView) clearAllFields() {
	for i := range m.fields {
		m.fields[i].SetValue("")
	}
	m.promptArea.SetValue("")
	m.contextArea.SetValue("")
}

func (m *CreateRunView) clearCurrentField() {
	if m.focusIndex < len(m.fields) {
		m.fields[m.focusIndex].SetValue("")
	} else if m.focusIndex == len(m.fields) {
		m.promptArea.SetValue("")
	} else if m.focusIndex == len(m.fields)+1 {
		m.contextArea.SetValue("")
	}
}

func (m *CreateRunView) initializeInputFields() {
	// Initialize text input fields
	m.fields = make([]textinput.Model, 4)
	
	fieldNames := []string{"Repository", "Source Branch", "Target Branch", "Title"}
	for i, name := range fieldNames {
		field := textinput.New()
		field.Placeholder = name
		field.Width = 50
		if i == 0 {
			field.Focus()
		}
		m.fields[i] = field
	}

	// Initialize text areas
	m.promptArea = textarea.New()
	m.promptArea.Placeholder = "Enter your prompt here..."
	m.promptArea.SetWidth(60)
	m.promptArea.SetHeight(6)

	m.contextArea = textarea.New()
	m.contextArea.Placeholder = "Additional context (optional)..."
	m.contextArea.SetWidth(60)
	m.contextArea.SetHeight(4)
}

func (m *CreateRunView) initErrorFocus() {
	m.prevFocusIndex = m.focusIndex
	m.prevBackButtonFocused = m.backButtonFocused
	m.prevSubmitButtonFocused = m.submitButtonFocused
	m.errorButtonFocused = true
	m.errorRowFocused = false
}

func (m *CreateRunView) restorePreviousFocus() {
	m.focusIndex = m.prevFocusIndex
	m.backButtonFocused = m.prevBackButtonFocused
	m.submitButtonFocused = m.prevSubmitButtonFocused
	m.errorButtonFocused = false
	m.updateFocus()
}