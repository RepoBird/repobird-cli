package components

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestNewScrollableList(t *testing.T) {
	t.Run("Default configuration", func(t *testing.T) {
		list := NewScrollableList()
		
		assert.NotNil(t, list)
		assert.Equal(t, 1, list.columns)
		assert.False(t, list.keyNav)
		assert.True(t, list.valueNav)
		assert.Equal(t, 0, list.selected)
		assert.Equal(t, 0, list.focusedCol)
		assert.Empty(t, list.items)
	})

	t.Run("With options", func(t *testing.T) {
		list := NewScrollableList(
			WithColumns(3),
			WithKeyNavigation(true),
			WithValueNavigation(false),
			WithDimensions(100, 50),
			WithColumnWidths([]int{30, 40, 30}),
		)
		
		assert.Equal(t, 3, list.columns)
		assert.True(t, list.keyNav)
		assert.False(t, list.valueNav)
		assert.Equal(t, 100, list.width)
		assert.Equal(t, 50, list.height)
		assert.Equal(t, []int{30, 40, 30}, list.columnWidths)
	})
}

func TestScrollableListSetItems(t *testing.T) {
	list := NewScrollableList(WithColumns(3))
	
	items := [][]string{
		{"Item 1", "Value 1", "Status 1"},
		{"Item 2", "Value 2", "Status 2"},
		{"Item 3", "Value 3", "Status 3"},
	}
	
	list.SetItems(items)
	
	assert.Equal(t, items, list.items)
	assert.Equal(t, 0, list.selected)
}

func TestScrollableListGetSelected(t *testing.T) {
	list := NewScrollableList(WithColumns(2))
	
	items := [][]string{
		{"Item 1", "Value 1"},
		{"Item 2", "Value 2"},
		{"Item 3", "Value 3"},
	}
	
	list.SetItems(items)
	
	t.Run("Get first item", func(t *testing.T) {
		selected := list.GetSelected()
		assert.Equal(t, []string{"Item 1", "Value 1"}, selected)
		assert.Equal(t, 0, list.GetSelectedIndex())
	})
	
	t.Run("Set and get selected", func(t *testing.T) {
		list.SetSelected(2)
		selected := list.GetSelected()
		assert.Equal(t, []string{"Item 3", "Value 3"}, selected)
		assert.Equal(t, 2, list.GetSelectedIndex())
	})
	
	t.Run("Invalid selection", func(t *testing.T) {
		list.SetSelected(10) // Out of bounds
		assert.Equal(t, 2, list.GetSelectedIndex()) // Should remain unchanged
	})
}

func TestScrollableListNavigation(t *testing.T) {
	list := NewScrollableList(
		WithColumns(3),
		WithValueNavigation(true),
		WithKeyNavigation(true),
	)
	
	items := [][]string{
		{"Item 1", "Value 1", "Status 1"},
		{"Item 2", "Value 2", "Status 2"},
		{"Item 3", "Value 3", "Status 3"},
		{"Item 4", "Value 4", "Status 4"},
	}
	
	list.SetItems(items)
	
	tests := []struct {
		name           string
		key            string
		expectedRow    int
		expectedCol    int
	}{
		{"Move down", "down", 1, 0},
		{"Move down again", "j", 2, 0},
		{"Move up", "up", 1, 0},
		{"Move up again", "k", 0, 0},
		{"Move right", "right", 0, 1},
		{"Move right again", "l", 0, 2},
		{"Move left", "left", 0, 1},
		{"Move left again", "h", 0, 0},
		{"Tab to next column", "tab", 0, 1},
		{"Tab wraps around", "tab", 0, 2},
		{"Tab wraps to first", "tab", 0, 0},
		{"Shift+tab to previous", "shift+tab", 0, 2},
		{"Home key", "home", 0, 2},
		{"End key", "end", 3, 2},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			if tt.key == "down" || tt.key == "up" || tt.key == "left" || tt.key == "right" {
				msg = tea.KeyMsg{Type: tea.KeyDown}
				switch tt.key {
				case "up":
					msg.Type = tea.KeyUp
				case "left":
					msg.Type = tea.KeyLeft
				case "right":
					msg.Type = tea.KeyRight
				}
			} else if tt.key == "tab" {
				msg = tea.KeyMsg{Type: tea.KeyTab}
			} else if tt.key == "shift+tab" {
				msg = tea.KeyMsg{Type: tea.KeyShiftTab}
			} else if tt.key == "home" {
				msg = tea.KeyMsg{Type: tea.KeyHome}
			} else if tt.key == "end" {
				msg = tea.KeyMsg{Type: tea.KeyEnd}
			}
			
			model, _ := list.Update(msg)
			updatedList := model.(*ScrollableList)
			
			assert.Equal(t, tt.expectedRow, updatedList.selected, "Row selection")
			assert.Equal(t, tt.expectedCol, updatedList.focusedCol, "Column focus")
		})
	}
}

func TestScrollableListValueNavOnly(t *testing.T) {
	list := NewScrollableList(
		WithColumns(3),
		WithValueNavigation(true),
		WithKeyNavigation(false), // Disable key navigation
	)
	
	items := [][]string{
		{"Item 1", "Value 1", "Status 1"},
		{"Item 2", "Value 2", "Status 2"},
	}
	
	list.SetItems(items)
	
	// Try to move right (should not change column)
	msg := tea.KeyMsg{Type: tea.KeyRight}
	model, _ := list.Update(msg)
	updatedList := model.(*ScrollableList)
	
	assert.Equal(t, 0, updatedList.focusedCol, "Column should not change")
	
	// Down should work
	msg = tea.KeyMsg{Type: tea.KeyDown}
	model, _ = list.Update(msg)
	updatedList = model.(*ScrollableList)
	
	assert.Equal(t, 1, updatedList.selected, "Should move down")
}

func TestScrollableListKeyNavOnly(t *testing.T) {
	list := NewScrollableList(
		WithColumns(3),
		WithValueNavigation(false), // Disable value navigation
		WithKeyNavigation(true),
	)
	
	items := [][]string{
		{"Item 1", "Value 1", "Status 1"},
		{"Item 2", "Value 2", "Status 2"},
	}
	
	list.SetItems(items)
	
	// Try to move down (should not change row)
	msg := tea.KeyMsg{Type: tea.KeyDown}
	model, _ := list.Update(msg)
	updatedList := model.(*ScrollableList)
	
	assert.Equal(t, 0, updatedList.selected, "Row should not change")
	
	// Right should work
	msg = tea.KeyMsg{Type: tea.KeyRight}
	model, _ = list.Update(msg)
	updatedList = model.(*ScrollableList)
	
	assert.Equal(t, 1, updatedList.focusedCol, "Should move right")
}

func TestScrollableListWindowResize(t *testing.T) {
	list := NewScrollableList(WithColumns(3))
	
	msg := tea.WindowSizeMsg{
		Width:  120,
		Height: 40,
	}
	
	model, _ := list.Update(msg)
	updatedList := model.(*ScrollableList)
	
	assert.Equal(t, 120, updatedList.width)
	assert.Equal(t, 40, updatedList.height)
	assert.Equal(t, 120, updatedList.viewport.Width)
	assert.Equal(t, 38, updatedList.viewport.Height) // Height - 2 for borders
}

func TestScrollableListRendering(t *testing.T) {
	t.Run("Empty list", func(t *testing.T) {
		list := NewScrollableList()
		list.width = 80
		list.height = 20
		
		view := list.View()
		assert.NotEmpty(t, view)
	})
	
	t.Run("With items", func(t *testing.T) {
		list := NewScrollableList(WithColumns(2))
		list.width = 80
		list.height = 20
		
		items := [][]string{
			{"Item 1", "Value 1"},
			{"Item 2", "Value 2"},
		}
		list.SetItems(items)
		
		content := list.renderContent()
		assert.Contains(t, content, "Item 1")
		assert.Contains(t, content, "Value 1")
		assert.Contains(t, content, "Item 2")
		assert.Contains(t, content, "Value 2")
	})
	
	t.Run("No items message", func(t *testing.T) {
		list := NewScrollableList()
		content := list.renderContent()
		assert.Contains(t, content, "No items to display")
	})
}

func TestScrollableListScrolling(t *testing.T) {
	list := NewScrollableList(WithDimensions(80, 5)) // Small height to test scrolling
	
	// Add many items to force scrolling
	var items [][]string
	for i := 0; i < 20; i++ {
		items = append(items, []string{
			"Item " + string(rune('A'+i)),
		})
	}
	
	list.SetItems(items)
	
	// Move down multiple times
	for i := 0; i < 10; i++ {
		msg := tea.KeyMsg{Type: tea.KeyDown}
		model, _ := list.Update(msg)
		list = model.(*ScrollableList)
	}
	
	assert.Equal(t, 10, list.selected)
	
	// Page down
	msg := tea.KeyMsg{Type: tea.KeyPgDown}
	model, _ := list.Update(msg)
	list = model.(*ScrollableList)
	
	// Page up
	msg = tea.KeyMsg{Type: tea.KeyPgUp}
	model, _ = list.Update(msg)
	list = model.(*ScrollableList)
}

func TestScrollableListFocusManagement(t *testing.T) {
	list := NewScrollableList()
	
	assert.True(t, list.Focused())
	
	list.Focus()
	list.Blur()
	
	// These are no-ops for now but should not panic
	assert.True(t, list.Focused())
}

func TestScrollableListInit(t *testing.T) {
	list := NewScrollableList()
	cmd := list.Init()
	assert.Nil(t, cmd)
}

func TestScrollableListTruncation(t *testing.T) {
	list := NewScrollableList(
		WithColumns(2),
		WithColumnWidths([]int{10, 10}),
	)
	
	items := [][]string{
		{"This is a very long item name", "Another very long value"},
		{"Short", "Val"},
	}
	
	list.SetItems(items)
	
	row := list.renderRow(items[0], false)
	assert.Contains(t, row, "...")  // Long text should be truncated
	
	row = list.renderRow(items[1], false)
	assert.Contains(t, row, "Short") // Short text should not be truncated
}

func TestScrollableListSelection(t *testing.T) {
	list := NewScrollableList(WithColumns(2))
	list.selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("240"))
	
	items := [][]string{
		{"Item 1", "Value 1"},
		{"Item 2", "Value 2"},
	}
	
	list.SetItems(items)
	
	// Render selected row
	selectedRow := list.renderRow(items[0], true)
	normalRow := list.renderRow(items[1], false)
	
	// Selected row should have different styling
	assert.NotEqual(t, selectedRow, normalRow)
}

func TestScrollableListBoundaryConditions(t *testing.T) {
	list := NewScrollableList()
	
	items := [][]string{
		{"Item 1"},
		{"Item 2"},
		{"Item 3"},
	}
	
	list.SetItems(items)
	
	t.Run("Cannot move up from first item", func(t *testing.T) {
		list.selected = 0
		msg := tea.KeyMsg{Type: tea.KeyUp}
		model, _ := list.Update(msg)
		updatedList := model.(*ScrollableList)
		assert.Equal(t, 0, updatedList.selected)
	})
	
	t.Run("Cannot move down from last item", func(t *testing.T) {
		list.selected = 2
		msg := tea.KeyMsg{Type: tea.KeyDown}
		model, _ := list.Update(msg)
		updatedList := model.(*ScrollableList)
		assert.Equal(t, 2, updatedList.selected)
	})
	
	t.Run("Cannot move left from first column", func(t *testing.T) {
		list.focusedCol = 0
		msg := tea.KeyMsg{Type: tea.KeyLeft}
		model, _ := list.Update(msg)
		updatedList := model.(*ScrollableList)
		assert.Equal(t, 0, updatedList.focusedCol)
	})
}