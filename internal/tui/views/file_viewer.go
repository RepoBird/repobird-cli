// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/messages"
)

// FileViewerView wraps the file viewer component as a full view
type FileViewerView struct {
	fileViewer *components.FileViewer
	client     APIClient
	width      int
	height     int
}

// NewFileViewerView creates a new file viewer view
func NewFileViewerView(client APIClient) (*FileViewerView, error) {
	fileViewer, err := components.NewFileViewer(".")
	if err != nil {
		return nil, err
	}

	fileViewer.Focus()

	return &FileViewerView{
		fileViewer: fileViewer,
		client:     client,
	}, nil
}

// Init implements tea.Model
func (v *FileViewerView) Init() tea.Cmd {
	return v.fileViewer.Init()
}

// Update implements tea.Model
func (v *FileViewerView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
		return v.fileViewer.Update(msg)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			// Return to dashboard using navigation message
			return v, func() tea.Msg {
				return messages.NavigateToDashboardMsg{}
			}

		case "enter":
			// Check if file was selected
			model, cmd := v.fileViewer.Update(msg)
			v.fileViewer = model.(*components.FileViewer)

			selectedFile := v.fileViewer.GetSelectedFile()
			if selectedFile != "" {
				// Return to dashboard using navigation message
				// Could set a message about the selected file if needed
				return v, func() tea.Msg {
					return messages.NavigateToDashboardMsg{}
				}
			}
			return v, cmd

		default:
			model, cmd := v.fileViewer.Update(msg)
			v.fileViewer = model.(*components.FileViewer)
			return v, cmd
		}

	default:
		model, cmd := v.fileViewer.Update(msg)
		v.fileViewer = model.(*components.FileViewer)
		return v, cmd
	}
}

// View implements tea.Model
func (v *FileViewerView) View() string {
	// Header
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true).
		Padding(0, 1)

	header := headerStyle.Render("📁 File Viewer")

	// Help text
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	helpText := helpStyle.Render("↑↓/jk: navigate • Enter: select • Tab: toggle preview • q: quit • Backspace: clear filter")

	// Status line
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(0, 1)

	selectedFile := v.fileViewer.GetSelectedFile()
	status := fmt.Sprintf("Selected: %s", selectedFile)
	if selectedFile == "" {
		status = "No file selected"
	}
	statusLine := statusStyle.Render(status)

	// Combine all elements
	var content strings.Builder
	content.WriteString(header)
	content.WriteString("\n")
	content.WriteString(helpText)
	content.WriteString("\n\n")
	content.WriteString(v.fileViewer.View())
	content.WriteString("\n")
	content.WriteString(statusLine)

	return content.String()
}
