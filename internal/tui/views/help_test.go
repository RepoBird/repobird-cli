// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHelpAPIClient for testing
type MockHelpAPIClient struct {
	mock.Mock
}

func (m *MockHelpAPIClient) ListRuns(ctx context.Context, page, limit int) (*models.ListRunsResponse, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ListRunsResponse), args.Error(1)
}

func (m *MockHelpAPIClient) ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RunResponse), args.Error(1)
}

func (m *MockHelpAPIClient) GetRun(id string) (*models.RunResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *MockHelpAPIClient) GetUserInfo() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockHelpAPIClient) GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockHelpAPIClient) ListRepositories(ctx context.Context) ([]models.APIRepository, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.APIRepository), args.Error(1)
}

func (m *MockHelpAPIClient) GetAPIEndpoint() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHelpAPIClient) VerifyAuth() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockHelpAPIClient) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *MockHelpAPIClient) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FileHashEntry), args.Error(1)
}

func TestHelpView_Creation(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()

	// Create help view
	helpView := NewHelpView(mockClient, testCache)

	// Assertions
	assert.NotNil(t, helpView)
	assert.NotNil(t, helpView.client)
	assert.NotNil(t, helpView.cache)
	assert.NotNil(t, helpView.helpComponent)
	assert.NotNil(t, helpView.keys)
	assert.NotNil(t, helpView.disabledKeys)
	assert.Nil(t, helpView.layout) // Should be nil until WindowSizeMsg
	assert.Equal(t, 0, helpView.width)
	assert.Equal(t, 0, helpView.height)
}

func TestHelpView_Init(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Execute
	cmd := helpView.Init()

	// Assertions
	assert.Nil(t, cmd) // Init should not send any commands
}

func TestHelpView_WindowSizeMsg(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Send window size message
	model, cmd := helpView.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Assertions
	assert.NotNil(t, model)
	assert.Nil(t, cmd)

	updatedView := model.(*HelpView)
	assert.Equal(t, 100, updatedView.width)
	assert.Equal(t, 30, updatedView.height)
	assert.NotNil(t, updatedView.layout) // Layout should be created now
}

func TestHelpView_NavigationKeys(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Initialize with window size
	helpView.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	tests := []struct {
		name     string
		key      string
		expected tea.Msg
	}{
		{"Quit key", "q", messages.NavigateBackMsg{}},
		{"Escape key", "esc", messages.NavigateBackMsg{}},
		{"Help toggle", "?", messages.NavigateBackMsg{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, cmd := helpView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)})

			assert.NotNil(t, model)
			assert.NotNil(t, cmd)

			// Execute the command to get the message
			if cmd != nil {
				msg := cmd()
				assert.IsType(t, tt.expected, msg)
			}
		})
	}
}

func TestHelpView_ForceQuit(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Initialize with window size
	helpView.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Test force quit
	model, cmd := helpView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Q")})

	assert.NotNil(t, model)
	assert.NotNil(t, cmd)
	// Check that it's a quit command by executing it
	if cmd != nil {
		msg := cmd()
		assert.IsType(t, tea.QuitMsg{}, msg)
	}
}

func TestHelpView_ScrollingKeys(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Initialize with window size
	helpView.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Test scrolling keys (these should be passed to help component)
	scrollKeys := []string{"j", "k", "down", "up", "ctrl+d", "ctrl+u", "J", "K", "pgdown", "pgup", "g", "G"}

	for _, key := range scrollKeys {
		t.Run("Scroll key: "+key, func(t *testing.T) {
			model, _ := helpView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

			assert.NotNil(t, model)
			// Scrolling keys might or might not return a command
			// The important thing is they don't trigger navigation
		})
	}
}

func TestHelpView_CopyKeys(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Initialize with window size
	helpView.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Test copy keys (these should be passed to help component)
	copyKeys := []string{"y", "Y"}

	for _, key := range copyKeys {
		t.Run("Copy key: "+key, func(t *testing.T) {
			model, _ := helpView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

			assert.NotNil(t, model)
			// Copy operations are handled by the help component
		})
	}
}

func TestHelpView_ViewRendering(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Test before window size is set
	view := helpView.View()
	assert.Equal(t, "", view) // Should return empty string

	// Initialize with window size
	helpView.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Test after window size is set
	view = helpView.View()
	assert.NotEqual(t, "", view)         // Should render content
	assert.Contains(t, view, "RepoBird") // Should contain help content
}

func TestHelpView_IsKeyDisabled(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Test that no keys are disabled by default
	assert.False(t, helpView.IsKeyDisabled("q"))
	assert.False(t, helpView.IsKeyDisabled("esc"))
	assert.False(t, helpView.IsKeyDisabled("j"))
	assert.False(t, helpView.IsKeyDisabled("k"))
}

func TestHelpView_HandleKey(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Test that HandleKey returns false for all keys (delegates to Update)
	handled, model, cmd := helpView.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})

	assert.False(t, handled)
	assert.Equal(t, helpView, model)
	assert.Nil(t, cmd)
}

func TestHelpView_GlobalDashboardKeys(t *testing.T) {
	// Setup
	mockClient := new(MockHelpAPIClient)
	testCache := cache.NewSimpleCache()
	helpView := NewHelpView(mockClient, testCache)

	// Initialize with window size
	helpView.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Test that 'h' and 'H' keys are NOT handled by the view
	// (they should fall through to the global navigation system)
	dashboardKeys := []string{"h", "H"}

	for _, key := range dashboardKeys {
		t.Run("Dashboard key: "+key, func(t *testing.T) {
			model, cmd := helpView.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})

			// These keys should NOT trigger any navigation from the view
			// They will be handled by the global system
			assert.Equal(t, helpView, model)
			assert.Nil(t, cmd)
		})
	}
}
