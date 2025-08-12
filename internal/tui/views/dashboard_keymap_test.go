package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/tui/keymap"
	"github.com/stretchr/testify/assert"
)

// MockAPIClient for testing
type MockAPIClient struct{}

func (m *MockAPIClient) ListRuns(ctx interface{}, page, limit int) (interface{}, error) {
	return nil, nil
}
func (m *MockAPIClient) ListRunsLegacy(limit, offset int) (interface{}, error)       { return nil, nil }
func (m *MockAPIClient) GetRun(id string) (interface{}, error)                       { return nil, nil }
func (m *MockAPIClient) GetUserInfo() (interface{}, error)                           { return nil, nil }
func (m *MockAPIClient) GetUserInfoWithContext(ctx interface{}) (interface{}, error) { return nil, nil }
func (m *MockAPIClient) ListRepositories(ctx interface{}) (interface{}, error)       { return nil, nil }
func (m *MockAPIClient) GetAPIEndpoint() string                                      { return "" }
func (m *MockAPIClient) VerifyAuth() (interface{}, error)                            { return nil, nil }
func (m *MockAPIClient) CreateRunAPI(request interface{}) (interface{}, error)       { return nil, nil }
func (m *MockAPIClient) GetFileHashes(ctx interface{}) (interface{}, error)          { return nil, nil }

func TestDashboardKeymapImplementation(t *testing.T) {
	t.Run("dashboard implements ViewKeymap interface", func(t *testing.T) {
		client := &MockAPIClient{}
		dashboard := NewDashboardView(client)

		// Check that dashboard implements the interface
		var viewKeymap keymap.ViewKeymap = dashboard

		// Verify back key is disabled
		assert.False(t, viewKeymap.IsNavigationKeyEnabled(keymap.NavigationKeyBack))

		// Verify other keys are enabled
		assert.True(t, viewKeymap.IsNavigationKeyEnabled(keymap.NavigationKeyNew))
		assert.True(t, viewKeymap.IsNavigationKeyEnabled(keymap.NavigationKeyRefresh))
		assert.True(t, viewKeymap.IsNavigationKeyEnabled(keymap.NavigationKeyStatus))
		assert.True(t, viewKeymap.IsNavigationKeyEnabled(keymap.NavigationKeyHelp))
		assert.True(t, viewKeymap.IsNavigationKeyEnabled(keymap.NavigationKeyQuit))
	})
}

func TestDashboardKeyHandling(t *testing.T) {
	t.Run("dashboard ignores disabled back key", func(t *testing.T) {
		client := &MockAPIClient{}
		dashboard := NewDashboardView(client)

		// Create a 'b' key press message
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}

		// The key should be ignored (return dashboard unchanged with nil command)
		// Note: We can't easily test the full Update flow due to complex dependencies,
		// but we can test that the keymap correctly identifies the key as disabled
		assert.False(t, dashboard.IsNavigationKeyEnabled(keymap.NavigationKeyBack))
	})
}
