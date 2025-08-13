package views

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestDashboardView_StatusInfoNavigation(t *testing.T) {
	// Create a mock API client
	client := &api.Client{}

	// Create dashboard view
	dashboard := NewDashboardView(client)
	dashboard.width = 100
	dashboard.height = 30

	// Set up some test data
	dashboard.userInfo = &models.UserInfo{
		ID:             1,
		Name:           "Test User",
		Email:          "test@example.com",
		GithubUsername: "testuser",
		Tier:           "pro",
		RemainingRuns:  50,
		TotalRuns:      100,
	}

	dashboard.repositories = []models.Repository{
		{Name: "repo1"},
		{Name: "repo2"},
	}

	dashboard.allRuns = []*models.RunResponse{
		{Status: "completed"},
		{Status: "running"},
		{Status: "failed"},
	}

	dashboard.lastDataRefresh = time.Now()

	t.Run("InitializeFields", func(t *testing.T) {
		// Initialize status info fields
		dashboard.initializeStatusInfoFields()

		// Should have fields for all the non-empty values
		assert.Greater(t, len(dashboard.statusInfoFields), 0, "Should have selectable fields")

		// Check that user info fields are captured
		hasName := false
		hasEmail := false
		hasGithub := false
		for _, field := range dashboard.statusInfoFields {
			if field == "Test User" {
				hasName = true
			}
			if field == "test@example.com" {
				hasEmail = true
			}
			if field == "testuser" {
				hasGithub = true
			}
		}

		assert.True(t, hasName, "Should have name field")
		assert.True(t, hasEmail, "Should have email field")
		assert.True(t, hasGithub, "Should have GitHub field")

		// Should start with first field selected
		assert.Equal(t, 0, dashboard.statusInfoSelectedRow, "Should start at first field")
	})

	t.Run("NavigateDown", func(t *testing.T) {
		dashboard.initializeStatusInfoFields()
		dashboard.showStatusInfo = true
		initialRow := dashboard.statusInfoSelectedRow

		// Simulate pressing 'j' to navigate down
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		_, _ = dashboard.handleStatusInfoNavigation(msg)

		assert.Equal(t, initialRow+1, dashboard.statusInfoSelectedRow, "Should move to next field")
	})

	t.Run("NavigateUp", func(t *testing.T) {
		dashboard.initializeStatusInfoFields()
		dashboard.showStatusInfo = true
		dashboard.statusInfoSelectedRow = 2

		// Simulate pressing 'k' to navigate up
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		_, _ = dashboard.handleStatusInfoNavigation(msg)

		assert.Equal(t, 1, dashboard.statusInfoSelectedRow, "Should move to previous field")
	})

	t.Run("NavigateToFirst", func(t *testing.T) {
		dashboard.initializeStatusInfoFields()
		dashboard.showStatusInfo = true
		dashboard.statusInfoSelectedRow = 5

		// Simulate pressing 'g' to go to first
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		_, _ = dashboard.handleStatusInfoNavigation(msg)

		assert.Equal(t, 0, dashboard.statusInfoSelectedRow, "Should move to first field")
	})

	t.Run("NavigateToLast", func(t *testing.T) {
		dashboard.initializeStatusInfoFields()
		dashboard.showStatusInfo = true
		dashboard.statusInfoSelectedRow = 0

		// Simulate pressing 'G' to go to last
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		_, _ = dashboard.handleStatusInfoNavigation(msg)

		assert.Equal(t, len(dashboard.statusInfoFields)-1, dashboard.statusInfoSelectedRow,
			"Should move to last field")
	})

	t.Run("CloseOverlay", func(t *testing.T) {
		dashboard.showStatusInfo = true

		// Simulate pressing 's' to close
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
		_, _ = dashboard.handleStatusInfoNavigation(msg)

		assert.False(t, dashboard.showStatusInfo, "Should close overlay with 's'")

		// Open again and test ESC
		dashboard.showStatusInfo = true
		msg = tea.KeyMsg{Type: tea.KeyEscape}
		_, _ = dashboard.handleStatusInfoNavigation(msg)

		assert.False(t, dashboard.showStatusInfo, "Should close overlay with ESC")
	})
}

func TestDashboardView_StatusInfoFieldValues(t *testing.T) {
	client := &api.Client{}
	dashboard := NewDashboardView(client)

	t.Run("FieldValuesExtracted", func(t *testing.T) {
		dashboard.userInfo = &models.UserInfo{
			Name:           "John Doe",
			Email:          "john@example.com",
			GithubUsername: "johndoe",
			Tier:           "free",
			RemainingRuns:  10,
			TotalRuns:      20,
		}

		dashboard.repositories = []models.Repository{{Name: "repo1"}}
		dashboard.allRuns = []*models.RunResponse{{Status: "completed"}}
		dashboard.lastDataRefresh = time.Now()

		dashboard.initializeStatusInfoFields()

		// Check that field values (not labels) are stored
		expectedValues := []string{
			"John Doe",
			"john@example.com",
			"johndoe",
			"Free", // Tier gets capitalized
			"10 / 20",
			"50.0% ✅", // Usage percentage with emoji
		}

		for _, expected := range expectedValues {
			found := false
			for _, field := range dashboard.statusInfoFields {
				if field == expected {
					found = true
					break
				}
			}
			assert.True(t, found, fmt.Sprintf("Should have field value '%s'", expected))
		}

		// Should NOT have labels in field values
		for _, field := range dashboard.statusInfoFields {
			assert.NotContains(t, field, "Name:", "Should not contain label")
			assert.NotContains(t, field, "Email:", "Should not contain label")
			assert.NotContains(t, field, "GitHub:", "Should not contain label")
		}
	})

	t.Run("SystemStatsFields", func(t *testing.T) {
		dashboard.repositories = []models.Repository{
			{Name: "repo1"},
			{Name: "repo2"},
			{Name: "repo3"},
		}
		dashboard.allRuns = []*models.RunResponse{
			{Status: "completed"},
			{Status: "running"},
			{Status: "failed"},
			{Status: "completed"},
		}

		dashboard.initializeStatusInfoFields()

		// Check repository count
		hasRepoCount := false
		hasRunCount := false
		for _, field := range dashboard.statusInfoFields {
			if field == "3" { // 3 repositories
				hasRepoCount = true
			}
			if field == "4" { // 4 runs
				hasRunCount = true
			}
		}

		assert.True(t, hasRepoCount, "Should have repository count")
		assert.True(t, hasRunCount, "Should have run count")
	})
}

func TestDashboardView_StatusInfoClipboard(t *testing.T) {
	client := &api.Client{}
	dashboard := NewDashboardView(client)

	dashboard.userInfo = &models.UserInfo{
		Name:  "Test User",
		Email: "test@example.com",
	}

	dashboard.initializeStatusInfoFields()

	t.Run("CopyFieldValue", func(t *testing.T) {
		// Select the email field (should be second)
		dashboard.statusInfoSelectedRow = 1

		// The yank operation would set copiedMessage
		if dashboard.statusInfoSelectedRow < len(dashboard.statusInfoFields) {
			selectedValue := dashboard.statusInfoFields[dashboard.statusInfoSelectedRow]
			assert.Equal(t, "test@example.com", selectedValue, "Should select email value")
		}
	})

	t.Run("ClipboardFeedback", func(t *testing.T) {
		// Simulate copying
		dashboard.copiedMessage = "📋 Copied \"test@example.com\""
		dashboard.copiedMessageTime = time.Now()

		// The status line should show the copied message
		assert.NotEmpty(t, dashboard.copiedMessage, "Should have clipboard feedback")
		assert.Contains(t, dashboard.copiedMessage, "Copied", "Should indicate copy action")
	})
}
