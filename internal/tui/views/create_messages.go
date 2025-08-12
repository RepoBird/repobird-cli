package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/utils"
)

// Message types for create view communication
type runCreatedMsg struct {
	run models.RunResponse
	err error
}

type repositorySelectedMsg struct {
	repository string
	err        error
}

type clipboardResultMsg struct {
	success bool
	text    string
}

type configLoadedMsg struct {
	config   *models.RunRequest
	filePath string
	fileHash string
}

type configLoadErrorMsg struct {
	err error
}

type fileSelectorActivatedMsg struct{}

type configSelectorTickMsg time.Time

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Animation/Timer commands - these need to be methods on CreateRunView
func (v *CreateRunView) startYankBlinkAnimation() tea.Cmd {
	return func() tea.Msg {
		// Single blink duration - quick flash (100ms)
		time.Sleep(100 * time.Millisecond)
		return yankBlinkMsg{}
	}
}

func (v *CreateRunView) startClearStatusTimer() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(3 * time.Second)
		return clearStatusMsg{}
	}
}

func (v *CreateRunView) tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return configSelectorTickMsg(t)
	})
}

// Clipboard operations
func (v *CreateRunView) copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		if err := utils.WriteToClipboard(text); err != nil {
			debug.LogToFilef("DEBUG: Failed to copy to clipboard: %v\n", err)
			return clipboardResultMsg{success: false, text: text}
		}
		debug.LogToFilef("DEBUG: Successfully copied to clipboard: %s\n", text)
		return clipboardResultMsg{success: true, text: text}
	}
}