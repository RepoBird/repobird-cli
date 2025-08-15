package views

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/stretchr/testify/assert"
)

// MockClientWithRetryExhaustion simulates a client that fails with retry exhaustion
type MockClientWithRetryExhaustion struct {
	failListRepositories bool
	failListRuns        bool
}

func (m *MockClientWithRetryExhaustion) ListRepositories(ctx context.Context) ([]models.APIRepository, error) {
	if m.failListRepositories {
		return nil, fmt.Errorf("giving up after 3 attempts: UNAUTHORIZED")
	}
	return []models.APIRepository{}, nil
}

func (m *MockClientWithRetryExhaustion) ListRuns(ctx context.Context, page, limit int) (*models.ListRunsResponse, error) {
	if m.failListRuns {
		return nil, fmt.Errorf("giving up after 3 attempts: server error")
	}
	return &models.ListRunsResponse{
		Data: []*models.RunResponse{},
	}, nil
}

func (m *MockClientWithRetryExhaustion) ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error) {
	return []*models.RunResponse{}, nil
}

func (m *MockClientWithRetryExhaustion) GetRun(id string) (*models.RunResponse, error) {
	return &models.RunResponse{ID: id}, nil
}

func (m *MockClientWithRetryExhaustion) GetUserInfo() (*models.UserInfo, error) {
	return &models.UserInfo{ID: 1, Email: "test@example.com"}, nil
}

func (m *MockClientWithRetryExhaustion) GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error) {
	return m.GetUserInfo()
}

func (m *MockClientWithRetryExhaustion) GetAPIEndpoint() string {
	return "https://api.test.com"
}

func (m *MockClientWithRetryExhaustion) VerifyAuth() (*models.UserInfo, error) {
	return m.GetUserInfo()
}

func (m *MockClientWithRetryExhaustion) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	return &models.RunResponse{ID: "test-123"}, nil
}

func (m *MockClientWithRetryExhaustion) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	return []models.FileHashEntry{}, nil
}

func TestDashboardRetryExhaustion(t *testing.T) {
	t.Run("Navigate to error view after retry exhaustion for ListRepositories", func(t *testing.T) {
		// Create mock client that simulates retry exhaustion
		mockClient := &MockClientWithRetryExhaustion{
			failListRepositories: true,
		}
		
		// Create cache
		mockCache := cache.NewSimpleCache()
		
		// Create dashboard view
		dashboard := NewDashboardView(mockClient, mockCache)
		
		// Initialize the view
		cmd := dashboard.Init()
		assert.NotNil(t, cmd)
		
		// Execute the loadDashboardData command
		msg := dashboard.loadDashboardData()()
		
		// Check that it's a dashboardDataLoadedMsg with error and retryExhausted flag
		loadedMsg, ok := msg.(dashboardDataLoadedMsg)
		assert.True(t, ok)
		assert.Error(t, loadedMsg.error)
		assert.True(t, loadedMsg.retryExhausted)
		assert.Contains(t, loadedMsg.error.Error(), "giving up after")
		assert.Contains(t, loadedMsg.error.Error(), "attempts")
		
		// Update dashboard with the message
		updatedView, cmd := dashboard.Update(loadedMsg)
		assert.NotNil(t, cmd)
		
		// The command should return a NavigateToErrorMsg
		navMsg := cmd()
		errorNavMsg, ok := navMsg.(messages.NavigateToErrorMsg)
		assert.True(t, ok)
		assert.NotNil(t, errorNavMsg.Error)
		assert.Equal(t, "Failed to load dashboard after 3 attempts", errorNavMsg.Message)
		assert.True(t, errorNavMsg.Recoverable)
		
		// Dashboard should have loading set to false
		dashboardView := updatedView.(*DashboardView)
		assert.False(t, dashboardView.loading)
		assert.False(t, dashboardView.initializing)
	})
	
	t.Run("Navigate to error view after retry exhaustion for ListRuns fallback", func(t *testing.T) {
		// Create mock client that simulates retry exhaustion for ListRuns
		mockClient := &MockClientWithRetryExhaustion{
			failListRepositories: true, // Fail repositories to trigger fallback
			failListRuns:        true,  // Also fail runs
		}
		
		// Create cache
		mockCache := cache.NewSimpleCache()
		
		// Create dashboard view
		dashboard := NewDashboardView(mockClient, mockCache)
		
		// Execute the loadFromRunsOnly command (fallback)
		msg := dashboard.loadFromRunsOnly()()
		
		// Check that it's a dashboardDataLoadedMsg with error and retryExhausted flag
		loadedMsg, ok := msg.(dashboardDataLoadedMsg)
		assert.True(t, ok)
		assert.Error(t, loadedMsg.error)
		assert.True(t, loadedMsg.retryExhausted)
		assert.Contains(t, loadedMsg.error.Error(), "giving up after")
		
		// Update dashboard with the message
		_, cmd := dashboard.Update(loadedMsg)
		assert.NotNil(t, cmd)
		
		// The command should return a NavigateToErrorMsg
		navMsg := cmd()
		errorNavMsg, ok := navMsg.(messages.NavigateToErrorMsg)
		assert.True(t, ok)
		assert.NotNil(t, errorNavMsg.Error)
		assert.Equal(t, "Failed to load dashboard after 3 attempts", errorNavMsg.Message)
		assert.True(t, errorNavMsg.Recoverable)
	})
	
	t.Run("Regular error without retry exhaustion shows inline error", func(t *testing.T) {
		// Create mock client that doesn't fail with retry exhaustion
		mockClient := &MockClientWithRetryExhaustion{
			failListRepositories: false,
			failListRuns:        false,
		}
		
		// Create cache
		mockCache := cache.NewSimpleCache()
		
		// Create dashboard view
		dashboard := NewDashboardView(mockClient, mockCache)
		
		// Simulate a regular error (not retry exhaustion)
		loadedMsg := dashboardDataLoadedMsg{
			error:          errors.New("network error"),
			retryExhausted: false,
		}
		
		// Update dashboard with the message
		updatedView, cmd := dashboard.Update(loadedMsg)
		
		// Should not navigate to error view
		assert.Nil(t, cmd)
		
		// Dashboard should have the error set for inline display
		dashboardView := updatedView.(*DashboardView)
		assert.Error(t, dashboardView.error)
		assert.Equal(t, "network error", dashboardView.error.Error())
	})
}

func TestIsRetryExhausted(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Retry exhaustion error",
			err:      fmt.Errorf("giving up after 3 attempts: connection refused"),
			expected: true,
		},
		{
			name:     "Retry exhaustion with different attempt count",
			err:      fmt.Errorf("giving up after 5 attempts: timeout"),
			expected: true,
		},
		{
			name:     "Regular error",
			err:      fmt.Errorf("connection refused"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Error with 'attempts' but not retry pattern",
			err:      fmt.Errorf("failed after multiple attempts"),
			expected: false,
		},
		{
			name:     "Error with 'giving up' but not full pattern",
			err:      fmt.Errorf("giving up on this task"),
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryExhausted(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}