package components

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/prompts"
)

// ValidationPromptView is a TUI component for displaying validation prompts
type ValidationPromptView struct {
	handler       *prompts.ValidationPromptHandler
	currentPrompt int
	responses     []string
	finished      bool
	cancelled     bool
	prompts       []prompts.ValidationPrompt
}

// ValidationPromptMsg is sent when validation prompts are completed
type ValidationPromptMsg struct {
	Cancelled bool
	Responses []string
}

// NewValidationPromptView creates a new validation prompt view
func NewValidationPromptView(handler *prompts.ValidationPromptHandler) *ValidationPromptView {
	return &ValidationPromptView{
		handler:       handler,
		currentPrompt: 0,
		responses:     []string{},
		finished:      false,
		cancelled:     false,
		prompts:       handler.GetPrompts(), // We'll need to add this method
	}
}

func (v *ValidationPromptView) Init() tea.Cmd {
	return nil
}

func (v *ValidationPromptView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			v.responses = append(v.responses, "y")
			v.currentPrompt++
			if v.currentPrompt >= len(v.prompts) {
				v.finished = true
				return v, v.finishCmd()
			}

		case "n", "N":
			v.responses = append(v.responses, "n")
			// Check if this is a required prompt
			if v.currentPrompt < len(v.prompts) && v.prompts[v.currentPrompt].Required {
				v.cancelled = true
				v.finished = true
				return v, v.finishCmd()
			}
			v.currentPrompt++
			if v.currentPrompt >= len(v.prompts) {
				v.finished = true
				return v, v.finishCmd()
			}

		case "enter":
			// Default response
			defaultResp := "y"
			if v.currentPrompt < len(v.prompts) && v.prompts[v.currentPrompt].DefaultNo {
				defaultResp = "n"
			}
			v.responses = append(v.responses, defaultResp)

			if defaultResp == "n" && v.currentPrompt < len(v.prompts) && v.prompts[v.currentPrompt].Required {
				v.cancelled = true
				v.finished = true
				return v, v.finishCmd()
			}

			v.currentPrompt++
			if v.currentPrompt >= len(v.prompts) {
				v.finished = true
				return v, v.finishCmd()
			}

		case "esc", "ctrl+c":
			v.cancelled = true
			v.finished = true
			return v, v.finishCmd()
		}
	}

	return v, nil
}

func (v *ValidationPromptView) View() string {
	if v.finished {
		return ""
	}

	if v.currentPrompt >= len(v.prompts) {
		return ""
	}

	prompt := v.prompts[v.currentPrompt]

	// Style definitions
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11")).
		Bold(true)

	numberStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	defaultStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("245"))

	var promptText string
	if len(v.prompts) > 1 {
		promptText = numberStyle.Render(fmt.Sprintf("[%d/%d] ", v.currentPrompt+1, len(v.prompts)))
	}

	promptText += promptStyle.Render(prompt.Message)

	// Add default indicator
	if prompt.DefaultNo {
		promptText += defaultStyle.Render(" [y/N]: ")
	} else {
		promptText += defaultStyle.Render(" [Y/n]: ")
	}

	// Add help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("\n\nPress Y/N to respond, Enter for default, ESC to cancel")

	return promptText + "\n" + helpText
}

func (v *ValidationPromptView) finishCmd() tea.Cmd {
	return func() tea.Msg {
		return ValidationPromptMsg{
			Cancelled: v.cancelled,
			Responses: v.responses,
		}
	}
}

// Helper key bindings
type validationKeys struct {
	Yes    key.Binding
	No     key.Binding
	Enter  key.Binding
	Cancel key.Binding
}

var ValidationKeys = validationKeys{
	Yes: key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y", "yes"),
	),
	No: key.NewBinding(
		key.WithKeys("n", "N"),
		key.WithHelp("n", "no"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "default"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "cancel"),
	),
}
