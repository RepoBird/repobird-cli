package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// Use gKeyTimeoutMsg from dash_messages.go to avoid duplication

// initializeInputFields initializes all the input fields for the form
func (v *CreateRunView) initializeInputFields() {
	// Initialize text inputs for the fields
	repoInput := textinput.New()
	repoInput.Placeholder = "username/repository"
	repoInput.Width = 50
	repoInput.CharLimit = 100

	sourceInput := textinput.New()
	sourceInput.Placeholder = "main"
	sourceInput.Width = 50
	sourceInput.CharLimit = 100

	targetInput := textinput.New()
	targetInput.Placeholder = "feature/my-change"
	targetInput.Width = 50
	targetInput.CharLimit = 100

	titleInput := textinput.New()
	titleInput.Placeholder = "Brief title for this run"
	titleInput.Width = 50
	titleInput.CharLimit = 200

	filesInput := textinput.New()
	filesInput.Placeholder = "src/main.go,pkg/utils.go (comma-separated)"
	filesInput.Width = 50
	filesInput.CharLimit = 500

	v.fields = []textinput.Model{
		repoInput,
		sourceInput,
		targetInput,
		titleInput,
		filesInput,
	}

	// Initialize text areas
	v.promptArea = textarea.New()
	v.promptArea.Placeholder = "Describe what you want to build or change..."
	v.promptArea.SetWidth(60)
	v.promptArea.SetHeight(4)
	v.promptArea.CharLimit = 5000

	v.contextArea = textarea.New()
	v.contextArea.Placeholder = "Additional context or requirements (optional)..."
	v.contextArea.SetWidth(60)
	v.contextArea.SetHeight(4)
	v.contextArea.CharLimit = 5000

	// Initialize file path input
	v.filePathInput = textinput.New()
	v.filePathInput.Placeholder = "path/to/task.json"
	v.filePathInput.Width = 50

	// Initialize status line
	v.statusLine = components.NewStatusLine()
}

// handleInsertMode handles keyboard input in insert mode
func (v *CreateRunView) handleInsertMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Always check for ESC first
	if msg.String() == "esc" {
		v.inputMode = components.NormalMode
		v.exitRequested = false
		v.blurAllFields()
		return v, nil
	}

	// Handle duplicate confirmation mode in insert mode
	if v.isDuplicateConfirm {
		switch msg.String() {
		case "y", "Y":
			// Force submit
			v.isDuplicateConfirm = false
			debug.LogToFile("DEBUG: User confirmed force submit for duplicate run\n")
			return v, v.submitWithForce()
		case "n", "N", "esc":
			// Cancel submission
			v.isDuplicateConfirm = false
			v.duplicateRunID = ""
			v.pendingTask = models.RunRequest{}
			debug.LogToFile("DEBUG: User cancelled duplicate run submission\n")
			return v, nil
		default:
			// Ignore other keys in duplicate confirm mode
			return v, nil
		}
	}

	switch {
	case msg.String() == "ctrl+f":
		// Activate FZF mode for repository field if focused on it (now index 1)
		if !v.useFileInput && v.focusIndex == 1 && !v.fzfActive {
			v.activateFZFMode()
			return v, nil
		} else {
			// Original file input toggle behavior
			v.useFileInput = !v.useFileInput
			if v.useFileInput {
				v.filePathInput.Focus()
			} else {
				v.fields[0].Focus()
				v.focusIndex = 1 // Repository is now at index 1
			}
		}
	case msg.String() == "ctrl+r":
		// Trigger repository selector when repository field is focused
		if !v.useFileInput && v.focusIndex == 1 {
			return v, v.selectRepository()
		}
	case msg.String() == "ctrl+s" || msg.String() == "ctrl+enter":
		if !v.submitting && !v.isSubmitting {
			debug.LogToFile("DEBUG: Ctrl+S pressed in INSERT MODE - submitting run\n")
			return v, v.submitRun()
		}
	case msg.String() == "enter":
		// For prompt area (index 3) or context area, allow Enter for newlines
		if v.focusIndex == 3 || (v.showContext && v.focusIndex == len(v.fields)+3) {
			// Handle text input for prompt/context areas - Enter creates newlines
			cmds = append(cmds, v.updateFields(msg)...)
		} else {
			// For other fields, Exit insert mode when Enter is pressed
			v.inputMode = components.NormalMode
			v.exitRequested = false
			v.blurAllFields()
			return v, nil
		}
	case key.Matches(msg, v.keys.Tab):
		v.nextField()
	case key.Matches(msg, v.keys.ShiftTab):
		v.prevField()
	default:
		// Handle text input
		if v.useFileInput {
			var cmd tea.Cmd
			v.filePathInput, cmd = v.filePathInput.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			cmds = append(cmds, v.updateFields(msg)...)
		}
	}

	return v, tea.Batch(cmds...)
}

// handleNormalMode handles keyboard input in normal mode
func (v *CreateRunView) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle error state first
	if v.error != nil && !v.submitting {
		return v.handleErrorMode(msg)
	}

	// Handle reset confirmation mode
	if v.resetConfirmMode {
		switch msg.String() {
		case "y":
			// Confirm reset - clear all fields and cache
			v.clearAllFields()
			v.resetConfirmMode = false
			v.lastLoadedFile = ""
			v.runType = models.RunTypeRun
			v.showContext = false
			v.promptCollapsed = false
			debug.LogToFile("DEBUG: User confirmed reset - all fields cleared\n")
			return v, nil
		case "n", "esc":
			// Cancel reset
			v.resetConfirmMode = false
			debug.LogToFile("DEBUG: User cancelled reset\n")
			return v, nil
		default:
			// Ignore other keys in reset confirm mode
			return v, nil
		}
	}

	switch {
	case msg.String() == "Q":
		// Capital Q to force quit from anywhere
		return v, tea.Quit
	case key.Matches(msg, v.keys.Quit), key.Matches(msg, v.keys.Back), msg.String() == "esc":
		// q, b, or ESC all go back to dashboard
		if !v.submitting {
			v.saveFormData()
			debug.LogToFile("DEBUG: CreateView q/b/ESC - returning to dashboard\n")
			// Return to dashboard view
			dashboard := NewDashboardView(v.client)
			dashboard.width = v.width
			dashboard.height = v.height
			return dashboard, dashboard.Init()
		}
	case key.Matches(msg, v.keys.Help):
		// Return to dashboard and show docs
		dashboard := NewDashboardView(v.client)
		dashboard.width = v.width
		dashboard.height = v.height
		dashboard.showDocs = true
		dashboard.docsCurrentPage = 4 // Show Create Run Form page
		return dashboard, dashboard.Init()
	case msg.String() == "i":
		// 'i' enters insert mode
		v.inputMode = components.InsertMode
		v.exitRequested = false
		v.updateFocus()
	case msg.String() == "enter":
		if v.backButtonFocused {
			// Enter on back button returns to dashboard
			v.saveFormData()
			debug.LogToFile("DEBUG: Back button pressed - returning to dashboard\n")
			dashboard := NewDashboardView(v.client)
			dashboard.width = v.width
			dashboard.height = v.height
			return dashboard, dashboard.Init()
		} else if v.submitButtonFocused {
			// Enter on submit button submits the run
			if !v.submitting && !v.isSubmitting {
				debug.LogToFile("DEBUG: Submit button pressed - submitting run\n")
				return v, v.submitRun()
			}
		} else if v.focusIndex == 0 {
			// Enter on load config field (index 0) activates file selector
			// Don't process if already loading or active
			if !v.fileSelectorLoading && !v.configFileSelectorActive {
				v.fileSelectorLoading = true
				return v, v.activateConfigFileSelector()
			}
		} else if v.focusIndex == 1 {
			// Enter on run type field (index 1) toggles it
			if v.runType == models.RunTypeRun {
				v.runType = models.RunTypePlan
			} else {
				v.runType = models.RunTypeRun
			}
		} else if v.focusIndex == 2 {
			// Repository field - enter insert mode
			v.inputMode = components.InsertMode
			v.exitRequested = false
			v.updateFocus()
		} else {
			// Enter on other fields enters insert mode
			v.inputMode = components.InsertMode
			v.exitRequested = false
			v.updateFocus()
		}
	case key.Matches(msg, v.keys.Up) || msg.String() == "k":
		v.prevField()
	case key.Matches(msg, v.keys.Down) || msg.String() == "j":
		v.nextField()
	case msg.String() == "ctrl+s":
		if !v.submitting && !v.isSubmitting {
			debug.LogToFile("DEBUG: Ctrl+S pressed in NORMAL MODE - submitting run\n")
			return v, v.submitRun()
		}
	case msg.String() == "ctrl+l":
		v.clearAllFields()
	case msg.String() == "ctrl+x":
		v.clearCurrentField()
	case msg.String() == "f":
		// In normal mode, 'f' activates FZF
		if v.focusIndex == 0 {
			// Load config field - activate file selector
			// Don't process if already loading or active
			if !v.fileSelectorLoading && !v.configFileSelectorActive {
				v.fileSelectorLoading = true
				return v, v.activateConfigFileSelector()
			}
		} else if v.focusIndex == 2 && !v.fzfActive {
			// Repository field - activate FZF for repository selection
			v.activateFZFMode()
			return v, nil
		}
	case msg.String() == "c":
		// Toggle context field visibility
		v.showContext = !v.showContext
	case msg.String() == "t":
		// Toggle between run types
		if v.runType == models.RunTypeRun {
			v.runType = models.RunTypePlan
		} else {
			v.runType = models.RunTypeRun
		}
	case msg.String() == "d":
		// Delete current field value for string input fields only (not load config or run type)
		if v.focusIndex != 0 && v.focusIndex != 1 { // Skip load config field (index 0) and run type field (index 1)
			v.clearCurrentField()
		}
	case msg.String() == "r":
		// Enter reset confirmation mode
		v.resetConfirmMode = true
		debug.LogToFile("DEBUG: Entering reset confirmation mode\n")
		return v, nil
	case msg.String() == "G":
		// Vim: Go to bottom (last field or submit button)
		v.waitingForG = false // Cancel any pending 'gg' command
		// Calculate total fields: 1 (load config) + 1 (run type) + 1 (repo) + 1 (prompt) + 4 (other fields) + context (if shown)
		totalFields := len(v.fields) + 3 // +1 for load config, +1 for run type, +1 for prompt
		if v.showContext {
			totalFields++ // +1 for context
		}
		// Go to submit button (which is after all fields)
		v.focusIndex = totalFields
		v.submitButtonFocused = true
		v.backButtonFocused = false
		v.updateFocus()
		return v, nil
	case msg.String() == "g":
		if v.waitingForG {
			// This is the second 'g' in 'gg' - go to top (first field)
			v.waitingForG = false
			v.focusIndex = 0 // Go to load config field
			v.submitButtonFocused = false
			v.backButtonFocused = false
			v.updateFocus()
		} else {
			// First 'g' pressed - wait for second 'g'
			v.waitingForG = true
			v.lastGPressTime = time.Now()
			// Start a timer to cancel the 'gg' command after 1 second
			return v, tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
				return gKeyTimeoutMsg{}
			})
		}
		return v, nil
	default:
		// Cancel any pending 'gg' command if another key is pressed
		if v.waitingForG {
			v.waitingForG = false
		}
		// Block vim navigation keys from doing anything else
	}

	return v, nil
}

// handleErrorMode handles keyboard input when in error mode
func (v *CreateRunView) handleErrorMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if v.errorButtonFocused {
			// Clear error and restore previous focus
			v.error = nil
			v.restorePreviousFocus()
			return v, nil
		} else if v.errorRowFocused {
			// Copy error to clipboard when row is selected
			errorText := fmt.Sprintf("Error: %v", v.error)
			return v, copyToClipboard(errorText)
		}
	case "y":
		// Copy error to clipboard
		errorText := fmt.Sprintf("Error: %v", v.error)
		return v, copyToClipboard(errorText)
	case "esc", "q":
		// Clear error and restore previous focus
		v.error = nil
		v.restorePreviousFocus()
		return v, nil
	case "up", "k":
		// Toggle between error row and OK button
		v.errorRowFocused = !v.errorRowFocused
		v.errorButtonFocused = !v.errorButtonFocused
	case "down", "j":
		// Toggle between error row and OK button
		v.errorRowFocused = !v.errorRowFocused
		v.errorButtonFocused = !v.errorButtonFocused
	case "tab":
		// Tab also toggles between error row and OK button
		v.errorRowFocused = !v.errorRowFocused
		v.errorButtonFocused = !v.errorButtonFocused
	}
	return v, nil
}

// updateFields handles text input for the current focused field
func (v *CreateRunView) updateFields(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	// Adjusted indices to account for load config and run type fields
	if v.focusIndex == 2 {
		// Repository field (index 2)
		var cmd tea.Cmd
		v.fields[0], cmd = v.fields[0].Update(msg)
		cmds = append(cmds, cmd)
	} else if v.focusIndex == 3 {
		// Prompt field (using textarea)
		var cmd tea.Cmd
		v.promptArea, cmd = v.promptArea.Update(msg)
		cmds = append(cmds, cmd)
	} else if v.focusIndex >= 4 && v.focusIndex < len(v.fields)+3 {
		// Other regular text fields (source, target, title, files)
		fieldIdx := v.focusIndex - 3
		var cmd tea.Cmd
		v.fields[fieldIdx], cmd = v.fields[fieldIdx].Update(msg)
		cmds = append(cmds, cmd)
	} else if v.showContext && v.focusIndex == len(v.fields)+3 {
		// Context field (only if shown)
		var cmd tea.Cmd
		v.contextArea, cmd = v.contextArea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return cmds
}

// nextField moves focus to the next field
func (v *CreateRunView) nextField() {
	// Calculate total fields
	totalFields := len(v.fields) + 3 // +1 for load config, +1 for run type, +1 for prompt
	if v.showContext {
		totalFields++ // +1 for context
	}

	// Handle button navigation
	if v.backButtonFocused {
		v.backButtonFocused = false
		v.submitButtonFocused = true
		return
	}
	if v.submitButtonFocused {
		v.submitButtonFocused = false
		v.focusIndex = 0 // Wrap to first field
		v.updateFocus()
		return
	}

	// Regular field navigation
	v.focusIndex++
	if v.focusIndex >= totalFields {
		// Move to back button
		v.focusIndex = totalFields - 1 // Keep index at last field
		v.backButtonFocused = true
		v.blurAllFields()
	} else {
		v.updateFocus()
	}
}

// prevField moves focus to the previous field
func (v *CreateRunView) prevField() {
	// Handle button navigation
	if v.submitButtonFocused {
		v.submitButtonFocused = false
		v.backButtonFocused = true
		return
	}
	if v.backButtonFocused {
		v.backButtonFocused = false
		// Calculate total fields
		totalFields := len(v.fields) + 3 // +1 for load config, +1 for run type, +1 for prompt
		if v.showContext {
			totalFields++ // +1 for context
		}
		v.focusIndex = totalFields - 1 // Go to last field
		v.updateFocus()
		return
	}

	// Regular field navigation
	v.focusIndex--
	if v.focusIndex < 0 {
		// Wrap to submit button
		v.focusIndex = 0
		v.submitButtonFocused = true
		v.blurAllFields()
	} else {
		v.updateFocus()
	}
}

// updateFocus updates which field has focus based on focusIndex
func (v *CreateRunView) updateFocus() {
	v.blurAllFields()

	// Special handling for buttons
	if v.backButtonFocused || v.submitButtonFocused {
		return
	}

	// Skip focus for load config and run type fields (indices 0 and 1)
	if v.focusIndex == 0 || v.focusIndex == 1 {
		// These are handled by normal mode navigation, don't focus them
		return
	}

	// Focus the appropriate field based on adjusted index
	if v.focusIndex == 2 {
		// Repository field
		v.fields[0].Focus()
	} else if v.focusIndex == 3 {
		// Prompt field
		v.promptArea.Focus()
	} else if v.focusIndex >= 4 && v.focusIndex < len(v.fields)+3 {
		// Other text fields (source, target, title, files)
		fieldIdx := v.focusIndex - 3
		v.fields[fieldIdx].Focus()
	} else if v.showContext && v.focusIndex == len(v.fields)+3 {
		// Context field (only if shown)
		v.contextArea.Focus()
	}
}

// blurAllFields removes focus from all fields
func (v *CreateRunView) blurAllFields() {
	for i := range v.fields {
		v.fields[i].Blur()
	}
	v.promptArea.Blur()
	v.contextArea.Blur()
	v.filePathInput.Blur()
}

// clearCurrentField clears the currently focused field
func (v *CreateRunView) clearCurrentField() {
	if v.focusIndex == 2 {
		// Repository field
		v.fields[0].SetValue("")
	} else if v.focusIndex == 3 {
		// Prompt field
		v.promptArea.SetValue("")
	} else if v.focusIndex >= 4 && v.focusIndex < len(v.fields)+3 {
		// Other text fields
		fieldIdx := v.focusIndex - 3
		v.fields[fieldIdx].SetValue("")
	} else if v.showContext && v.focusIndex == len(v.fields)+3 {
		// Context field
		v.contextArea.SetValue("")
	}
}

// restorePreviousFocus restores the focus state from before error mode
func (v *CreateRunView) restorePreviousFocus() {
	v.inputMode = components.NormalMode
	v.focusIndex = v.prevFocusIndex
	v.backButtonFocused = v.prevBackButtonFocused
	v.submitButtonFocused = v.prevSubmitButtonFocused
	v.errorButtonFocused = false
	v.errorRowFocused = false
}
