package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/tui/views"
)

// Integration tests for complete navigation flows
func TestCompleteNavigationFlow(t *testing.T) {
	t.Run("Dashboard -> Create -> Details -> Back to Dashboard", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		
		// Initialize app (starts with dashboard)
		_ = app.Init()
		assert.IsType(t, &views.DashboardView{}, app.current)
		assert.Len(t, app.viewStack, 0)
		
		// Navigate to Create view
		model, _ := app.Update(messages.NavigateToCreateMsg{
			SelectedRepository: "test/repo",
		})
		appModel := model.(*App)
		assert.IsType(t, &views.CreateRunView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 1)
		
		// Verify context was set
		repo := appModel.cache.GetNavigationContext("selected_repo")
		assert.Equal(t, "test/repo", repo)
		
		// Navigate to Details view
		model, _ = appModel.Update(messages.NavigateToDetailsMsg{
			RunID:      "run-123",
			FromCreate: true,
		})
		appModel = model.(*App)
		assert.IsType(t, &views.RunDetailsView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 2)
		
		// Navigate back (should go to Create)
		model, _ = appModel.Update(messages.NavigateBackMsg{})
		appModel = model.(*App)
		assert.IsType(t, &views.CreateRunView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 1)
		
		// Navigate back again (should go to Dashboard)
		model, _ = appModel.Update(messages.NavigateBackMsg{})
		appModel = model.(*App)
		assert.IsType(t, &views.DashboardView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 0)
	})
	
	t.Run("Deep navigation stack", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Build deep navigation stack
		// Dashboard -> List -> Details -> Error
		
		// Go to List
		model, _ := app.Update(messages.NavigateToListMsg{
			SelectedIndex: 5,
		})
		appModel := model.(*App)
		assert.IsType(t, &views.RunListView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 1)
		
		// Go to Details
		model, _ = appModel.Update(messages.NavigateToDetailsMsg{
			RunID: "run-456",
		})
		appModel = model.(*App)
		assert.IsType(t, &views.RunDetailsView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 2)
		
		// Go to Error (recoverable)
		model, _ = appModel.Update(messages.NavigateToErrorMsg{
			Error:       errors.New("test error"),
			Message:     "Something went wrong",
			Recoverable: true,
		})
		appModel = model.(*App)
		assert.IsType(t, &views.ErrorView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 3)
		
		// Navigate back through the entire stack
		for i := 3; i > 0; i-- {
			model, _ = appModel.Update(messages.NavigateBackMsg{})
			appModel = model.(*App)
			assert.Len(t, appModel.viewStack, i-1)
		}
		
		// Should be back at dashboard
		assert.IsType(t, &views.DashboardView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 0)
	})
	
	t.Run("Navigate to Dashboard clears stack", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Build navigation stack
		app.Update(messages.NavigateToListMsg{})
		app.Update(messages.NavigateToDetailsMsg{RunID: "123"})
		model, _ := app.Update(messages.NavigateToCreateMsg{})
		appModel := model.(*App)
		assert.Len(t, appModel.viewStack, 3)
		
		// Navigate directly to dashboard
		model, _ = appModel.Update(messages.NavigateToDashboardMsg{})
		appModel = model.(*App)
		
		assert.IsType(t, &views.DashboardView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 0) // Stack should be cleared
		
		// Verify navigation context was cleared
		assert.Nil(t, appModel.cache.GetNavigationContext("any_key"))
	})
}

func TestNavigationWithContext(t *testing.T) {
	t.Run("Context persists during navigation", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Set some context
		app.cache.SetNavigationContext("user_selection", "option1")
		app.cache.SetContext("persistent_data", "value1")
		
		// Navigate to Create
		model, _ := app.Update(messages.NavigateToCreateMsg{
			SelectedRepository: "org/repo",
		})
		appModel := model.(*App)
		
		// Both contexts should persist
		assert.Equal(t, "option1", appModel.cache.GetNavigationContext("user_selection"))
		assert.Equal(t, "value1", appModel.cache.GetContext("persistent_data"))
		assert.Equal(t, "org/repo", appModel.cache.GetNavigationContext("selected_repo"))
		
		// Navigate to Details
		model, _ = appModel.Update(messages.NavigateToDetailsMsg{RunID: "123"})
		appModel = model.(*App)
		
		// Context still persists
		assert.Equal(t, "option1", appModel.cache.GetNavigationContext("user_selection"))
		assert.Equal(t, "value1", appModel.cache.GetContext("persistent_data"))
		
		// Navigate to Dashboard (should clear navigation context)
		model, _ = appModel.Update(messages.NavigateToDashboardMsg{})
		appModel = model.(*App)
		
		// Navigation context cleared, but regular context remains
		assert.Nil(t, appModel.cache.GetNavigationContext("user_selection"))
		assert.Nil(t, appModel.cache.GetNavigationContext("selected_repo"))
		assert.Equal(t, "value1", appModel.cache.GetContext("persistent_data"))
	})
}

func TestErrorNavigation(t *testing.T) {
	t.Run("Recoverable error allows going back", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Navigate to Create
		app.Update(messages.NavigateToCreateMsg{})
		originalView := app.current
		
		// Encounter recoverable error
		model, _ := app.Update(messages.NavigateToErrorMsg{
			Error:       errors.New("validation error"),
			Message:     "Invalid input",
			Recoverable: true,
		})
		appModel := model.(*App)
		
		assert.IsType(t, &views.ErrorView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 2) // Dashboard and Create in stack
		
		// Navigate back
		model, _ = appModel.Update(messages.NavigateBackMsg{})
		appModel = model.(*App)
		
		// Should be back at Create view
		assert.Equal(t, originalView, appModel.current)
		assert.Len(t, appModel.viewStack, 1)
	})
	
	t.Run("Non-recoverable error clears stack", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Build navigation stack
		app.Update(messages.NavigateToListMsg{})
		app.Update(messages.NavigateToDetailsMsg{RunID: "123"})
		
		// Encounter non-recoverable error
		model, _ := app.Update(messages.NavigateToErrorMsg{
			Error:       errors.New("fatal error"),
			Message:     "System failure",
			Recoverable: false,
		})
		appModel := model.(*App)
		
		assert.IsType(t, &views.ErrorView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 0) // Stack cleared
		
		// Navigate back should go to dashboard (no stack)
		model, _ = appModel.Update(messages.NavigateBackMsg{})
		appModel = model.(*App)
		
		assert.IsType(t, &views.DashboardView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 0)
	})
}

func TestNavigationEdgeCases(t *testing.T) {
	t.Run("Multiple NavigateBack with empty stack", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Multiple back navigations with empty stack
		for i := 0; i < 5; i++ {
			model, _ := app.Update(messages.NavigateBackMsg{})
			appModel := model.(*App)
			
			// Should always stay at dashboard
			assert.IsType(t, &views.DashboardView{}, appModel.current)
			assert.Len(t, appModel.viewStack, 0)
		}
	})
	
	t.Run("Navigate to same view type", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Navigate to Create
		app.Update(messages.NavigateToCreateMsg{
			SelectedRepository: "repo1",
		})
		
		// Navigate to Create again (different context)
		model, _ := app.Update(messages.NavigateToCreateMsg{
			SelectedRepository: "repo2",
		})
		appModel := model.(*App)
		
		// Should have two Create views in stack
		assert.IsType(t, &views.CreateRunView{}, appModel.current)
		assert.Len(t, appModel.viewStack, 2)
		
		// Context should be updated
		assert.Equal(t, "repo2", appModel.cache.GetNavigationContext("selected_repo"))
	})
	
	t.Run("Quit during navigation", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Navigate somewhere
		app.Update(messages.NavigateToListMsg{})
		
		// Send quit command
		model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		
		assert.IsType(t, app, model)
		assert.NotNil(t, cmd) // Should return quit command
	})
}

func TestNavigationMessageDelegation(t *testing.T) {
	t.Run("Non-navigation messages are delegated", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Regular key press
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		model, _ := app.Update(msg)
		appModel := model.(*App)
		
		// Should still be at dashboard, message was delegated
		assert.IsType(t, &views.DashboardView{}, appModel.current)
		
		// Window resize
		msg2 := tea.WindowSizeMsg{Width: 100, Height: 30}
		model, _ = appModel.Update(msg2)
		appModel = model.(*App)
		
		// Should still be at dashboard, message was delegated
		assert.IsType(t, &views.DashboardView{}, appModel.current)
	})
}

func TestListWithSelectedIndex(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)
	_ = app.Init()
	
	// Navigate to list with selected index
	model, _ := app.Update(messages.NavigateToListMsg{
		SelectedIndex: 10,
	})
	appModel := model.(*App)
	
	assert.IsType(t, &views.RunListView{}, appModel.current)
	
	// Verify context was set
	index := appModel.cache.GetNavigationContext("list_selected_index")
	assert.Equal(t, 10, index)
}

func TestDetailsViewCreation(t *testing.T) {
	mockClient := &MockAPIClient{}
	app := NewApp(mockClient)
	_ = app.Init()
	
	// Navigate to details
	model, _ := app.Update(messages.NavigateToDetailsMsg{
		RunID:      "test-run-id",
		FromCreate: true,
	})
	appModel := model.(*App)
	
	assert.IsType(t, &views.RunDetailsView{}, appModel.current)
	
	// Verify the details view was created
	// We can't access the private run field, but we know it was created with the right ID
	assert.IsType(t, &views.RunDetailsView{}, appModel.current)
}

func TestNavigationMemoryManagement(t *testing.T) {
	t.Run("Large navigation stack", func(t *testing.T) {
		mockClient := &MockAPIClient{}
		app := NewApp(mockClient)
		_ = app.Init()
		
		// Build large navigation stack
		for i := 0; i < 100; i++ {
			if i%2 == 0 {
				app.Update(messages.NavigateToListMsg{SelectedIndex: i})
			} else {
				app.Update(messages.NavigateToCreateMsg{})
			}
		}
		
		// Should handle large stack
		assert.Len(t, app.viewStack, 100)
		
		// Navigate to dashboard clears it all
		model, _ := app.Update(messages.NavigateToDashboardMsg{})
		appModel := model.(*App)
		
		assert.Len(t, appModel.viewStack, 0)
	})
}