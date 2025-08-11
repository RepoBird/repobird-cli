package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBulkFileSelector_NoInfiniteLoop(t *testing.T) {
	// Create a new bulk file selector
	selector := NewBulkFileSelector(80, 24)
	assert.NotNil(t, selector)

	// Test that Activate returns a command, not an error
	cmd := selector.Activate()
	assert.NotNil(t, cmd, "Activate should return a command")

	// Execute the command with timeout to ensure no infinite loop
	done := make(chan bool)
	go func() {
		msg := cmd()
		assert.NotNil(t, msg, "Command should return a message")
		done <- true
	}()

	select {
	case <-done:
		// Good, command completed
	case <-time.After(2 * time.Second):
		t.Fatal("Command execution timed out - possible infinite loop")
	}
}

func TestBulkFileSelector_Activation(t *testing.T) {
	selector := NewBulkFileSelector(80, 24)
	
	// Initially should not be active
	assert.False(t, selector.IsActive())
	
	// Activate
	cmd := selector.Activate()
	require.NotNil(t, cmd)
	
	// Should now be active
	assert.True(t, selector.IsActive())
	assert.True(t, selector.loading)
}

func TestBulkFileSelector_KeyHandling(t *testing.T) {
	selector := NewBulkFileSelector(80, 24)
	selector.SetActive(true)
	
	// Test ESC key
	escMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")}
	updatedSelector, cmd := selector.Update(escMsg)
	assert.NotNil(t, updatedSelector)
	assert.NotNil(t, cmd) // Should return a cancel message
	assert.False(t, updatedSelector.IsActive())
}

func TestBulkFileSelector_View(t *testing.T) {
	selector := NewBulkFileSelector(80, 24)
	statusLine := NewStatusLine()
	
	// Test inactive view
	view := selector.View(statusLine)
	assert.Empty(t, view, "Inactive selector should return empty view")
	
	// Test active view
	selector.SetActive(true)
	view = selector.View(statusLine)
	assert.NotEmpty(t, view, "Active selector should return non-empty view")
	assert.Contains(t, view, "Bulk Config Files")
}

func TestBulkFileSelector_NoTickLoop(t *testing.T) {
	selector := NewBulkFileSelector(80, 24)
	selector.SetActive(true)
	
	// Test that TickMsg doesn't create infinite loop
	tickMsg := TickMsg(time.Now())
	updatedSelector, cmd := selector.Update(tickMsg)
	assert.NotNil(t, updatedSelector)
	
	// Should return tick command that continues the blink
	assert.NotNil(t, cmd)
	
	// But we should be able to stop it with ESC
	escMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{27}} // ESC key
	updatedSelector, _ = selector.Update(escMsg)
	assert.False(t, updatedSelector.IsActive())
}