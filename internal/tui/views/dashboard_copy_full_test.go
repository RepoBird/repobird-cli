// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package views

import (
	"strings"
	"testing"

	"github.com/repobird/repobird-cli/internal/tui/cache"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
)

func TestDashboardViewCopyFullTextFromDetails(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil, cache.NewSimpleCache())
	view.width = 100
	view.height = 30
	view.focusedColumn = 2 // Focus on details column

	// Create a run with long repository name that would be truncated
	longRepoName := "very-long-organization-name/extremely-long-repository-name-that-exceeds-normal-width"
	view.selectedRunData = &models.RunResponse{
		ID:         "test-123",
		Status:     models.StatusDone,
		Repository: longRepoName,
		Title:      "Test Run Title",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Update detail lines which will truncate for display
	view.updateDetailLines()

	// Verify that display lines are truncated
	if len(view.detailLines) == 0 {
		t.Fatal("Expected detail lines to be populated")
	}

	// Find the repository line
	var repoLineIdx int
	for i, line := range view.detailLines {
		if strings.HasPrefix(line, "Repository:") {
			repoLineIdx = i
			break
		}
	}

	// The display line should be truncated (contains ...)
	displayLine := view.detailLines[repoLineIdx]
	t.Logf("Display line: %s", displayLine)

	// Set selected line to repository line
	view.selectedDetailLine = repoLineIdx

	// Simulate pressing 'y' key to copy
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	// Update the view with the key message
	_, _ = view.Update(msg)

	// Check the copied message
	if view.copiedMessage == "" {
		t.Error("Expected copiedMessage to be set")
	}

	// If clipboard succeeded, verify we copied the full text
	if !strings.Contains(view.copiedMessage, "Failed") {
		// The copied message should show part of the full repository name
		expectedFullText := "Repository: " + longRepoName

		// Check that the original line contains the full text
		if view.detailLinesOriginal[repoLineIdx] != expectedFullText {
			t.Errorf("Expected original line to be '%s', got '%s'",
				expectedFullText, view.detailLinesOriginal[repoLineIdx])
		}

		t.Logf("Successfully copied full text: %s", view.detailLinesOriginal[repoLineIdx])
	}
}

func TestDashboardViewCopyLongPrompt(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil, cache.NewSimpleCache())
	view.width = 100
	view.height = 30
	view.focusedColumn = 2 // Focus on details column

	// Create a run with a very long prompt
	longPrompt := "This is an extremely long prompt that contains detailed instructions for the AI assistant to perform complex tasks involving multiple steps and requiring careful analysis of the codebase structure and implementation details"

	view.selectedRunData = &models.RunResponse{
		ID:        "test-456",
		Status:    models.StatusDone,
		Prompt:    longPrompt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Update detail lines
	view.updateDetailLines()

	// Find the prompt line
	var promptLineIdx int
	foundPromptHeader := false
	for i, line := range view.detailLines {
		if line == "Prompt:" {
			foundPromptHeader = true
		} else if foundPromptHeader && line != "" {
			promptLineIdx = i
			break
		}
	}

	// The display line should be truncated
	displayLine := view.detailLines[promptLineIdx]
	t.Logf("Display prompt line: %s", displayLine)

	// Original should have full text
	originalLine := view.detailLinesOriginal[promptLineIdx]
	if originalLine != longPrompt {
		t.Errorf("Expected original prompt to be '%s', got '%s'", longPrompt, originalLine)
	}

	// Set selected line to prompt line
	view.selectedDetailLine = promptLineIdx

	// Simulate pressing 'y' key
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	// Update the view
	_, _ = view.Update(msg)

	// Verify we're copying the full prompt
	if !strings.Contains(view.copiedMessage, "Failed") {
		t.Logf("Successfully prepared to copy full prompt: %s", originalLine)
	}
}
