package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestDashboardView_HKeyColumnNavigation(t *testing.T) {
	// Create a new dashboard view
	view := NewDashboardView(nil)
	view.width = 100
	view.height = 30

	t.Run("H and h keys enabled across all columns", func(t *testing.T) {
		// Both 'h' and 'H' should be enabled across all columns since they're handled by HandleKey
		for column := 0; column <= 2; column++ {
			view.focusedColumn = column
			
			assert.False(t, view.IsKeyDisabled("h"), "'h' key should be enabled on column %d", column)
			assert.False(t, view.IsKeyDisabled("H"), "'H' key should be enabled on column %d", column)
		}
	})

	t.Run("H key moves left like h key", func(t *testing.T) {
		// Test 'H' key moving left from column 2 to 1
		view.focusedColumn = 2 // Third column (details)
		
		handled, model, cmd := view.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
		
		assert.True(t, handled, "'H' key should be handled by dashboard")
		assert.Equal(t, view, model, "Model should be the same view")
		assert.Nil(t, cmd, "No command should be returned")
		assert.Equal(t, 1, view.focusedColumn, "Should move from column 2 to column 1")
	})

	t.Run("H key does nothing on first column", func(t *testing.T) {
		// Test 'H' key on first column (nowhere to go left)
		view.focusedColumn = 0 // First column (repositories)
		
		handled, model, cmd := view.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("H")})
		
		assert.True(t, handled, "'H' key should be handled by dashboard")
		assert.Equal(t, view, model, "Model should be the same view")
		assert.Nil(t, cmd, "No command should be returned")
		assert.Equal(t, 0, view.focusedColumn, "Should stay on column 0")
	})

	t.Run("h and H behave identically", func(t *testing.T) {
		// Test that 'h' and 'H' have identical behavior
		for _, key := range []string{"h", "H"} {
			// Test from column 1 to 0
			view.focusedColumn = 1
			
			handled, model, cmd := view.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			
			assert.True(t, handled, "'%s' key should be handled by dashboard", key)
			assert.Equal(t, view, model, "Model should be the same view for '%s'", key)
			assert.Nil(t, cmd, "No command should be returned for '%s'", key)
			assert.Equal(t, 0, view.focusedColumn, "Should move to column 0 for '%s'", key)
			
			// Reset for next test
			view.focusedColumn = 1
		}
	})

	t.Run("L key moves right like l key", func(t *testing.T) {
		// Test 'L' key moving right from column 0 to 1
		view.focusedColumn = 0 // First column (repositories)
		
		// Simulate 'L' key press through the navigation handler
		// Since this is handled in dash_navigation.go, we need to test via Update method
		model, cmd := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
		
		assert.Equal(t, view, model, "Model should be the same view")
		assert.Nil(t, cmd, "No command should be returned for column navigation")
		assert.Equal(t, 1, view.focusedColumn, "Should move from column 0 to column 1")
	})

	t.Run("L key does nothing on last column", func(t *testing.T) {
		// Test 'L' key on last column (nowhere to go right)
		view.focusedColumn = 2 // Third column (details)
		
		model, cmd := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
		
		assert.Equal(t, view, model, "Model should be the same view")
		assert.Nil(t, cmd, "No command should be returned")
		assert.Equal(t, 2, view.focusedColumn, "Should stay on column 2")
	})

	t.Run("l and L behave identically for right movement", func(t *testing.T) {
		// Test that 'l' and 'L' have identical behavior for right movement
		for _, key := range []string{"l", "L"} {
			// Test from column 0 to 1
			view.focusedColumn = 0
			
			model, cmd := view.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
			
			assert.Equal(t, view, model, "Model should be the same view for '%s'", key)
			assert.Nil(t, cmd, "No command should be returned for '%s'", key)
			assert.Equal(t, 1, view.focusedColumn, "Should move to column 1 for '%s'", key)
			
			// Reset for next test
			view.focusedColumn = 0
		}
	})

	t.Run("Static disabled keys remain disabled", func(t *testing.T) {
		// Test that statically disabled keys remain disabled
		for column := 0; column <= 2; column++ {
			view.focusedColumn = column
			assert.True(t, view.IsKeyDisabled("esc"), 
				"'esc' key should be disabled on column %d", column)
		}
	})
}