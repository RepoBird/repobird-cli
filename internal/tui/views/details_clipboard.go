// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/utils"
)

// copyWithFeedback copies text to clipboard with status feedback
func (v *RunDetailsView) copyWithFeedback(text, description string) tea.Cmd {
	if text == "" {
		v.statusLine.SetTemporaryMessageWithType("✗ No content to copy", components.MessageError, 100*time.Millisecond)
		return nil
	}

	cmd, err := v.clipboardManager.CopyWithBlink(text, description)
	if err != nil {
		v.statusLine.SetTemporaryMessageWithType("✗ Failed to copy", components.MessageError, 100*time.Millisecond)
		return nil
	}

	// Truncate for display
	displayText := truncateForDisplay(text, 30)
	message := fmt.Sprintf("📋 Copied %s", description)
	if description == "" {
		message = fmt.Sprintf("📋 Copied \"%s\"", displayText)
	}

	v.statusLine.SetTemporaryMessageWithType(message, components.MessageSuccess, 100*time.Millisecond)
	return cmd
}

// truncateForDisplay truncates text for display in status messages
func truncateForDisplay(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// handleClipboardOperations handles clipboard-related key presses
func (v *RunDetailsView) handleClipboardOperations(key string) tea.Cmd {
	switch key {
	case "y":
		// Copy selected field or current line
		var textToCopy string
		var description string

		if v.navigationMode && v.selectedRow >= 0 && v.selectedRow < len(v.fieldValues) {
			textToCopy = v.fieldValues[v.selectedRow]
		} else {
			textToCopy = v.getCurrentLine()
		}

		return v.copyWithFeedback(textToCopy, description)

	case "Y":
		// Copy all content to clipboard
		cmd, err := v.clipboardManager.CopyWithBlink(v.fullContent, "all content")
		if err != nil {
			v.statusLine.SetTemporaryMessageWithType("✗ Failed to copy", components.MessageError, 100*time.Millisecond)
			return nil
		}
		v.statusLine.SetTemporaryMessageWithType("📋 Copied all content", components.MessageSuccess, 100*time.Millisecond)
		return cmd

	case "o":
		// Open URL in browser if current selection contains a URL
		var urlText string
		if v.navigationMode && v.selectedRow >= 0 && v.selectedRow < len(v.fieldValues) {
			// Check if the selected field contains a URL
			fieldValue := v.fieldValues[v.selectedRow]
			if utils.IsURL(fieldValue) {
				urlText = utils.ExtractURL(fieldValue)
			}
		} else {
			// Fallback to current line (old behavior)
			currentLine := v.getCurrentLine()
			if utils.IsURL(currentLine) {
				urlText = utils.ExtractURL(currentLine)
			}
		}

		if urlText != "" {
			if err := utils.OpenURLWithTimeout(urlText); err == nil {
				v.statusLine.SetTemporaryMessageWithType("🌐 Opened URL in browser", components.MessageSuccess, 1*time.Second)
			} else {
				v.statusLine.SetTemporaryMessageWithType(fmt.Sprintf("✗ Failed to open URL: %v", err), components.MessageError, 1*time.Second)
			}
			// No blink for URL opening, just message clear timer
			return v.startMessageClearTimer(1 * time.Second)
		}
	}
	return nil
}

// getCurrentLine returns the current line in the viewport
func (v *RunDetailsView) getCurrentLine() string {
	// Get all lines
	lines := v.renderContentWithCursor()
	currentLineIdx := v.viewport.YOffset

	if currentLineIdx >= 0 && currentLineIdx < len(lines) {
		return lines[currentLineIdx]
	}
	return ""
}

// startMessageClearTimer starts a timer to clear temporary messages
func (v *RunDetailsView) startMessageClearTimer(duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return messageClearMsg{}
	})
}
