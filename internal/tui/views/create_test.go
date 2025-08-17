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

// MockCreateAPIClient for testing
type MockCreateAPIClient struct {
	mock.Mock
}

func (m *MockCreateAPIClient) ListRuns(ctx context.Context, page, limit int) (*models.ListRunsResponse, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ListRunsResponse), args.Error(1)
}

func (m *MockCreateAPIClient) ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RunResponse), args.Error(1)
}

func (m *MockCreateAPIClient) GetRun(id string) (*models.RunResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *MockCreateAPIClient) GetUserInfo() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockCreateAPIClient) GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockCreateAPIClient) ListRepositories(ctx context.Context) ([]models.APIRepository, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.APIRepository), args.Error(1)
}

func (m *MockCreateAPIClient) GetAPIEndpoint() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockCreateAPIClient) VerifyAuth() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockCreateAPIClient) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *MockCreateAPIClient) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FileHashEntry), args.Error(1)
}

func TestCreateRunView_HandleKey_ESC(t *testing.T) {
	tests := []struct {
		name           string
		insertMode     bool
		key            string
		wantHandled    bool
		wantInsertMode bool
		wantCmd        bool
	}{
		{
			name:           "ESC in insert mode exits to normal mode",
			insertMode:     true,
			key:            "esc",
			wantHandled:    true,
			wantInsertMode: false,
			wantCmd:        false,
		},
		{
			name:           "ESC in normal mode does nothing",
			insertMode:     false,
			key:            "esc",
			wantHandled:    true,
			wantInsertMode: false,
			wantCmd:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create view
			client := &MockCreateAPIClient{}
			cache := cache.NewSimpleCache()
			view := NewCreateRunView(client, cache)

			// Set initial mode
			view.form.SetInsertMode(tt.insertMode)

			// Create key message
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune(tt.key),
			}
			if tt.key == "esc" {
				keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
			}

			// Call HandleKey
			handled, _, cmd := view.HandleKey(keyMsg)

			// Check results
			assert.Equal(t, tt.wantHandled, handled, "handled mismatch")
			assert.Equal(t, tt.wantInsertMode, view.form.IsInsertMode(), "insert mode mismatch")
			assert.Equal(t, tt.wantCmd, cmd != nil, "cmd presence mismatch")
		})
	}
}

func TestCreateRunView_HandleKey_Navigation(t *testing.T) {
	tests := []struct {
		name        string
		insertMode  bool
		key         string
		wantHandled bool
		wantNavMsg  bool
	}{
		{
			name:        "q in normal mode navigates back",
			insertMode:  false,
			key:         "q",
			wantHandled: false, // Navigation keys are now handled by App, not the view
			wantNavMsg:  false,
		},
		{
			name:        "b in normal mode navigates back",
			insertMode:  false,
			key:         "b",
			wantHandled: false, // Navigation keys are now handled by App, not the view
			wantNavMsg:  false,
		},
		{
			name:        "q in insert mode is not handled (types q)",
			insertMode:  true,
			key:         "q",
			wantHandled: false,
			wantNavMsg:  false,
		},
		{
			name:        "b in insert mode is not handled (types b)",
			insertMode:  true,
			key:         "b",
			wantHandled: false,
			wantNavMsg:  false,
		},
		{
			name:        "h in insert mode is not handled (types h, doesn't navigate)",
			insertMode:  true,
			key:         "h",
			wantHandled: false,
			wantNavMsg:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create view
			client := &MockCreateAPIClient{}
			cache := cache.NewSimpleCache()
			view := NewCreateRunView(client, cache)

			// Set initial mode
			view.form.SetInsertMode(tt.insertMode)

			// Create key message
			keyMsg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune(tt.key),
			}

			// Call HandleKey
			handled, _, cmd := view.HandleKey(keyMsg)

			// Check results
			assert.Equal(t, tt.wantHandled, handled, "handled mismatch")

			if tt.wantNavMsg {
				assert.NotNil(t, cmd, "expected navigation command")
				// Execute the command to get the message
				if cmd != nil {
					msg := cmd()
					_, isNavBack := msg.(messages.NavigateBackMsg)
					assert.True(t, isNavBack, "expected NavigateBackMsg")
				}
			} else {
				if cmd != nil {
					// If there's a command, it should not be a navigation message
					msg := cmd()
					_, isNavBack := msg.(messages.NavigateBackMsg)
					assert.False(t, isNavBack, "unexpected NavigateBackMsg")
				}
			}
		})
	}
}

func TestCreateRunView_HandleKey_Backspace(t *testing.T) {
	// Create view
	client := &MockCreateAPIClient{}
	cache := cache.NewSimpleCache()
	view := NewCreateRunView(client, cache)

	// Test backspace in normal mode (should be blocked)
	view.form.SetInsertMode(false)
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}

	handled, _, cmd := view.HandleKey(keyMsg)

	assert.True(t, handled, "backspace in normal mode should be handled")
	assert.Nil(t, cmd, "backspace in normal mode should not produce command")
}

func TestCreateRunView_IsKeyDisabled(t *testing.T) {
	tests := []struct {
		name         string
		insertMode   bool
		key          string
		wantDisabled bool
	}{
		// Insert mode tests
		{
			name:         "h is disabled in insert mode",
			insertMode:   true,
			key:          "h",
			wantDisabled: true,
		},
		{
			name:         "j is disabled in insert mode",
			insertMode:   true,
			key:          "j",
			wantDisabled: true,
		},
		{
			name:         "k is disabled in insert mode",
			insertMode:   true,
			key:          "k",
			wantDisabled: true,
		},
		{
			name:         "l is disabled in insert mode",
			insertMode:   true,
			key:          "l",
			wantDisabled: true,
		},
		{
			name:         "q is disabled in insert mode",
			insertMode:   true,
			key:          "q",
			wantDisabled: true,
		},
		{
			name:         "backspace is disabled in insert mode",
			insertMode:   true,
			key:          "backspace",
			wantDisabled: true,
		},
		{
			name:         "esc is NOT disabled in insert mode",
			insertMode:   true,
			key:          "esc",
			wantDisabled: false,
		},
		{
			name:         "ctrl+c is NOT disabled in insert mode",
			insertMode:   true,
			key:          "ctrl+c",
			wantDisabled: false,
		},
		// Normal mode tests
		{
			name:         "h is NOT disabled in normal mode",
			insertMode:   false,
			key:          "h",
			wantDisabled: false,
		},
		{
			name:         "q is NOT disabled in normal mode",
			insertMode:   false,
			key:          "q",
			wantDisabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create view
			client := &MockCreateAPIClient{}
			cache := cache.NewSimpleCache()
			view := NewCreateRunView(client, cache)

			// Set insert mode
			view.form.SetInsertMode(tt.insertMode)

			// Test IsKeyDisabled
			gotDisabled := view.IsKeyDisabled(tt.key)
			assert.Equal(t, tt.wantDisabled, gotDisabled, "IsKeyDisabled(%s) = %v, want %v", tt.key, gotDisabled, tt.wantDisabled)
		})
	}
}

func TestCreateRunView_FormStatePersistence(t *testing.T) {
	// Create view
	client := &MockCreateAPIClient{}
	cacheInstance := cache.NewSimpleCache()
	view := NewCreateRunView(client, cacheInstance)

	// Set some form data
	view.form.SetValue("title", "Test Title")
	view.form.SetValue("repository", "test/repo")
	view.form.SetValue("prompt", "Test prompt")

	// Save form data
	view.saveFormData()

	// Check that data was saved to cache
	formData := cacheInstance.GetFormData()
	assert.NotNil(t, formData)
	assert.Equal(t, "Test Title", formData.Title)
	assert.Equal(t, "test/repo", formData.Repository)
	assert.Equal(t, "Test prompt", formData.Prompt)

	// Create a new view with the same cache
	view2 := NewCreateRunView(client, cacheInstance)
	view2.Init()

	// Check that data was restored
	values := view2.form.GetValues()
	assert.Equal(t, "Test Title", values["title"])
	assert.Equal(t, "test/repo", values["repository"])
	assert.Equal(t, "Test prompt", values["prompt"])
}
