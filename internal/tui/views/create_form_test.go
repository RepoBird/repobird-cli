package views

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/models"
	tuicache "github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAPIClient for testing - implements APIClient interface
type MockAPIClient struct {
	CreateRunFunc func(request *models.APIRunRequest) (*models.RunResponse, error)
}

func (m *MockAPIClient) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	if m.CreateRunFunc != nil {
		return m.CreateRunFunc(request)
	}
	return &models.RunResponse{
		ID:         "test-run-123",
		Repository: request.RepositoryName,
		Status:     "running",
	}, nil
}

func (m *MockAPIClient) GetRunAPI(id string) (*models.RunResponse, error) {
	return &models.RunResponse{ID: id, Status: "running"}, nil
}

func (m *MockAPIClient) GetRunsAPI() ([]models.RunResponse, error) {
	return []models.RunResponse{}, nil
}

func (m *MockAPIClient) GetAPIEndpoint() string {
	return "https://api.test.com"
}

// Implement remaining interface methods
func (m *MockAPIClient) ListRuns(ctx context.Context, page, limit int) (*models.ListRunsResponse, error) {
	return &models.ListRunsResponse{Runs: []models.RunResponse{}}, nil
}

func (m *MockAPIClient) ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error) {
	return []*models.RunResponse{}, nil
}

func (m *MockAPIClient) GetRun(id string) (*models.RunResponse, error) {
	return &models.RunResponse{ID: id, Status: "running"}, nil
}

func (m *MockAPIClient) GetUserInfo() (*models.UserInfo, error) {
	return &models.UserInfo{ID: 1, Name: "Test User"}, nil
}

func (m *MockAPIClient) GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error) {
	return &models.UserInfo{ID: 1, Name: "Test User"}, nil
}

func (m *MockAPIClient) ListRepositories(ctx context.Context) ([]models.APIRepository, error) {
	return []models.APIRepository{}, nil
}

func (m *MockAPIClient) VerifyAuth() (*models.UserInfo, error) {
	return &models.UserInfo{ID: 1, Name: "Test User"}, nil
}

func (m *MockAPIClient) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	return []models.FileHashEntry{}, nil
}

func TestCreateRunView_InitWithCachedRepository(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create cache and set last used repository
	cache := tuicache.NewSimpleCache()
	err := cache.SetLastUsedRepository("cached/repository")
	require.NoError(t, err)

	// Create view
	client := &MockAPIClient{}
	view := NewCreateRunView(client, cache)

	// Initialize the view
	cmd := view.Init()
	assert.NotNil(t, cmd)

	// Check that repository field is populated with cached value
	values := view.form.GetValues()
	assert.Equal(t, "cached/repository", values["repository"])

	// Source branch should be empty (no git detection)
	assert.Equal(t, "", values["source"])

	// Target branch should be empty
	assert.Equal(t, "", values["target"])
}

func TestCreateRunView_InitWithNavigationContext(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := tuicache.NewSimpleCache()

	// Set navigation context (has priority over cached repository)
	cache.SetNavigationContext("selected_repo", "nav/context/repo")

	// Also set last used repository
	err := cache.SetLastUsedRepository("cached/repository")
	require.NoError(t, err)

	client := &MockAPIClient{}
	view := NewCreateRunView(client, cache)

	// Initialize
	_ = view.Init()

	// Navigation context should take priority
	values := view.form.GetValues()
	assert.Equal(t, "nav/context/repo", values["repository"])
}

func TestCreateRunView_InitFallbackToRecentRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := tuicache.NewSimpleCache()

	// Add some runs to cache (no last repository set)
	runs := []models.RunResponse{
		{
			ID:         "run1",
			Repository: "recent/repo1",
			Status:     "completed",
		},
		{
			ID:         "run2",
			Repository: "recent/repo2",
			Status:     "running",
		},
	}
	cache.SetRuns(runs)

	client := &MockAPIClient{}
	view := NewCreateRunView(client, cache)

	// Initialize
	_ = view.Init()

	// Should use repository from most recent run
	values := view.form.GetValues()
	assert.Equal(t, "recent/repo1", values["repository"])
}

func TestCreateRunView_SavesRepositoryOnSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := tuicache.NewSimpleCache()

	// Initially no last repository
	_, found := cache.GetLastUsedRepository()
	assert.False(t, found)

	client := &MockAPIClient{
		CreateRunFunc: func(request *models.APIRunRequest) (*models.RunResponse, error) {
			return &models.RunResponse{
				ID:         "new-run-123",
				Repository: request.RepositoryName,
				Status:     "running",
			}, nil
		},
	}

	view := NewCreateRunView(client, cache)

	// Simulate successful run creation
	msg := runCreatedMsg{
		run: &models.RunResponse{
			ID:         "new-run-123",
			Repository: "newly/created/repo",
			Status:     "running",
		},
		err: nil,
	}

	// Process the message
	newModel, cmd := view.handleRunCreated(msg)
	assert.NotNil(t, newModel)
	assert.NotNil(t, cmd)

	// Repository should be saved to cache
	repo, found := cache.GetLastUsedRepository()
	assert.True(t, found)
	assert.Equal(t, "newly/created/repo", repo)
}

func TestCreateRunView_NoGitDetection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Create a cache with no data
	cache := tuicache.NewSimpleCache()

	client := &MockAPIClient{}
	view := NewCreateRunView(client, cache)

	// Initialize
	_ = view.Init()

	values := view.form.GetValues()

	// Repository should be empty (no git detection, no cache)
	assert.Equal(t, "", values["repository"])

	// Source branch should be empty (no git detection)
	assert.Equal(t, "", values["source"])

	// Target branch should be empty
	assert.Equal(t, "", values["target"])
}

func TestCreateRunView_FormFieldDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := tuicache.NewSimpleCache()
	client := &MockAPIClient{}
	view := NewCreateRunView(client, cache)

	// Check form field defaults
	values := view.form.GetValues()

	// Title should be empty
	assert.Equal(t, "", values["title"])

	// Source should be empty (no default "main")
	assert.Equal(t, "", values["source"])

	// Target should be empty
	assert.Equal(t, "", values["target"])

	// Prompt should be empty
	assert.Equal(t, "", values["prompt"])

	// Context should be empty
	assert.Equal(t, "", values["context"])

	// Run type should default to "run"
	assert.Equal(t, "run", values["runtype"])
}

func TestCreateRunView_RepositoryCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// First session - create a run
	cache1 := tuicache.NewSimpleCache()
	client := &MockAPIClient{}
	view1 := NewCreateRunView(client, cache1)

	// Simulate run creation
	msg := runCreatedMsg{
		run: &models.RunResponse{
			ID:         "run-456",
			Repository: "persisted/repo",
			Status:     "completed",
		},
		err: nil,
	}

	_, _ = view1.handleRunCreated(msg)
	cache1.Stop()

	// Second session - should retrieve cached repository
	cache2 := tuicache.NewSimpleCache()
	view2 := NewCreateRunView(client, cache2)

	_ = view2.Init()

	values := view2.form.GetValues()
	assert.Equal(t, "persisted/repo", values["repository"])

	cache2.Stop()
}

func TestCreateRunView_HandleKeyMsg(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cache := tuicache.NewSimpleCache()
	client := &MockAPIClient{}
	view := NewCreateRunView(client, cache)

	// Test ESC key handling in insert mode
	view.form.SetInsertMode(true)
	handled, _, _ := view.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, handled)
	assert.False(t, view.form.IsInsertMode())

	// Test ESC key in normal mode
	view.form.SetInsertMode(false)
	handled, _, _ = view.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, handled)
	assert.False(t, view.form.IsInsertMode())
}
