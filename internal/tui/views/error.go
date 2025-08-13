package views

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/messages"
)

// ErrorView displays error messages with optional recovery
type ErrorView struct {
	err         error
	message     string
	recoverable bool
	width       int
	height      int
	keymaps     components.KeyMap
	layout      *components.WindowLayout
}

// NewErrorView creates a new error view
func NewErrorView(err error, message string, recoverable bool) *ErrorView {
	return &ErrorView{
		err:         err,
		message:     message,
		recoverable: recoverable,
		keymaps:     components.DefaultKeyMap,
		layout:      nil, // ⚠️ CRITICAL: Don't initialize here
	}
}

// Init implements tea.Model
func (e *ErrorView) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (e *ErrorView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		e.width = msg.Width
		e.height = msg.Height
		
		// Initialize layout on first WindowSizeMsg only
		if e.layout == nil {
			e.layout = components.NewWindowLayout(msg.Width, msg.Height)
		} else {
			e.layout.Update(msg.Width, msg.Height)
		}
		return e, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "enter", "esc":
			if e.recoverable {
				// Go back to previous view
				return e, func() tea.Msg {
					return messages.NavigateBackMsg{}
				}
			}
			// Go to dashboard (home)
			return e, func() tea.Msg {
				return messages.NavigateToDashboardMsg{}
			}

		case "q", "ctrl+c":
			return e, tea.Quit
		}
	}

	return e, nil
}

// View implements tea.Model
func (e *ErrorView) View() string {
	if e.layout == nil || e.width == 0 || e.height == 0 {
		return "" // Wait for proper dimensions
	}
	
	if !e.layout.IsValidDimensions() {
		return e.layout.GetMinimalView("Error - Loading...")
	}

	// Use WindowLayout for consistent sizing
	boxStyle := e.layout.CreateStandardBox()
	contentStyle := e.layout.CreateContentStyle()

	// Error-specific styles
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginTop(1).
		MarginBottom(2)

	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242")).
		Italic(true)

	// Build content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		errorStyle.Render("⚠ Error"),
		messageStyle.Render(e.message),
	)

	// Add error details if available
	if e.err != nil {
		errorDetails := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render(fmt.Sprintf("Details: %v", e.err))
		content = lipgloss.JoinVertical(lipgloss.Center, content, errorDetails)
	}

	// Add instructions
	var instruction string
	if e.recoverable {
		instruction = "Press Enter or ESC to go back"
	} else {
		instruction = "Press Enter to return to dashboard"
	}
	content = lipgloss.JoinVertical(
		lipgloss.Center,
		content,
		instructionStyle.Render(instruction),
	)

	// Center content within the box
	viewportWidth, viewportHeight := e.layout.GetViewportDimensions()
	centeredContent := contentStyle.
		Width(viewportWidth).
		Height(viewportHeight).
		Align(lipgloss.Center, lipgloss.Center).
		Render(content)

	return boxStyle.Render(centeredContent)
}
