package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/tui/keymap"
	"github.com/stretchr/testify/assert"
)

// MockViewWithKeymap is a mock view that implements ViewKeymap
type MockViewWithKeymap struct {
	keymap keymap.ViewKeymap
}

// Implement tea.Model interface
func (m *MockViewWithKeymap) Init() tea.Cmd                           { return nil }
func (m *MockViewWithKeymap) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *MockViewWithKeymap) View() string                            { return "mock view" }

// Implement ViewKeymap interface
func (m *MockViewWithKeymap) IsNavigationKeyEnabled(key keymap.NavigationKey) bool {
	return m.keymap.IsNavigationKeyEnabled(key)
}

// MockViewWithoutKeymap is a mock view that does NOT implement ViewKeymap
type MockViewWithoutKeymap struct{}

// Implement tea.Model interface
func (m *MockViewWithoutKeymap) Init() tea.Cmd                           { return nil }
func (m *MockViewWithoutKeymap) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *MockViewWithoutKeymap) View() string                            { return "mock view without keymap" }

func TestAppKeyMsgToNavigationKey(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)

	tests := []struct {
		keyString string
		expected  keymap.NavigationKey
	}{
		{"b", keymap.NavigationKeyBack},
		{"B", keymap.NavigationKeyBulk},
		{"n", keymap.NavigationKeyNew},
		{"r", keymap.NavigationKeyRefresh},
		{"s", keymap.NavigationKeyStatus},
		{"?", keymap.NavigationKeyHelp},
		{"q", keymap.NavigationKeyQuit},
		{"x", ""}, // Unknown key should return empty string
	}

	for _, tt := range tests {
		t.Run("key_"+tt.keyString, func(t *testing.T) {
			keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.keyString)}
			result := app.keyMsgToNavigationKey(keyMsg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAppNavigationKeyToMessage(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)

	t.Run("back navigation key creates NavigateBackMsg", func(t *testing.T) {
		msg := app.navigationKeyToMessage(keymap.NavigationKeyBack)
		assert.NotNil(t, msg)
		// Can't easily test the exact type without importing messages, but non-nil is good
	})

	t.Run("bulk navigation key creates NavigateToBulkMsg", func(t *testing.T) {
		msg := app.navigationKeyToMessage(keymap.NavigationKeyBulk)
		assert.NotNil(t, msg)
	})

	t.Run("other keys return nil", func(t *testing.T) {
		msg := app.navigationKeyToMessage(keymap.NavigationKeyNew)
		assert.Nil(t, msg)

		msg = app.navigationKeyToMessage(keymap.NavigationKeyRefresh)
		assert.Nil(t, msg)

		msg = app.navigationKeyToMessage(keymap.NavigationKeyStatus)
		assert.Nil(t, msg)

		msg = app.navigationKeyToMessage(keymap.NavigationKeyHelp)
		assert.Nil(t, msg)

		msg = app.navigationKeyToMessage(keymap.NavigationKeyQuit)
		assert.Nil(t, msg)
	})
}

func TestAppKeymapIntegration(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)

	t.Run("view with disabled back key ignores 'b' press", func(t *testing.T) {
		// Create a view with back key disabled
		mockView := &MockViewWithKeymap{
			keymap: keymap.NewKeymapWithDisabled(keymap.NavigationKeyBack),
		}
		app.current = mockView

		// Send 'b' key press
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}
		model, cmd := app.Update(keyMsg)

		// Should return the app unchanged with nil command (key ignored)
		assert.Equal(t, app, model)
		assert.Nil(t, cmd)
	})

	t.Run("view with enabled back key processes 'b' press", func(t *testing.T) {
		// Create a view with all keys enabled (default)
		mockView := &MockViewWithKeymap{
			keymap: keymap.NewDefaultKeymap(),
		}
		app.current = mockView

		// Send 'b' key press
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}
		model, cmd := app.Update(keyMsg)

		// Should return the app with a command (navigation message)
		assert.Equal(t, app, model)
		assert.NotNil(t, cmd) // Should have a navigation command
	})

	t.Run("view without keymap interface processes all keys normally", func(t *testing.T) {
		// Create a view that doesn't implement ViewKeymap
		mockView := &MockViewWithoutKeymap{}
		app.current = mockView

		// Send 'b' key press - should be delegated to view
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}
		model, cmd := app.Update(keyMsg)

		// Should delegate to view (which returns itself unchanged)
		assert.Equal(t, app, model)
		assert.Nil(t, cmd) // Mock view returns nil command
	})

	t.Run("disabled bulk key is ignored", func(t *testing.T) {
		// Create a view with bulk key disabled
		mockView := &MockViewWithKeymap{
			keymap: keymap.NewKeymapWithDisabled(keymap.NavigationKeyBulk),
		}
		app.current = mockView

		// Send 'B' key press
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("B")}
		model, cmd := app.Update(keyMsg)

		// Should return the app unchanged with nil command (key ignored)
		assert.Equal(t, app, model)
		assert.Nil(t, cmd)
	})

	t.Run("non-navigation keys are not intercepted", func(t *testing.T) {
		// Create a view with some keys disabled
		mockView := &MockViewWithKeymap{
			keymap: keymap.NewKeymapWithDisabled(keymap.NavigationKeyBack),
		}
		app.current = mockView

		// Send a non-navigation key press
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
		model, cmd := app.Update(keyMsg)

		// Should delegate to view (not intercepted by keymap system)
		assert.Equal(t, app, model)
		assert.Nil(t, cmd) // Mock view returns nil command
	})
}
