package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
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
		layout:      components.NewWindowLayout(80, 24), // Default dimensions like StatusView
		width:       80,
		height:      24,
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
		e.layout.Update(msg.Width, msg.Height)
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

	// Create box
	boxedContent := boxStyle.Render(centeredContent)
	
	// Create status line
	statusLine := e.renderStatusLine()
	
	// Join box and status line directly without gap
	return lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusLine)
}

// renderStatusLine renders the status line with appropriate help text
func (e *ErrorView) renderStatusLine() string {
	var helpText string
	if e.recoverable {
		helpText = "[enter/esc]go back [q]quit"
	} else {
		helpText = "[enter]dashboard [q]quit"
	}
	
	statusLine := components.NewStatusLine().
		SetWidth(e.width).
		SetLeft("[ERROR]").
		SetRight("").
		SetHelp(helpText).
		ResetStyle()
	
	return statusLine.Render()
}
