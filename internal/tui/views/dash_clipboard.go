package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/utils"
)

// copyToClipboard copies the given text to clipboard
func (d *DashboardView) copyToClipboard(text string) error {
	return utils.WriteToClipboard(text)
}

// startYankBlinkAnimation starts the single blink animation for clipboard feedback
func (d *DashboardView) startYankBlinkAnimation() tea.Cmd {
	return func() tea.Msg {
		// Single blink duration - visible flash (150ms)
		time.Sleep(150 * time.Millisecond)
		return yankBlinkMsg{}
	}
}

// startMessageClearTimer starts a timer to trigger UI refresh when message expires
func (d *DashboardView) startMessageClearTimer(duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duration)
		return messageClearMsg{}
	}
}

// startClearStatusTimer starts a timer to clear the status message
func (d *DashboardView) startClearStatusTimer() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(250 * time.Millisecond)
		return clearStatusMsg{}
	}
}
