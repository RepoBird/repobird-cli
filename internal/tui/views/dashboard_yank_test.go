package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
)

func TestDashboardViewYankFeedback(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil) // Pass nil client for testing
	view.width = 100
	view.height = 30

	// Add some test data
	view.repositories = []models.Repository{
		{Name: "test-repo-1"},
		{Name: "test-repo-2"},
	}
	view.selectedRepoIdx = 0
	view.selectedRepo = &view.repositories[0]

	// Simulate pressing 'y' key
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	// Update the view with the key message
	_, cmd := view.Update(msg)

	// Check that clipboard message was set
	if view.copiedMessage == "" {
		t.Error("Expected copiedMessage to be set after pressing 'y'")
	}

	// Check that it contains the expected content
	// Note: In CI/test environments, clipboard may fail, so we accept both success and failure messages
	// The success message should now show the actual text that was copied
	expectedSuccessMsg := "📋 Copied \"test-repo-1\""
	expectedFailMsg := "✗ Failed to copy"
	if view.copiedMessage != expectedSuccessMsg && view.copiedMessage != expectedFailMsg {
		t.Errorf("Expected copiedMessage to be '%s' or '%s', got '%s'", expectedSuccessMsg, expectedFailMsg, view.copiedMessage)
	}

	// Check that copiedMessageTime was set
	if view.copiedMessageTime.IsZero() {
		t.Error("Expected copiedMessageTime to be set")
	}

	// Check that yankBlink was set to true
	if !view.yankBlink {
		t.Error("Expected yankBlink to be true")
	}

	// Check that a command was returned (for animation)
	if cmd == nil {
		t.Error("Expected a command to be returned for animation")
	}

	// Test that the status line shows the copied message
	statusLine := view.renderStatusLine("Test Layout")
	if statusLine == "" {
		t.Error("Expected status line to be rendered")
	}

	// Simulate time passing to test message expiration
	view.copiedMessageTime = time.Now().Add(-4 * time.Second)
	statusLine2 := view.renderStatusLine("Test Layout")

	// The status lines should be different (clipboard message should be gone)
	if statusLine == statusLine2 {
		t.Error("Expected status line to be different after clipboard message expires")
	}
}

func TestDashboardViewYankInRunsColumn(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil) // Pass nil client for testing
	view.width = 100
	view.height = 30
	view.focusedColumn = 1 // Focus on runs column

	// Add test run data
	view.filteredRuns = []*models.RunResponse{
		{
			ID:    "run-123",
			Title: "Test Run",
		},
	}
	view.selectedRunIdx = 0
	view.selectedRunData = view.filteredRuns[0]

	// Simulate pressing 'y' key
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	// Update the view with the key message
	_, cmd := view.Update(msg)

	// Check that clipboard message was set
	// Note: In CI/test environments, clipboard may fail, so we accept both success and failure messages
	// The success message should show the actual text (truncated if needed)
	expectedSuccessMsg := "📋 Copied \"run-123 - Test Run\""
	expectedFailMsg := "✗ Failed to copy"
	if view.copiedMessage != expectedSuccessMsg && view.copiedMessage != expectedFailMsg {
		t.Errorf("Expected copiedMessage to be '%s' or '%s', got '%s'", expectedSuccessMsg, expectedFailMsg, view.copiedMessage)
	}

	// Check that a command was returned
	if cmd == nil {
		t.Error("Expected a command to be returned for animation")
	}
}

func TestDashboardViewYankInDetailsColumn(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil) // Pass nil client for testing
	view.width = 100
	view.height = 30
	view.focusedColumn = 2 // Focus on details column

	// Add test detail lines (both display and original versions)
	view.detailLines = []string{
		"Detail line 1",
		"Detail line 2",
		"Detail line 3",
	}
	view.detailLinesOriginal = []string{
		"Detail line 1",
		"Detail line 2",
		"Detail line 3",
	}
	view.selectedDetailLine = 1

	// Simulate pressing 'y' key
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'y'},
	}

	// Update the view with the key message
	_, cmd := view.Update(msg)

	// Check that clipboard message was set
	// Note: In CI/test environments, clipboard may fail, so we accept both success and failure messages
	// The success message should show the actual text
	expectedSuccessMsg := "📋 Copied \"Detail line 2\""
	expectedFailMsg := "✗ Failed to copy"
	if view.copiedMessage != expectedSuccessMsg && view.copiedMessage != expectedFailMsg {
		t.Errorf("Expected copiedMessage to be '%s' or '%s', got '%s'", expectedSuccessMsg, expectedFailMsg, view.copiedMessage)
	}

	// Check that a command was returned
	if cmd == nil {
		t.Error("Expected a command to be returned for animation")
	}
}

func TestDashboardViewBlinkAnimation(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil) // Pass nil client for testing
	view.width = 100
	view.height = 30

	// Set clipboard message with recent time
	view.copiedMessage = "📋 Copied test"
	view.copiedMessageTime = time.Now()
	view.yankBlink = true

	// Test that blinking is applied in the first second
	statusLine := view.renderStatusLine("Test")
	if statusLine == "" {
		t.Error("Expected status line to be rendered")
	}

	// Simulate yankBlinkMsg to toggle blink
	_, _ = view.Update(yankBlinkMsg{})

	// Check that yankBlink toggled
	if view.yankBlink {
		t.Error("Expected yankBlink to toggle to false")
	}

	// Another toggle
	_, _ = view.Update(yankBlinkMsg{})

	// Check that yankBlink toggled back
	if !view.yankBlink {
		t.Error("Expected yankBlink to toggle back to true")
	}
}

func TestDashboardViewClearStatus(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil) // Pass nil client for testing
	view.width = 100
	view.height = 30

	// Set clipboard message
	view.copiedMessage = "📋 Copied \"test\""
	view.copiedMessageTime = time.Now()
	view.yankBlink = true

	// Simulate clearStatusMsg
	_, _ = view.Update(clearStatusMsg{})

	// Check that message was cleared
	if view.copiedMessage != "" {
		t.Errorf("Expected copiedMessage to be cleared, got '%s'", view.copiedMessage)
	}

	// Check that yankBlink was reset
	if view.yankBlink {
		t.Error("Expected yankBlink to be false after clearing")
	}
}
