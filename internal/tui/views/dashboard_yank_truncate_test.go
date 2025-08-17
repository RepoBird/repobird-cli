// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package views

import (
	"strings"
	"testing"

	"github.com/repobird/repobird-cli/internal/tui/cache"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
)

func TestDashboardViewYankTruncation(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil, cache.NewSimpleCache())
	view.width = 100
	view.height = 30

	// Add a repository with a very long name
	longName := "very-long-repository-name-that-exceeds-thirty-characters-for-testing-truncation"
	view.repositories = []models.Repository{
		{Name: longName},
	}
	view.selectedRepoIdx = 0
	view.selectedRepo = &view.repositories[0]

	// Simulate pressing 'y' key
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	// Update the view with the key message
	_, _ = view.Update(msg)

	// Check that the message was set
	if view.copiedMessage == "" {
		t.Error("Expected copiedMessage to be set")
	}

	// If the clipboard succeeded, check truncation
	if view.copiedMessage != "✗ Failed to copy" {
		// Check that the message was truncated to 30 chars + "..." for the text part
		// The full message is: 📋 Copied "text..."
		if !strings.Contains(view.copiedMessage, "...") {
			t.Errorf("Expected copiedMessage to contain '...' for truncation, got: %s", view.copiedMessage)
		}
	} else {
		// If clipboard failed, that's OK in test environment
		t.Logf("Clipboard operation failed (expected in CI): %s", view.copiedMessage)
	}

	// Extract the quoted part to check length
	start := strings.Index(view.copiedMessage, "\"")
	end := strings.LastIndex(view.copiedMessage, "\"")
	if start != -1 && end != -1 && start < end {
		quotedText := view.copiedMessage[start+1 : end]
		if len(quotedText) > 30 {
			t.Errorf("Expected quoted text to be at most 30 chars, got %d: %s", len(quotedText), quotedText)
		}
	}
}

func TestDashboardViewYankShortText(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil, cache.NewSimpleCache())
	view.width = 100
	view.height = 30

	// Add a repository with a short name
	shortName := "my-repo"
	view.repositories = []models.Repository{
		{Name: shortName},
	}
	view.selectedRepoIdx = 0
	view.selectedRepo = &view.repositories[0]

	// Simulate pressing 'y' key
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	// Update the view with the key message
	_, _ = view.Update(msg)

	// Check that the message was NOT truncated for short text
	expectedSuccessMsg := "📋 Copied \"my-repo\""
	expectedFailMsg := "✗ Failed to copy"
	if view.copiedMessage != expectedSuccessMsg && view.copiedMessage != expectedFailMsg {
		t.Errorf("Expected copiedMessage to be '%s' or '%s', got '%s'", expectedSuccessMsg, expectedFailMsg, view.copiedMessage)
	}

	// Short text should not contain ellipsis
	if strings.Contains(view.copiedMessage, "...") && view.copiedMessage != expectedFailMsg {
		t.Error("Expected short text not to be truncated")
	}
}
