package views

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
)

// View renders the create view
func (v *CreateRunView) View() string {
	if v.width <= 0 || v.height <= 0 {
		// Return a styled loading message instead of plain text
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true).
			Render("⟳ Initializing...")
	}

	// Calculate available height for content
	// We have v.height total, minus 1 for statusbar, minus 2 for margin
	availableHeight := v.height - 3
	if availableHeight < 5 {
		availableHeight = 5
	}

	var content string

	if v.error != nil && !v.submitting {
		// Error mode - render error in bordered box similar to form
		content = v.renderErrorLayout(availableHeight)
	} else if v.submitting {
		loadingContent := "⟳ Creating run...\n\nPlease wait..."
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Bold(true).
			Width(v.width).
			Align(lipgloss.Center).
			MarginTop((availableHeight - 2) / 2)
		content = loadingStyle.Render(loadingContent)
	} else if v.useFileInput {
		// File input mode - centered box
		fileContent := v.renderFileInputMode()
		boxWidth := 60
		boxHeight := 10

		fileBoxStyle := lipgloss.NewStyle().
			Width(boxWidth).
			Height(boxHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2)

		fileBox := fileBoxStyle.Render(fileContent)

		// Center the box
		content = lipgloss.Place(
			v.width,
			availableHeight,
			lipgloss.Center,
			lipgloss.Center,
			fileBox,
		)
	} else {
		// Form input mode - single panel layout
		content = v.renderSinglePanelLayout(availableHeight)
	}

	// Create status bar
	statusBar := v.renderStatusBar()

	// Join all components with status bar
	finalView := lipgloss.JoinVertical(
		lipgloss.Left,
		content,
		statusBar,
	)

	// If enhanced config file selector is active, show it as an overlay
	if v.configFileSelector != nil && v.configFileSelector.IsActive() {
		return v.configFileSelector.View()
	}

	// If file selector is active, show modal instead of overlay
	if v.fileSelector != nil && v.fileSelector.IsActive() {
		return v.renderFileSelectionModal(statusBar)
	}

	// If FZF mode is active, overlay the dropdown
	if v.fzfMode != nil && v.fzfMode.IsActive() {
		return v.renderWithFZFOverlay(finalView)
	}

	return finalView
}

// renderSinglePanelLayout renders the form in a single panel with compact fields
func (v *CreateRunView) renderSinglePanelLayout(availableHeight int) string {
	// Account for borders (2 chars for top/bottom) in the content dimensions
	// Width calculation: terminal width minus some padding for cleaner look
	panelWidth := v.width - 2
	if panelWidth < 60 {
		panelWidth = 60
	}

	// Height should fill available space
	panelHeight := availableHeight
	if panelHeight < 3 {
		panelHeight = 3
	}

	// Content dimensions (accounting for border and padding)
	// Border takes 2 from width and height, padding takes another 2 from each
	contentWidth := panelWidth - 4
	contentHeight := panelHeight - 4

	if contentWidth < 40 {
		contentWidth = 40
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Create the single panel content
	panelContent := v.renderCompactForm(contentWidth, contentHeight)

	// Style for single panel - Width includes the border
	// Use Height to maintain consistent window size regardless of content
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight). // Use Height to maintain consistent size
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1)

	// Wrap with margin top to prevent border cutoff
	panel := panelStyle.Render(panelContent)
	return lipgloss.NewStyle().MarginTop(2).Render(panel)
}

// renderCompactForm renders all fields in a compact single-column layout
func (v *CreateRunView) renderCompactForm(width, height int) string {
	var b strings.Builder

	// Add title header inside the form
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	b.WriteString(titleStyle.Render("Create New Run"))
	b.WriteString("\n")

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(24)

	// Load from Config field (new, at index 0)
	b.WriteString(labelStyle.Render("📄 Load Config:"))
	if v.focusIndex == 0 && !v.backButtonFocused && !v.submitButtonFocused {
		b.WriteString(v.renderFieldIndicator())
	} else {
		b.WriteString("   ")
	}

	loadConfigValue := "Press Enter or 'f' to select config file"
	if v.fileSelectorLoading {
		loadConfigValue = "⟳ Loading file selector..."
	} else if v.lastLoadedFile != "" {
		loadConfigValue = fmt.Sprintf("Loaded: %s", filepath.Base(v.lastLoadedFile))
	}

	// Style the load config value based on focus
	loadConfigStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(v.focusIndex == 0 && !v.backButtonFocused && !v.submitButtonFocused)

	if v.focusIndex == 0 && !v.backButtonFocused && !v.submitButtonFocused && v.inputMode == components.NormalMode {
		loadConfigStyle = loadConfigStyle.Background(lipgloss.Color("236"))
	}

	b.WriteString(loadConfigStyle.Render(loadConfigValue))
	b.WriteString("\n")

	// Run type field (selectable, now at index 1)
	b.WriteString(labelStyle.Render("⚙️ Run Type:"))
	if v.focusIndex == 1 && !v.backButtonFocused && !v.submitButtonFocused {
		b.WriteString(v.renderFieldIndicator())
	} else {
		b.WriteString("   ")
	}

	runTypeValue := "▶️ Run (execution)"
	if v.runType == models.RunTypePlan {
		runTypeValue = "📋 Plan (pro-plan)"
	}

	// Style the run type value based on focus
	runTypeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(v.focusIndex == 1 && !v.backButtonFocused && !v.submitButtonFocused)

	if v.focusIndex == 1 && !v.backButtonFocused && !v.submitButtonFocused && v.inputMode == components.NormalMode {
		runTypeStyle = runTypeStyle.Background(lipgloss.Color("236"))
	}

	b.WriteString(runTypeStyle.Render(runTypeValue))
	b.WriteString("\n")

	// Repository field (now at index 2)
	b.WriteString(labelStyle.Render("📁 Repository:"))
	if v.focusIndex == 2 {
		b.WriteString(v.renderFieldIndicator())
	} else {
		b.WriteString("   ")
	}
	v.fields[0].Width = min(width-22, 80)
	b.WriteString(v.fields[0].View())
	b.WriteString("\n")

	// Prompt area (now at index 3) - can be collapsed
	b.WriteString(labelStyle.Render("✏️ Prompt:"))
	if v.focusIndex == 3 {
		b.WriteString(v.renderFieldIndicator())
	} else {
		b.WriteString("   ")
	}

	// Adjust prompt width
	v.promptArea.SetWidth(min(width-22, 100))

	// Show collapsed or full prompt
	if v.promptCollapsed && v.promptArea.Value() != "" {
		// Show first two lines when collapsed
		promptLines := strings.Split(v.promptArea.Value(), "\n")
		collapsedStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Italic(true)

		// Show up to 2 lines
		linesToShow := 2
		if len(promptLines) < linesToShow {
			linesToShow = len(promptLines)
		}

		for i := 0; i < linesToShow; i++ {
			line := promptLines[i]
			if len(line) > width-24 {
				line = line[:width-27] + "..."
			}
			b.WriteString(collapsedStyle.Render(line))
			if i < linesToShow-1 {
				b.WriteString("\n                           ") // Indent continuation
			}
		}

		// Show [+] indicator if there's more content
		if len(promptLines) > 2 {
			b.WriteString(" [+]")
		}
	} else {
		b.WriteString(v.promptArea.View())
	}
	b.WriteString("\n")

	// Other fields in compact layout
	fieldInfo := []struct {
		label string
		index int
	}{
		{"🌿 Source (optional):", 1},
		{"🎯 Target (optional):", 2},
		{"📝 Title (optional):", 3},
		{"📂 Files (optional):", 4},
	}

	for _, field := range fieldInfo {
		b.WriteString(labelStyle.Render(field.label))
		adjustedIndex := field.index + 3 // +3 because load config is at 0, run type is at 1, repository is at 2, prompt is at index 3
		if v.focusIndex == adjustedIndex {
			b.WriteString(v.renderFieldIndicator())
		} else {
			b.WriteString("   ")
		}
		v.fields[field.index].Width = min(width-22, 60)
		b.WriteString(v.fields[field.index].View())
		b.WriteString("\n")
	}

	// Context area (optional, toggled with 'c')
	if v.showContext {
		b.WriteString("\n")
		b.WriteString(labelStyle.Render("💭 Context (optional):"))
		if v.focusIndex == len(v.fields)+3 { // +3 for load config, run type, and prompt
			b.WriteString(v.renderFieldIndicator())
		} else {
			b.WriteString("   ")
		}
		v.contextArea.SetWidth(min(width-22, 100))
		b.WriteString(v.contextArea.View())
	} else if v.contextArea.Value() != "" {
		// Show hint that context exists
		b.WriteString("\n")
		contextHint := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Render("[Press 'c' to show context]")
		b.WriteString(contextHint)
	}

	// Submit button and validation on same line
	b.WriteString("\n\n")
	submitStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true)

	if v.submitButtonFocused {
		if v.inputMode == components.NormalMode {
			submitStyle = submitStyle.
				Background(lipgloss.Color("236")).
				Padding(0, 1)
		}
	}

	// Change text based on submission state
	submitText := "🚀 Submit Run"
	if v.isSubmitting {
		submitText = "⏳ SUBMITTING..."
	}
	b.WriteString(submitStyle.Render(submitText))

	// Validation indicator to the right of submit button
	isValid, validationError := v.validateForm()
	validationStyle := lipgloss.NewStyle().
		Padding(0, 0, 0, 2) // Add left padding to separate from submit button

	if isValid {
		// Show checkmark when valid
		validationStyle = validationStyle.Foreground(lipgloss.Color("82")) // Green
		b.WriteString(validationStyle.Render("✓ Ready to submit"))
	} else {
		// Show error message when invalid
		validationStyle = validationStyle.Foreground(lipgloss.Color("203")) // Red
		b.WriteString(validationStyle.Render("✗ " + validationError))
	}

	// Back button at bottom
	b.WriteString("\n\n")
	backStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	if v.backButtonFocused {
		if v.inputMode == components.InsertMode {
			backStyle = backStyle.
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255"))
		} else {
			backStyle = backStyle.
				Foreground(lipgloss.Color("63")).
				Bold(true)
		}
	}

	b.WriteString(backStyle.Render("← Back to Dashboard"))

	return b.String()
}

// renderErrorLayout renders the error message in a bordered box
func (v *CreateRunView) renderErrorLayout(availableHeight int) string {
	// Calculate box dimensions similar to form layout
	panelWidth := v.width - 2
	if panelWidth < 60 {
		panelWidth = 60
	}

	panelHeight := availableHeight
	if panelHeight < 8 {
		panelHeight = 8
	}

	// Error content
	var b strings.Builder

	// Add title header inside the error panel
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63"))

	b.WriteString(titleStyle.Render("Create New Run"))
	b.WriteString("\n")

	// Error header
	errorHeaderStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	b.WriteString(errorHeaderStyle.Render("❌ Error"))
	b.WriteString("\n\n")

	// Error message row (selectable)
	errorRowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Width(panelWidth - 6) // Account for padding and potential selection indicator

	if v.errorRowFocused {
		// Show selection indicator and highlight when focused
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Render(" ● "))

		// Add blinking effect if recently copied
		if v.yankBlink && !v.yankBlinkTime.IsZero() && time.Since(v.yankBlinkTime) < 2*time.Second {
			if v.yankBlink {
				// Bright green flash
				errorRowStyle = errorRowStyle.
					Background(lipgloss.Color("82")).
					Foreground(lipgloss.Color("0"))
			} else {
				// Normal focused style
				errorRowStyle = errorRowStyle.
					Background(lipgloss.Color("236"))
			}
		} else {
			errorRowStyle = errorRowStyle.
				Background(lipgloss.Color("236"))
		}
	} else {
		b.WriteString("   ")
	}

	b.WriteString(errorRowStyle.Render(v.error.Error()))
	b.WriteString("\n\n")

	// Back button
	backStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63"))

	if v.errorButtonFocused {
		// Show selection indicator and highlight when focused
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("33")).
			Render(" ● "))
		backStyle = backStyle.
			Bold(true).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
	} else {
		b.WriteString("   ")
		backStyle = backStyle.Foreground(lipgloss.Color("240"))
	}

	b.WriteString(backStyle.Render("← Back to Form"))
	b.WriteString("\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	var helpText string
	if v.errorRowFocused {
		helpText = "[j/k] navigate [Enter] back to form [y] copy error [q] back to form [r] retry"
	} else if v.errorButtonFocused {
		helpText = "[j/k] navigate [Enter] back to form [q] back to form [r] retry"
	} else {
		helpText = "[j/k] navigate [Enter] back to form [y] copy error [q] back to form [r] retry"
	}

	b.WriteString(helpStyle.Render(helpText))

	// Style for the panel
	panelStyle := lipgloss.NewStyle().
		Width(panelWidth).
		Height(panelHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")).
		Padding(1)

	return panelStyle.Render(b.String())
}

// renderFieldIndicator renders the field focus indicator
func (v *CreateRunView) renderFieldIndicator() string {
	if v.inputMode == components.InsertMode {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true).
			Render(" ▶ ")
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Render(" ● ")
}

// renderFileInputMode renders the file input interface
func (v *CreateRunView) renderFileInputMode() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Render("Load Task from File"))
	b.WriteString("\n\n")

	b.WriteString("File Path:\n")
	b.WriteString(v.filePathInput.View())
	b.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	b.WriteString(helpStyle.Render("Ctrl+F: Manual input | Ctrl+S: Submit | ESC: Cancel"))

	return b.String()
}

// renderStatusBar renders the status bar at the bottom
func (v *CreateRunView) renderStatusBar() string {
	var statusText string

	// Handle reset confirmation mode - yellow styling like URL mode
	if v.resetConfirmMode {
		// Use yellow background color (226) to match URL opener style
		resetStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("226")).
			Foreground(lipgloss.Color("0")).
			Width(v.width).
			Align(lipgloss.Center)

		statusContent := "[RESET] ⚠️  RESET ALL FIELDS? [y] confirm [n] cancel"
		return resetStyle.Render(statusContent)
	}

	// Handle error mode status
	if v.error != nil && !v.submitting {
		if v.errorRowFocused {
			statusText = "[Enter] back to form [j/k] navigate [y] copy error [q] back to form [r] retry [Q]uit"
		} else if v.errorButtonFocused {
			statusText = "[Enter] back to form [j/k] navigate [q] back to form [r] retry [Q]uit"
		} else {
			statusText = "[Enter] back to form [j/k] navigate [y] copy error [q] back to form [r] retry [Q]uit"
		}

		// Use status line component for consistent formatting
		if v.statusLine != nil {
			return v.statusLine.
				SetWidth(v.width).
				SetLeft("[ERROR]").
				SetRight("").
				SetHelp(statusText).
				Render()
		}
		return statusText
	}

	// Handle submitting state
	if v.isSubmitting {
		statusText = "[ESC] cancel submission [Q] quit"
		if v.statusLine != nil {
			return v.statusLine.
				SetWidth(v.width).
				SetLeft("[SUBMITTING]").
				SetRight("").
				SetHelp(statusText).
				Render()
		}
		return statusText
	}

	// Handle duplicate confirmation mode
	if v.isDuplicateConfirm {
		// Create yellow status line similar to reset confirmation
		duplicateStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).  // Black text
			Background(lipgloss.Color("11")). // Yellow background
			Bold(true).
			Width(v.width).
			Align(lipgloss.Center)

		statusContent := fmt.Sprintf("[DUPLICATE] ⚠️  DUPLICATE RUN DETECTED (ID: %s) - Override? [y] yes [n] no", v.duplicateRunID)
		return duplicateStyle.Render(statusContent)
	}

	if v.inputMode == components.InsertMode {
		if !v.useFileInput && v.focusIndex == 2 {
			// When repository field is focused (now at index 2), show FZF options
			statusText = "[Enter] exit insert [Tab] next [Ctrl+F] fuzzy [Ctrl+R] browse [Ctrl+S] submit"
		} else {
			statusText = "[Enter] exit insert [Tab] next [Ctrl+S] submit [Ctrl+X] clear"
		}
	} else {
		if v.submitButtonFocused {
			// Submit button in normal mode
			statusText = "[q]back [Enter] 🚀 SUBMIT RUN [j/k] navigate [Q]uit"
		} else if v.backButtonFocused {
			// Back button in normal mode
			statusText = "[Enter] back to dashboard [j/k] navigate [Q]uit"
		} else {
			switch v.focusIndex {
			case 0:
				// Load config field in normal mode (index 0)
				statusText = "[q]back [Enter] load config [f] file select [j/k] navigate [c] context [r] reset [Ctrl+S] submit [Q]uit"
			case 1:
				// Run type field in normal mode (index 1)
				statusText = "[q]back [Enter] toggle type [j/k] navigate [c] context [r] reset [Ctrl+S] submit [Q]uit"
			case 2:
				// Repository field in normal mode (index 2)
				statusText = "[q]back [Enter] edit [f] fuzzy [j/k] navigate [c] context [r] reset [Ctrl+S] submit [Q]uit"
			default:
				statusText = "[q]back [Enter] edit [j/k] navigate [c] context [r] reset [Ctrl+S] submit [?] help [Q]uit"
			}
		}
	}

	// Use status line component for consistent formatting with [CREATE] label
	if v.statusLine != nil {
		return v.statusLine.
			SetWidth(v.width).
			SetLeft("[CREATE]").
			SetRight("").
			SetHelp(statusText).
			Render()
	}

	return statusText
}

// renderFileSelectionModal renders a modal for file selection, replacing the entire view
func (v *CreateRunView) renderFileSelectionModal(statusBar string) string {
	if v.fileSelector == nil || !v.fileSelector.IsActive() {
		return ""
	}

	// Create modal content
	modalTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		PaddingLeft(1).
		Render("📄 Select Config File (JSON/Markdown)")

	// Get file selector view
	fileSelectorView := v.fileSelector.View()

	// Create help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		PaddingLeft(1).
		Render("Type to search • ↑↓ to navigate • Enter to select • ESC to cancel")

	// Calculate available height
	availableHeight := v.height - 4 // title + help + statusbar + padding
	if availableHeight < 10 {
		availableHeight = 10
	}

	// Create content area
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		modalTitle,
		"",
		fileSelectorView,
		"",
		helpText,
	)

	// Center the content vertically
	modalContent := lipgloss.Place(
		v.width,
		availableHeight,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	// Join with status bar
	return lipgloss.JoinVertical(
		lipgloss.Left,
		modalContent,
		statusBar,
	)
}

// renderWithFZFOverlay renders the view with FZF dropdown overlay
func (v *CreateRunView) renderWithFZFOverlay(baseView string) string {
	if v.fzfMode == nil || !v.fzfMode.IsActive() {
		return baseView
	}

	// Calculate position for FZF dropdown (repository field)
	// Title + border + load config + run type + repository field = about 5 lines
	yOffset := 5
	xOffset := 19 // After "Repository:    " label and indicator

	return v.renderOverlayDropdown(baseView, v.fzfMode.View(), yOffset, xOffset)
}

// renderOverlayDropdown renders a dropdown overlay on the base view
func (v *CreateRunView) renderOverlayDropdown(baseView, overlayView string, yOffset, xOffset int) string {
	// Split base view into lines
	baseLines := strings.Split(baseView, "\n")

	// Create overlay dropdown view lines
	overlayLines := strings.Split(overlayView, "\n")

	// Create a new view with the dropdown overlaid
	result := make([]string, max(len(baseLines), yOffset+len(overlayLines)))
	copy(result, baseLines)

	// Ensure we have enough lines
	for i := len(baseLines); i < len(result); i++ {
		result[i] = ""
	}

	// Insert dropdown at the calculated position
	for i, overlayLine := range overlayLines {
		lineIdx := yOffset + i
		if lineIdx >= 0 && lineIdx < len(result) {
			// Create the overlay line
			if xOffset < len(result[lineIdx]) {
				// Preserve part of the base line before the dropdown
				basePart := ""
				if xOffset > 0 {
					basePart = result[lineIdx][:min(xOffset, len(result[lineIdx]))]
				}
				// Add the overlay line
				result[lineIdx] = basePart + overlayLine
			} else {
				// Line is shorter than offset, pad and add overlay
				padding := strings.Repeat(" ", max(0, xOffset-len(result[lineIdx])))
				result[lineIdx] = result[lineIdx] + padding + overlayLine
			}
		}
	}

	return strings.Join(result, "\n")
}
