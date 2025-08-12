package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
)

// Message types for create view communication
type runCreatedMsg struct {
	run models.RunResponse
}

type repositorySelectedMsg struct {
	repository string
}

type clipboardResultMsg struct {
	success bool
}

type configLoadedMsg struct {
	config   *models.RunRequest
	filePath string
}

type configLoadErrorMsg struct {
	err      error
	filePath string
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

// Animation/Timer commands
func startYankBlinkAnimation() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return t
	})
}

func startClearStatusTimer() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return t
	})
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return configSelectorTickMsg(t)
	})
}

// Clipboard operations
func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		return clipboardResultMsg{success: true}
	}
}