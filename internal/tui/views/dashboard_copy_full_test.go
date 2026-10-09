// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"strings"
	"testing"

	"time"

	"github.com/repobird/repobird-cli/internal/tui/cache"

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

func TestDashboardViewDisplaysServerSelectedPolicy(t *testing.T) {
	tests := []struct {
		name            string
		model           string
		reasoningEffort string
		wantModel       bool
		wantReasoning   bool
	}{
		{
			name:            "complete policy",
			model:           "openai/gpt-5.6",
			reasoningEffort: "max",
			wantModel:       true,
			wantReasoning:   true,
		},
		{
			name:      "model only with unfamiliar value",
			model:     "future-provider/future-model:v9-beta",
			wantModel: true,
		},
		{
			name:            "reasoning effort only with unfamiliar value",
			reasoningEffort: "future-ultra/adaptive:v2",
			wantReasoning:   true,
		},
		{
			name: "legacy response without policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewDashboardView(nil, cache.NewSimpleCache())
			view.width = 90
			view.height = 20
			view.focusedColumn = 2
			view.updateViewportSizes()
			view.selectedRunData = &models.RunResponse{
				ID:              "policy-run-123",
				Status:          models.StatusDone,
				Repository:      "acme/webapp",
				CreatedAt:       time.Date(2026, time.August, 15, 7, 0, 0, 0, time.UTC),
				UpdatedAt:       time.Date(2026, time.August, 15, 7, 5, 0, 0, time.UTC),
				Model:           tt.model,
				ReasoningEffort: tt.reasoningEffort,
			}

			view.updateDetailLines()

			assertDashboardPolicyLine(t, view, "Model", tt.model, tt.wantModel)
			assertDashboardPolicyLine(t, view, "Reasoning effort", tt.reasoningEffort, tt.wantReasoning)
			if !containsLine(view.detailLinesOriginal, "ID: policy-run-123") {
				t.Fatalf("existing dashboard details were lost: %q", view.detailLinesOriginal)
			}
			if content := view.detailsViewport.View(); !strings.Contains(content, "ID: policy-run-123") {
				t.Fatalf("details viewport became unusable, content: %q", content)
			}
		})
	}
}

func TestDashboardViewPolicyCopyUsesFullOpaqueValue(t *testing.T) {
	view := NewDashboardView(nil, cache.NewSimpleCache())
	view.width = 60
	view.height = 20
	view.focusedColumn = 2
	view.updateViewportSizes()

	longModel := "future-provider/very-long-model-identifier:v99-experimental"
	longReasoningEffort := "future-ultra-adaptive-reasoning-policy:v42"
	view.selectedRunData = &models.RunResponse{
		ID:              "policy-copy-123",
		Status:          models.StatusDone,
		Repository:      "acme/webapp",
		CreatedAt:       time.Date(2026, time.August, 15, 7, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2026, time.August, 15, 7, 5, 0, 0, time.UTC),
		Model:           longModel,
		ReasoningEffort: longReasoningEffort,
	}

	view.updateDetailLines()

	for _, policy := range []struct {
		label string
		value string
	}{
		{label: "Model", value: longModel},
		{label: "Reasoning effort", value: longReasoningEffort},
	} {
		expected := fmt.Sprintf("%s: %s", policy.label, policy.value)
		lineIndex := lineIndex(view.detailLinesOriginal, expected)
		if lineIndex < 0 {
			t.Fatalf("missing %s copy metadata in %q", policy.label, view.detailLinesOriginal)
		}
		if display := view.detailLines[lineIndex]; display == expected || !strings.HasSuffix(display, "...") {
			t.Fatalf("expected truncated %s display, got %q", policy.label, display)
		}

		view.selectedDetailLine = lineIndex
		if copied := view.getDetailTextToCopy(); copied != expected {
			t.Fatalf("expected full %s value for copy, got %q", policy.label, copied)
		}
	}
}

func assertDashboardPolicyLine(t *testing.T, view *DashboardView, label, value string, want bool) {
	t.Helper()

	prefix := label + ":"
	if !want {
		for _, line := range view.detailLinesOriginal {
			if strings.HasPrefix(line, prefix) {
				t.Fatalf("empty policy value produced an invented line %q", line)
			}
		}
		return
	}

	expected := fmt.Sprintf("%s: %s", label, value)
	if !containsLine(view.detailLinesOriginal, expected) {
		t.Fatalf("missing policy line %q in %q", expected, view.detailLinesOriginal)
	}
}

func containsLine(lines []string, expected string) bool {
	return lineIndex(lines, expected) >= 0
}

func lineIndex(lines []string, expected string) int {
	for i, line := range lines {
		if line == expected {
			return i
		}
	}
	return -1
}
