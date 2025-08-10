package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

func TestRunDetailsView_Navigation(t *testing.T) {
	// Create a mock run with multiple fields
	run := models.RunResponse{
		ID:         "test-123",
		Title:      "Test Run Title",
		Repository: "test/repo",
		Source:     "main",
		Target:     "feature",
		RunType:    "run",
		Status:     models.StatusDone,
		Prompt:     "This is a test prompt",
		Context:    "This is test context",
		Plan:       "This is a test plan",
		CreatedAt:  time.Now().Add(-1 * time.Hour),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		keys:           components.DefaultKeyMap,
		navigationMode: true,
		width:          80,
		height:         24,
	}

	// Initialize content to populate field values
	view.updateContent()

	t.Run("NavigateDown", func(t *testing.T) {
		initialRow := view.selectedRow

		// Simulate pressing 'j' to navigate down
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		view.handleRowNavigation(msg)

		assert.Equal(t, initialRow+1, view.selectedRow, "Should move to next row")
		assert.True(t, view.selectedRow < len(view.fieldValues), "Should stay within bounds")
	})

	t.Run("NavigateUp", func(t *testing.T) {
		// Move to second row first
		view.selectedRow = 1

		// Simulate pressing 'k' to navigate up
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		view.handleRowNavigation(msg)

		assert.Equal(t, 0, view.selectedRow, "Should move to previous row")
	})

	t.Run("NavigateToFirst", func(t *testing.T) {
		// Move to middle
		view.selectedRow = len(view.fieldValues) / 2

		// Simulate pressing 'g' to go to first
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		view.handleRowNavigation(msg)

		assert.Equal(t, 0, view.selectedRow, "Should move to first row")
	})

	t.Run("NavigateToLast", func(t *testing.T) {
		// Start at first
		view.selectedRow = 0

		// Simulate pressing 'G' to go to last
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		view.handleRowNavigation(msg)

		assert.Equal(t, len(view.fieldValues)-1, view.selectedRow, "Should move to last row")
	})

	t.Run("BoundaryChecks", func(t *testing.T) {
		// Try to navigate up from first row
		view.selectedRow = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		view.handleRowNavigation(msg)
		assert.Equal(t, 0, view.selectedRow, "Should stay at first row")

		// Try to navigate down from last row
		view.selectedRow = len(view.fieldValues) - 1
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		view.handleRowNavigation(msg)
		assert.Equal(t, len(view.fieldValues)-1, view.selectedRow, "Should stay at last row")
	})
}

func TestRunDetailsView_FieldExtraction(t *testing.T) {
	run := models.RunResponse{
		ID:         "test-456",
		Title:      "Another Test",
		Repository: "example/project",
		Source:     "develop",
		Target:     "release",
		RunType:    "plan",
		Status:     models.StatusDone,
		Prompt:     "Test prompt content",
		Plan:       "Test plan content",
		Context:    "Test context content",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		navigationMode: true,
		width:          80,
		height:         24,
	}

	view.updateContent()

	t.Run("FieldValuesExtracted", func(t *testing.T) {
		assert.Greater(t, len(view.fieldValues), 0, "Should have extracted field values")

		// Check that important fields are present
		hasID := false
		hasRepo := false
		hasPrompt := false
		hasPlan := false

		for _, value := range view.fieldValues {
			if strings.Contains(value, "test-456") {
				hasID = true
			}
			if strings.Contains(value, "example/project") {
				hasRepo = true
			}
			if value == "Test prompt content" {
				hasPrompt = true
			}
			if value == "Test plan content" {
				hasPlan = true
			}
		}

		assert.True(t, hasID, "Should have ID field")
		assert.True(t, hasRepo, "Should have Repository field")
		assert.True(t, hasPrompt, "Should have Prompt field")
		assert.True(t, hasPlan, "Should have Plan field")
	})

	t.Run("FieldIndicesMatch", func(t *testing.T) {
		assert.Equal(t, len(view.fieldValues), len(view.fieldIndices),
			"Should have same number of values and indices")

		// Verify indices are in ascending order
		for i := 1; i < len(view.fieldIndices); i++ {
			assert.GreaterOrEqual(t, view.fieldIndices[i], view.fieldIndices[i-1],
				"Indices should be in ascending order")
		}
	})
}

func TestRunDetailsView_HighlightedContent(t *testing.T) {
	run := models.RunResponse{
		ID:         "highlight-test",
		Title:      "Highlight Test",
		Repository: "test/highlight",
		Status:     models.StatusDone,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	view := &RunDetailsView{
		run:            run,
		navigationMode: true,
		selectedRow:    1, // Select second field
		width:          80,
		height:         24,
	}

	view.updateContent()

	t.Run("CreatesHighlightedContent", func(t *testing.T) {
		lines := []string{
			"Line 1",
			"Line 2",
			"Line 3",
		}

		// Set up field indices to match lines
		view.fieldIndices = []int{0, 1, 2}
		view.selectedRow = 1

		highlighted := view.createHighlightedContent(lines)

		assert.NotEmpty(t, highlighted, "Should create highlighted content")
		// The highlighted content should contain all the lines
		for _, line := range lines {
			assert.Contains(t, highlighted, line, "Should contain all lines")
		}
		// The function should be callable without panicking
		// (lipgloss may not render ANSI codes in test environment)
	})
}
