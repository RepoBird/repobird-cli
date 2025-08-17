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

// startMessageClearTimer starts a timer to trigger UI refresh when message expires
func (d *DashboardView) startMessageClearTimer(duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(duration)
		return messageClearMsg{}
	}
}
