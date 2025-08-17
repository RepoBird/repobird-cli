// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// copyToClipboard copies the given text to clipboard using ClipboardManager
func (d *DashboardView) copyToClipboard(text string) tea.Cmd {
	cmd, err := d.clipboardManager.CopyWithBlink(text, "")
	if err != nil {
		// Handle error - could set error message
		return nil
	}
	return cmd
}

// copyToClipboardWithDescription copies text with a description for feedback
func (d *DashboardView) copyToClipboardWithDescription(text, description string) tea.Cmd {
	cmd, err := d.clipboardManager.CopyWithBlink(text, description)
	if err != nil {
		// Handle error - could set error message
		return nil
	}
	return cmd
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
