// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RunEditor component handles editing individual runs
type RunEditor struct {
	run          *BulkRunItem
	promptInput  textinput.Model
	titleInput   textinput.Model
	targetInput  textinput.Model
	contextInput textinput.Model
	focusedField int
}

func NewRunEditor() *RunEditor {
	prompt := textinput.New()
	prompt.Placeholder = "Enter prompt (required)"
	prompt.CharLimit = 500
	prompt.Focus()

	title := textinput.New()
	title.Placeholder = "Enter title (optional)"
	title.CharLimit = 100

	target := textinput.New()
	target.Placeholder = "Enter target branch (optional)"
	target.CharLimit = 100

	context := textinput.New()
	context.Placeholder = "Enter context (optional)"
	context.CharLimit = 500

	return &RunEditor{
		promptInput:  prompt,
		titleInput:   title,
		targetInput:  target,
		contextInput: context,
		focusedField: 0,
	}
}

func (e *RunEditor) SetRun(run *BulkRunItem) {
	e.run = run
	e.promptInput.SetValue(run.Prompt)
	e.titleInput.SetValue(run.Title)
	e.targetInput.SetValue(run.Target)
	e.contextInput.SetValue(run.Context)
}

func (e *RunEditor) UpdateRunEditor(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			e.focusedField++
			if e.focusedField > 3 {
				e.focusedField = 0
			}
			e.updateFocus()
		case "shift+tab", "up":
			e.focusedField--
			if e.focusedField < 0 {
				e.focusedField = 3
			}
			e.updateFocus()
		case "enter":
			// Save changes
			if e.run != nil {
				e.run.Prompt = e.promptInput.Value()
				e.run.Title = e.titleInput.Value()
				e.run.Target = e.targetInput.Value()
				e.run.Context = e.contextInput.Value()
			}
			// Return to list mode (handled by parent)
		}
	}

	// Update inputs
	var cmd tea.Cmd
	e.promptInput, cmd = e.promptInput.Update(msg)
	cmds = append(cmds, cmd)

	e.titleInput, cmd = e.titleInput.Update(msg)
	cmds = append(cmds, cmd)

	e.targetInput, cmd = e.targetInput.Update(msg)
	cmds = append(cmds, cmd)

	e.contextInput, cmd = e.contextInput.Update(msg)
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (e *RunEditor) View() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	title := titleStyle.Render("Edit Run")

	fields := []string{
		"Prompt (required):",
		e.promptInput.View(),
		"",
		"Title (optional):",
		e.titleInput.View(),
		"",
		"Target Branch (optional):",
		e.targetInput.View(),
		"",
		"Context (optional):",
		e.contextInput.View(),
	}

	content := lipgloss.JoinVertical(lipgloss.Left, fields...)

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		"tab/↓: next field | shift+tab/↑: prev field | enter: save | esc: cancel",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		content,
		"",
		help,
	)
}

func (e *RunEditor) updateFocus() {
	e.promptInput.Blur()
	e.titleInput.Blur()
	e.targetInput.Blur()
	e.contextInput.Blur()

	switch e.focusedField {
	case 0:
		e.promptInput.Focus()
	case 1:
		e.titleInput.Focus()
	case 2:
		e.targetInput.Focus()
	case 3:
		e.contextInput.Focus()
	}
}
