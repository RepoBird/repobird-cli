package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/tui/views"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function to simulate authentication completion
// Since authCompleteMsg is not exported, we'll simulate it by manually setting the fields
func completeAuthentication(app *App) {
	// Simulate successful authentication by setting authenticated flag
	// and creating the dashboard view manually
	app.authenticated = true
	app.cache = cache.NewSimpleCache()
	app.current = views.NewDashboardView(app.client)
}

// MockAPIClient for testing
type MockAPIClient struct {
	mock.Mock
}

func (m *MockAPIClient) ListRuns(ctx context.Context, page, limit int) (*models.ListRunsResponse, error) {
	args := m.Called(ctx, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ListRunsResponse), args.Error(1)
}

func (m *MockAPIClient) ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error) {
	args := m.Called(limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.RunResponse), args.Error(1)
}

func (m *MockAPIClient) GetRun(id string) (*models.RunResponse, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *MockAPIClient) GetUserInfo() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockAPIClient) GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockAPIClient) ListRepositories(ctx context.Context) ([]models.APIRepository, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.APIRepository), args.Error(1)
}

func (m *MockAPIClient) GetAPIEndpoint() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockAPIClient) VerifyAuth() (*models.UserInfo, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.UserInfo), args.Error(1)
}

func (m *MockAPIClient) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	args := m.Called(request)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RunResponse), args.Error(1)
}

func (m *MockAPIClient) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.FileHashEntry), args.Error(1)
}

func TestAppInitialization(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)

	assert.NotNil(t, app)
	assert.Equal(t, mockClient, app.client)
	assert.Nil(t, app.cache) // Cache is nil until authentication completes
	assert.Empty(t, app.viewStack)
	assert.Nil(t, app.current)
}

func TestAppInit(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)

	cmd := app.Init()

	// After Init, authentication starts but current is still nil
	assert.Nil(t, app.current) // Dashboard created after authentication completes
	assert.NotNil(t, cmd) // Should return authentication command
}

func TestAppViewMethod(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)

	// Before Init
	view := app.View()
	assert.Equal(t, "🔐 Authenticating...", view)

	// After Init (but before authentication completes)
	_ = app.Init()
	view = app.View()
	assert.Equal(t, "🔐 Authenticating...", view) // Still authenticating
}

func TestAppNavigationMessages(t *testing.T) {
	tests := []struct {
		name            string
		msg             messages.NavigationMsg
		expectStackSize int
		expectViewType  string
	}{
		{
			name:            "Navigate to Create",
			msg:             messages.NavigateToCreateMsg{SelectedRepository: "test/repo"},
			expectStackSize: 1, // Dashboard pushed to stack
			expectViewType:  "*views.CreateRunView",
		},
		{
			name:            "Navigate to Details",
			msg:             messages.NavigateToDetailsMsg{RunID: "123"},
			expectStackSize: 1,
			expectViewType:  "*views.RunDetailsView",
		},
		{
			name:            "Navigate to Dashboard",
			msg:             messages.NavigateToDashboardMsg{},
			expectStackSize: 0, // Stack cleared
			expectViewType:  "*views.DashboardView",
		},
		{
			name:            "Navigate to List",
			msg:             messages.NavigateToListMsg{SelectedIndex: 5},
			expectStackSize: 1,
			expectViewType:  "*views.RunListView",
		},
		{
			name:            "Navigate to Error",
			msg:             messages.NavigateToErrorMsg{Message: "Test error", Recoverable: true},
			expectStackSize: 1, // Recoverable, so current pushed to stack
			expectViewType:  "*views.ErrorView",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAPIClient{}
			app := NewApp(mockClient)
			_ = app.Init() // Initialize with authentication
			completeAuthentication(app) // Complete authentication to enable navigation

			// Handle navigation
			model, _ := app.handleNavigation(tt.msg)

			assert.IsType(t, app, model)
			appModel := model.(*App)
			assert.Len(t, appModel.viewStack, tt.expectStackSize)
			// Note: cmd can be nil if the view's Init() returns nil (e.g., ErrorView)
		})
	}
}

func TestAppNavigateBack(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)
	_ = app.Init()
	completeAuthentication(app)

	// Save initial dashboard
	initialView := app.current

	// Navigate to create view
	app.handleNavigation(messages.NavigateToCreateMsg{})
	assert.Len(t, app.viewStack, 1)

	// Navigate back
	model, _ := app.handleNavigation(messages.NavigateBackMsg{})
	appModel := model.(*App)

	assert.Len(t, appModel.viewStack, 0)
	assert.Equal(t, initialView, appModel.current)
}

func TestAppNavigateBackWithEmptyStack(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)
	_ = app.Init()
	completeAuthentication(app)

	// Navigate back with empty stack should go to dashboard
	model, _ := app.handleNavigation(messages.NavigateBackMsg{})
	appModel := model.(*App)

	assert.Len(t, appModel.viewStack, 0)
	assert.IsType(t, &views.DashboardView{}, appModel.current)
}

func TestAppNavigationContext(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)
	completeAuthentication(app) // Need cache for navigation context

	// Test setting navigation context
	app.setNavigationContext("test_key", "test_value")

	// Test getting navigation context
	value := app.getNavigationContext("test_key")
	assert.Equal(t, "test_value", value)

	// Test clearing all navigation context
	app.setNavigationContext("another_key", "another_value")
	app.clearAllNavigationContext()

	value = app.getNavigationContext("test_key")
	assert.Nil(t, value)
	value = app.getNavigationContext("another_key")
	assert.Nil(t, value)
}

func TestAppBulkViewNavigation(t *testing.T) {
	t.Run("With api.Client", func(t *testing.T) {
		// Use real api.Client for this test
		apiClient := &api.Client{}
		app := NewApp(apiClient)
		_ = app.Init()
		completeAuthentication(app)

		model, cmd := app.handleNavigation(messages.NavigateToBulkMsg{})
		appModel := model.(*App)

		assert.NotNil(t, appModel.current)
		assert.IsType(t, &views.BulkView{}, appModel.current)
		assert.NotNil(t, cmd)
	})

	t.Run("With MockAPIClient", func(t *testing.T) {
		// Mock client shouldn't create bulk view
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		completeAuthentication(app)

		initialView := app.current
		model, cmd := app.handleNavigation(messages.NavigateToBulkMsg{})
		appModel := model.(*App)

		// Should not navigate since client is not *api.Client
		assert.Equal(t, initialView, appModel.current)
		assert.Nil(t, cmd)
	})
}

func TestAppUpdate(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)
	_ = app.Init()
	completeAuthentication(app)

	t.Run("Handle navigation message", func(t *testing.T) {
		msg := messages.NavigateToCreateMsg{}
		model, _ := app.Update(msg)

		assert.IsType(t, app, model)
		appModel := model.(*App)
		assert.IsType(t, &views.CreateRunView{}, appModel.current)
	})

	t.Run("Handle quit message", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		_, cmd := app.Update(msg)

		// Should return quit command
		assert.NotNil(t, cmd)
	})

	t.Run("Delegate to current view", func(t *testing.T) {
		// Regular key message should be delegated to current view
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		model, _ := app.Update(msg)

		assert.IsType(t, app, model)
	})
}

func TestAppNavigationWithContext(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)
	_ = app.Init()
	completeAuthentication(app)

	t.Run("Navigate to Create with repository context", func(t *testing.T) {
		msg := messages.NavigateToCreateMsg{
			SelectedRepository: "org/repo",
		}

		model, _ := app.handleNavigation(msg)
		appModel := model.(*App)

		// Check context was set
		repo := appModel.cache.GetNavigationContext("selected_repo")
		assert.Equal(t, "org/repo", repo)
	})

	t.Run("Navigate to List with selected index", func(t *testing.T) {
		msg := messages.NavigateToListMsg{
			SelectedIndex: 10,
		}

		model, _ := app.handleNavigation(msg)
		appModel := model.(*App)

		// Check context was set
		index := appModel.cache.GetNavigationContext("list_selected_index")
		assert.Equal(t, 10, index)
	})

	t.Run("Navigate to Dashboard clears context", func(t *testing.T) {
		// Set some context
		app.setNavigationContext("test", "value")

		model, _ := app.handleNavigation(messages.NavigateToDashboardMsg{})
		appModel := model.(*App)

		// Context should be cleared
		value := appModel.cache.GetNavigationContext("test")
		assert.Nil(t, value)
	})
}

func TestAppErrorNavigation(t *testing.T) {
	t.Run("Recoverable error", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		completeAuthentication(app)
		
		initialView := app.current

		msg := messages.NavigateToErrorMsg{
			Message:     "Test error",
			Recoverable: true,
		}

		model, _ := app.handleNavigation(msg)
		appModel := model.(*App)

		// Should push current to stack
		assert.Len(t, appModel.viewStack, 1)
		assert.Equal(t, initialView, appModel.viewStack[0])
		assert.IsType(t, &views.ErrorView{}, appModel.current)
	})

	t.Run("Non-recoverable error", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		completeAuthentication(app)
		
		// First navigate somewhere
		app.handleNavigation(messages.NavigateToCreateMsg{})
		assert.Len(t, app.viewStack, 1)

		msg := messages.NavigateToErrorMsg{
			Message:     "Fatal error",
			Recoverable: false,
		}

		model, _ := app.handleNavigation(msg)
		appModel := model.(*App)

		// Should clear stack
		assert.Len(t, appModel.viewStack, 0)
		assert.IsType(t, &views.ErrorView{}, appModel.current)
	})
}
