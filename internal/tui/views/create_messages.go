package views

import (
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Message types for create view

type repositorySelectedMsg struct {
	repository string
}

type clipboardResultMsg struct {
	success bool
	text    string
}

type configLoadedMsg struct {
	config interface{}
	path   string
}

type configLoadErrorMsg struct {
	err error
}

type fileSelectorActivatedMsg struct{}

type configSelectorTickMsg time.Time

// Reuse from dashboard messages - commented out to avoid conflict
// type yankBlinkMsg struct{}
// type clearStatusMsg struct{}

type fileHashCacheLoadedMsg struct {
	cache map[string]string
}

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

func (v *CreateRunView) startYankBlinkAnimation() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return yankBlinkMsg{}
	})
}

func (v *CreateRunView) startClearStatusTimer() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return configSelectorTickMsg(t)
	})
}

// Clipboard command

func copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		// Try multiple clipboard commands
		clipboardCommands := [][]string{
			{"pbcopy"},                           // macOS
			{"xclip", "-selection", "clipboard"}, // Linux (X11)
			{"xsel", "--clipboard", "--input"},   // Linux (X11, alternative)
			{"wl-copy"},                          // Linux (Wayland)
		}

		for _, cmdArgs := range clipboardCommands {
			if err := tryClipboardCommand(cmdArgs, text); err == nil {
				return clipboardResultMsg{success: true, text: text}
			}
		}

		return clipboardResultMsg{success: false, text: text}
	}
}

func tryClipboardCommand(cmdArgs []string, text string) error {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
