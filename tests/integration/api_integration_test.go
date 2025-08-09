package integration

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/tests/helpers"
)

func TestAPIClient_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Set up mock API server
	mockServer := helpers.NewMockAPIServer(t)

	// Configure mock responses
	expectedRun := models.RunResponse{
		ID:         "integration-test-123",
		Status:     models.StatusQueued,
		Repository: "test/integration",
		Source:     "main",
		Target:     "feature",
		Prompt:     "Integration test prompt",
		Title:      "Integration Test",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	mockServer.SetCreateRunResponse(expectedRun)
	mockServer.SetGetRunResponse("integration-test-123", expectedRun)
	mockServer.SetAuthVerifyResponse(true)

	// Create API client
	client := api.NewClient("test-api-key", mockServer.URL(), false)

	t.Run("CreateRun", func(t *testing.T) {
		request := &models.RunRequest{
			Prompt:     "Integration test prompt",
			Repository: "test/integration",
			Source:     "main",
			Target:     "feature",
			RunType:    models.RunTypeRun,
			Title:      "Integration Test",
		}

		resp, err := client.CreateRun(request)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, expectedRun.ID, resp.ID)
		assert.Equal(t, expectedRun.Status, resp.Status)
		assert.Equal(t, expectedRun.Repository, resp.Repository)
	})

	t.Run("GetRun", func(t *testing.T) {
		resp, err := client.GetRun("integration-test-123")
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.Equal(t, expectedRun.ID, resp.ID)
		assert.Equal(t, expectedRun.Status, resp.Status)
	})

	t.Run("ListRuns", func(t *testing.T) {
		runs := []*models.RunResponse{&expectedRun}
		mockServer.SetRunsListResponse([]models.RunResponse{expectedRun})

		resp, err := client.ListRuns(10, 0)
		require.NoError(t, err)
		require.Len(t, resp, 1)

		assert.Equal(t, runs[0].ID, resp[0].ID)
	})

	t.Run("VerifyAuth", func(t *testing.T) {
		userInfo, err := client.VerifyAuth()
		require.NoError(t, err)
		require.NotNil(t, userInfo)

		// Mock server returns default user info in auth verify
		assert.NotEmpty(t, userInfo.Email)
	})
}

func TestAPIClient_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	mockServer := helpers.NewMockAPIServer(t)

	t.Run("Unauthorized", func(t *testing.T) {
		mockServer.SetAuthVerifyResponse(false)
		client := api.NewClient("invalid-key", mockServer.URL(), false)

		_, err := client.VerifyAuth()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "401")
	})

	t.Run("NotFound", func(t *testing.T) {
		client := api.NewClient("test-key", mockServer.URL(), false)
		mockServer.SetResponse("GET", "/api/v1/runs/nonexistent", helpers.MockResponse{
			StatusCode: 404,
			Body:       "Run not found",
		})

		_, err := client.GetRun("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})

	t.Run("RateLimited", func(t *testing.T) {
		client := api.NewClient("test-key", mockServer.URL(), false)
		mockServer.SetResponse("POST", "/api/v1/runs", helpers.MockResponse{
			StatusCode: 429,
			Body: map[string]interface{}{
				"error": map[string]interface{}{
					"message": "Rate limit exceeded",
				},
			},
		})

		request := &models.RunRequest{
			Prompt:     "Test",
			Repository: "test/repo",
			Source:     "main",
			Target:     "feature",
			RunType:    models.RunTypeRun,
		}

		_, err := client.CreateRun(request)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "429")
	})
}

func TestAPIClient_WithEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	mockServer := helpers.NewMockAPIServer(t)

	// Set up environment
	originalAPIURL := os.Getenv("REPOBIRD_API_URL")
	originalAPIKey := os.Getenv("REPOBIRD_API_KEY")

	defer func() {
		if originalAPIURL != "" {
			os.Setenv("REPOBIRD_API_URL", originalAPIURL)
		} else {
			os.Unsetenv("REPOBIRD_API_URL")
		}
		if originalAPIKey != "" {
			os.Setenv("REPOBIRD_API_KEY", originalAPIKey)
		} else {
			os.Unsetenv("REPOBIRD_API_KEY")
		}
	}()

	os.Setenv("REPOBIRD_API_URL", mockServer.URL())
	os.Setenv("REPOBIRD_API_KEY", "env-test-key")

	mockServer.SetAuthVerifyResponse(true)

	// Client should use environment variables
	client := api.NewClient(os.Getenv("REPOBIRD_API_KEY"), os.Getenv("REPOBIRD_API_URL"), false)

	userInfo, err := client.VerifyAuth()
	require.NoError(t, err)
	require.NotNil(t, userInfo)
}

func TestAPIClient_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	mockServer := helpers.NewMockAPIServer(t)
	client := api.NewClient("test-key", mockServer.URL(), false)

	// Set up responses for multiple runs
	for i := 0; i < 5; i++ {
		runID := "concurrent-test-" + string(rune('1'+i))
		mockServer.SetGetRunResponse(runID, models.RunResponse{
			ID:     runID,
			Status: models.StatusDone,
		})
	}

	// Test concurrent API calls
	results := make(chan error, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			runID := "concurrent-test-" + string(rune('1'+id))
			_, err := client.GetRun(runID)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < 5; i++ {
		select {
		case err := <-results:
			assert.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent test timed out")
		}
	}
}