package views

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
)

func TestDashboardViewStatusLineShowsActualCopiedText(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil)
	view.width = 100
	view.height = 30
	view.focusedColumn = 2 // Focus on details column

	// Create a run with a very long repository name
	longRepoName := "github.com/very-long-organization-name/extremely-long-repository-name-that-definitely-exceeds-thirty-characters"
	view.selectedRunData = &models.RunResponse{
		ID:         "test-123",
		Status:     models.StatusDone,
		Repository: longRepoName,
		Title:      "Test Run",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Update detail lines (this will create truncated display versions)
	view.updateDetailLines()

	// Find and select the repository line
	var repoLineIdx int
	for i, line := range view.detailLines {
		if strings.HasPrefix(line, "Repository:") {
			repoLineIdx = i
			break
		}
	}
	view.selectedDetailLine = repoLineIdx

	// The display line should be truncated
	displayLine := view.detailLines[repoLineIdx]
	if !strings.Contains(displayLine, "...") {
		t.Logf("Display line might not be truncated (OK if short): %s", displayLine)
	}

	// The original line should have the full text
	originalLine := view.detailLinesOriginal[repoLineIdx]
	expectedOriginal := "Repository: " + longRepoName
	if originalLine != expectedOriginal {
		t.Errorf("Expected original line to be '%s', got '%s'", expectedOriginal, originalLine)
	}

	// Simulate pressing 'y' key to copy
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	// Update the view
	_, _ = view.Update(msg)

	// Check the copied message
	if view.copiedMessage == "" {
		t.Error("Expected copiedMessage to be set")
	}

	// If clipboard succeeded, verify the status line message
	if !strings.Contains(view.copiedMessage, "Failed") {
		// The status line should show the actual copied text (truncated for display)
		// It should show: 📋 Copied "Repository: github.com/very-l..."
		// NOT: 📋 Copied "Repository: github.com/ve..."

		// Extract the quoted part
		start := strings.Index(view.copiedMessage, "\"")
		end := strings.LastIndex(view.copiedMessage, "\"")
		if start == -1 || end == -1 || start >= end {
			t.Errorf("Expected copiedMessage to contain quoted text, got: %s", view.copiedMessage)
			return
		}

		quotedText := view.copiedMessage[start+1 : end]

		// The quoted text should be from the ORIGINAL (what's on clipboard), not the display version
		// It should start with the full repository text (up to truncation point)
		expectedPrefix := "Repository: " + longRepoName[:20] // At least first 20 chars of the actual repo name
		if !strings.HasPrefix(quotedText, expectedPrefix) {
			t.Errorf("Expected status line to show actual copied text starting with '%s', but got: %s",
				expectedPrefix, quotedText)
		}

		// If truncated in status line, should have ellipsis
		if len(originalLine) > 30 && !strings.HasSuffix(quotedText, "...") {
			t.Error("Expected long copied text to be truncated with ... in status line")
		}

		t.Logf("Status line correctly shows: %s", view.copiedMessage)
		t.Logf("Actual clipboard content: %s", originalLine)
	}
}

func TestDashboardViewStatusLineNoEllipsisForShortText(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil)
	view.width = 100
	view.height = 30
	view.focusedColumn = 2

	// Create a run with a short repository name
	shortRepoName := "myorg/myrepo"
	view.selectedRunData = &models.RunResponse{
		ID:         "test-456",
		Status:     models.StatusDone,
		Repository: shortRepoName,
		Title:      "Test",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// Update detail lines
	view.updateDetailLines()

	// Find and select the repository line
	var repoLineIdx int
	for i, line := range view.detailLines {
		if strings.HasPrefix(line, "Repository:") {
			repoLineIdx = i
			break
		}
	}
	view.selectedDetailLine = repoLineIdx

	// Simulate pressing 'y'
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	_, _ = view.Update(msg)

	// Check the copied message
	if !strings.Contains(view.copiedMessage, "Failed") {
		// For short text, status line should show the full text without ellipsis
		expectedMsg := "📋 Copied \"Repository: myorg/myrepo\""
		if view.copiedMessage != expectedMsg {
			t.Errorf("Expected status line to show '%s', got '%s'", expectedMsg, view.copiedMessage)
		}

		// Should NOT have ellipsis for short text
		if strings.Contains(view.copiedMessage, "...") {
			t.Error("Short text should not be truncated with ... in status line")
		}
	}
}
