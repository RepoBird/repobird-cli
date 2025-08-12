package keymap

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestCoreKeyRegistry(t *testing.T) {
	t.Run("default registry has correct mappings", func(t *testing.T) {
		registry := NewCoreKeyRegistry()

		// Test navigation actions
		assert.Equal(t, ActionNavigateBack, registry.GetAction("b"))
		assert.Equal(t, ActionNavigateBulk, registry.GetAction("B"))
		assert.Equal(t, ActionNavigateNew, registry.GetAction("n"))
		assert.Equal(t, ActionNavigateRefresh, registry.GetAction("r"))
		assert.Equal(t, ActionNavigateQuit, registry.GetAction("q"))
		assert.Equal(t, ActionNavigateHelp, registry.GetAction("?"))

		// Test global actions
		assert.Equal(t, ActionGlobalQuit, registry.GetAction("Q"))
		assert.Equal(t, ActionGlobalQuit, registry.GetAction("ctrl+c"))

		// Test view-specific actions
		assert.Equal(t, ActionViewSpecific, registry.GetAction("s"))
		assert.Equal(t, ActionViewSpecific, registry.GetAction("f"))
		assert.Equal(t, ActionViewSpecific, registry.GetAction("enter"))
		assert.Equal(t, ActionViewSpecific, registry.GetAction("tab"))

		// Test unknown key
		assert.Equal(t, ActionViewSpecific, registry.GetAction("unknown"))
	})

	t.Run("can register custom mappings", func(t *testing.T) {
		registry := NewCoreKeyRegistry()

		// Register a custom mapping
		registry.Register("x", ActionNavigateBack, "custom back")

		// Test it works
		assert.Equal(t, ActionNavigateBack, registry.GetAction("x"))

		mapping, exists := registry.GetMapping("x")
		assert.True(t, exists)
		assert.Equal(t, "x", mapping.Key)
		assert.Equal(t, ActionNavigateBack, mapping.Action)
		assert.Equal(t, "custom back", mapping.Help)
	})
}

func TestKeyActionClassification(t *testing.T) {
	t.Run("correctly identifies navigation actions", func(t *testing.T) {
		assert.True(t, IsNavigationAction(ActionNavigateBack))
		assert.True(t, IsNavigationAction(ActionNavigateBulk))
		assert.True(t, IsNavigationAction(ActionNavigateNew))
		assert.True(t, IsNavigationAction(ActionNavigateRefresh))
		assert.True(t, IsNavigationAction(ActionNavigateQuit))
		assert.True(t, IsNavigationAction(ActionNavigateHelp))

		assert.False(t, IsNavigationAction(ActionViewSpecific))
		assert.False(t, IsNavigationAction(ActionIgnore))
		assert.False(t, IsNavigationAction(ActionGlobalQuit))
		assert.False(t, IsNavigationAction(ActionGlobalHelp))
	})

	t.Run("correctly identifies global actions", func(t *testing.T) {
		assert.True(t, IsGlobalAction(ActionGlobalQuit))
		assert.True(t, IsGlobalAction(ActionGlobalHelp))

		assert.False(t, IsGlobalAction(ActionNavigateBack))
		assert.False(t, IsGlobalAction(ActionViewSpecific))
		assert.False(t, IsGlobalAction(ActionIgnore))
	})
}

// MockView implements CoreViewKeymap for testing
type MockView struct {
	disabledKeys map[string]bool
	customKeys   map[string]func(tea.KeyMsg) (bool, tea.Model, tea.Cmd)
}

func NewMockView() *MockView {
	return &MockView{
		disabledKeys: make(map[string]bool),
		customKeys:   make(map[string]func(tea.KeyMsg) (bool, tea.Model, tea.Cmd)),
	}
}

func (m *MockView) IsKeyDisabled(keyString string) bool {
	return m.disabledKeys[keyString]
}

func (m *MockView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	if handler, exists := m.customKeys[keyMsg.String()]; exists {
		return handler(keyMsg)
	}
	return false, m, nil
}

func (m *MockView) DisableKey(key string) {
	m.disabledKeys[key] = true
}

func (m *MockView) SetCustomHandler(key string, handler func(tea.KeyMsg) (bool, tea.Model, tea.Cmd)) {
	m.customKeys[key] = handler
}

// Implement tea.Model interface for MockView
func (m *MockView) Init() tea.Cmd                           { return nil }
func (m *MockView) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *MockView) View() string                            { return "mock view" }

func TestCoreViewKeymapInterface(t *testing.T) {
	t.Run("mock view implements interface correctly", func(t *testing.T) {
		view := NewMockView()

		// Test key disabled functionality
		assert.False(t, view.IsKeyDisabled("b"))
		view.DisableKey("b")
		assert.True(t, view.IsKeyDisabled("b"))

		// Test custom handler functionality
		handlerCalled := false
		view.SetCustomHandler("x", func(keyMsg tea.KeyMsg) (bool, tea.Model, tea.Cmd) {
			handlerCalled = true
			return true, view, nil
		})

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}
		handled, model, cmd := view.HandleKey(keyMsg)

		assert.True(t, handled)
		assert.Equal(t, view, model)
		assert.Nil(t, cmd)
		assert.True(t, handlerCalled)
	})

	t.Run("interface compliance", func(t *testing.T) {
		view := NewMockView()

		// Should be able to assign to interface
		var keymap CoreViewKeymap = view

		// Should be able to call interface methods
		assert.False(t, keymap.IsKeyDisabled("test"))

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("test")}
		handled, model, cmd := keymap.HandleKey(keyMsg)
		assert.False(t, handled)
		assert.Equal(t, view, model)
		assert.Nil(t, cmd)
	})
}
