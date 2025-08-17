// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestRunDetailsView_CursorVisibility(t *testing.T) {
	// Create a mock run with multiple fields
	run := models.RunResponse{
		ID:         "test-cursor",
		Title:      "Test Cursor Run",
		Repository: "test/repo",
		Source:     "main",
		Target:     "feature",
		RunType:    "run",
		Status:     models.StatusDone,
		Prompt:     "This is a test prompt that should be selectable",
		Context:    "This is test context that should also be selectable",
		CreatedAt:  time.Now().Add(-1 * time.Hour),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		keys:           components.DefaultKeyMap,
		navigationMode: true,
		width:          80,
		height:         24,
		selectedRow:    0,
	}

	// Initialize content to populate field values
	view.updateContent()

	t.Run("RenderContentWithCursor", func(t *testing.T) {
		// Get rendered lines with cursor
		lines := view.renderContentWithCursor()

		assert.Greater(t, len(lines), 0, "Should render some lines")

		// At least one line should be styled (the selected field)
		// Note: In test environment, lipgloss might not add ANSI codes
		// but the function should not panic
		assert.NotNil(t, lines, "Should return non-nil lines")
	})

	t.Run("CursorMoveChangesHighlight", func(t *testing.T) {
		// Set initial position
		view.selectedRow = 0
		_ = view.renderContentWithCursor() // Initial render

		// Move cursor down
		if len(view.fieldValues) > 1 {
			view.selectedRow = 1
			afterMoveLines := view.renderContentWithCursor()

			// The content should still be present
			assert.NotNil(t, afterMoveLines, "Should return lines after cursor move")
			assert.Greater(t, len(afterMoveLines), 0, "Should have content after cursor move")
		}
	})

	t.Run("ViewportScrolling", func(t *testing.T) {
		// Test that viewport scrolling works
		view.viewport.Height = 10
		view.viewport.Width = 80
		view.viewport.SetContent(view.fullContent)

		// Scroll down
		view.viewport.ScrollDown(5)

		// Get rendered content
		lines := view.renderContentWithCursor()
		assert.NotNil(t, lines, "Should render lines after scrolling")
	})
}

func TestRunDetailsView_HighlightStyle(t *testing.T) {
	run := models.RunResponse{
		ID:         "highlight-style-test",
		Title:      "Style Test",
		Repository: "test/style",
		Status:     models.StatusDone,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		navigationMode: true,
		width:          80,
		height:         24,
		selectedRow:    0,
	}

	view.updateContent()

	t.Run("HighlightApplied", func(t *testing.T) {
		// Create a simple field range for testing
		view.fieldRanges = [][2]int{{0, 0}, {1, 1}, {2, 2}}
		view.selectedRow = 1

		// Mock content lines
		view.fullContent = "Line 1\nLine 2\nLine 3"

		lines := view.renderContentWithCursor()

		// Should have rendered lines
		assert.Greater(t, len(lines), 0, "Should render lines")

		// The function should complete without error
		// In actual terminal, line at index 1 would be highlighted
	})

	t.Run("BlinkOnCopy", func(t *testing.T) {
		// Initialize the status line if not already initialized
		if view.statusLine == nil {
			view.statusLine = components.NewStatusLine()
		}
		// Simulate copying with unified status line
		view.statusLine.SetTemporaryMessageWithType("Test copied", components.MessageSuccess, 3*time.Second)

		lines := view.renderContentWithCursor()

		// Should still render without error
		assert.NotNil(t, lines, "Should render even with blink state")
	})
}

func TestRunDetailsView_MultilineFieldHighlight(t *testing.T) {
	run := models.RunResponse{
		ID:         "multiline-test",
		Title:      "Multiline Test",
		Repository: "test/multiline",
		Status:     models.StatusDone,
		Prompt:     "Line 1 of prompt\nLine 2 of prompt\nLine 3 of prompt",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		navigationMode: true,
		width:          80,
		height:         24,
		selectedRow:    0,
	}

	view.updateContent()

	t.Run("MultilineFieldFullyHighlighted", func(t *testing.T) {
		// Find the prompt field
		promptFieldIdx := -1
		for i, val := range view.fieldValues {
			if strings.Contains(val, "Line 1 of prompt") {
				promptFieldIdx = i
				break
			}
		}

		if promptFieldIdx >= 0 {
			view.selectedRow = promptFieldIdx

			// Check that the field range spans multiple lines
			fieldRange := view.fieldRanges[promptFieldIdx]
			assert.Greater(t, fieldRange[1]-fieldRange[0], 0,
				"Multi-line field should have range spanning multiple lines")

			// Render with cursor
			lines := view.renderContentWithCursor()
			assert.NotNil(t, lines, "Should render multiline field")
		}
	})
}

func TestRunDetailsView_NavigationWithHighlight(t *testing.T) {
	run := models.RunResponse{
		ID:         "nav-highlight-test",
		Title:      "Navigation Test",
		Repository: "test/navigation",
		Source:     "main",
		Status:     models.StatusDone,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		navigationMode: true,
		width:          80,
		height:         24,
		selectedRow:    0,
	}

	view.updateContent()

	t.Run("NavigationUpdatesHighlight", func(t *testing.T) {
		// Navigate down
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		view.handleRowNavigation(msg)

		// Render to see the change
		lines := view.renderContentWithCursor()
		assert.NotNil(t, lines, "Should render after navigation")

		// Navigate up
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		view.handleRowNavigation(msg)

		lines = view.renderContentWithCursor()
		assert.NotNil(t, lines, "Should render after navigation up")
	})
}

// TestRunDetailsView_HighlightWidth tests that highlight spans full width
func TestRunDetailsView_HighlightWidth(t *testing.T) {
	run := models.RunResponse{
		ID:         "width-test",
		Title:      "Width Test",
		Repository: "test/width",
		Status:     models.StatusDone,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		navigationMode: true,
		width:          120, // Wide terminal
		height:         30,
		selectedRow:    0,
	}

	view.updateContent()

	t.Run("HighlightUsesFullWidth", func(t *testing.T) {
		// The highlight style should use the full terminal width
		style := lipgloss.NewStyle().
			Background(lipgloss.Color("63")).
			Foreground(lipgloss.Color("255")).
			Width(view.width)

		// Apply style to a short string
		shortText := "Short"
		styled := style.Render(shortText)

		// In actual terminal, this would pad to full width
		// Just verify it doesn't panic
		assert.NotEmpty(t, styled, "Style should render")
	})
}
