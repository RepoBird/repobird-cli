package mock

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

var (
	statusOptions = []models.RunStatus{
		models.StatusQueued,
		models.StatusInitializing,
		models.StatusProcessing,
		models.StatusPostProcess,
		models.StatusDone,
		models.StatusFailed,
	}

	runTypes = []string{
		"run",
		"approval",
	}

	repositories = []string{
		"facebook/react",
		"vuejs/vue",
		"angular/angular",
		"vercel/next.js",
		"sveltejs/svelte",
		"remix-run/remix",
		"gatsbyjs/gatsby",
		"nuxt/nuxt",
		"emberjs/ember.js",
		"preactjs/preact",
		"solidjs/solid",
		"alpinejs/alpine",
		"lit/lit",
		"aurelia/framework",
		"polymer/polymer",
		"mithriljs/mithril.js",
		"riotjs/riot",
		"hyperapp/hyperapp",
		"infernojs/inferno",
		"cyclejs/cyclejs",
		"qwik-dev/qwik",
		"withastro/astro",
		"redwoodjs/redwood",
		"blitz-js/blitz",
		"microsoft/typescript",
		"nodejs/node",
		"denoland/deno",
		"rust-lang/rust",
		"golang/go",
		"python/cpython",
	}

	branches = []string{
		"main",
		"master",
		"develop",
		"staging",
		"production",
		"feature/auth",
		"feature/payments",
		"fix/login-bug",
		"fix/memory-leak",
		"chore/update-deps",
	}

	titles = []string{
		"Implement user authentication flow",
		"Fix memory leak in production",
		"Add payment processing integration",
		"Optimize database queries",
		"Refactor legacy codebase",
		"Update dependencies to latest versions",
		"Add unit tests for core modules",
		"Implement dark mode theme",
		"Fix responsive design issues",
		"Add CI/CD pipeline configuration",
		"Migrate to TypeScript",
		"Implement caching strategy",
		"Add error boundary components",
		"Fix accessibility issues",
		"Implement real-time notifications",
		"Add internationalization support",
		"Optimize bundle size",
		"Fix security vulnerabilities",
		"Add API rate limiting",
		"Implement search functionality",
	}

	prompts = []string{
		"Please review and optimize the authentication flow for better security",
		"Fix the memory leak issue that's causing server crashes",
		"Integrate Stripe payment processing with webhook support",
		"Optimize all N+1 queries in the application",
		"Refactor the legacy code to use modern patterns",
		"Update all dependencies and fix any breaking changes",
		"Add comprehensive test coverage for the user service",
		"Implement a dark mode toggle with system preference detection",
		"Fix all responsive design issues on mobile devices",
		"Set up GitHub Actions for automated testing and deployment",
		"Convert the entire codebase from JavaScript to TypeScript",
		"Implement Redis caching for frequently accessed data",
		"Add error boundaries to prevent app crashes",
		"Fix all WCAG 2.1 AA accessibility violations",
		"Implement WebSocket-based real-time notifications",
		"Add i18n support for English, Spanish, and French",
		"Reduce the bundle size by at least 30%",
		"Fix all critical and high security vulnerabilities",
		"Implement rate limiting to prevent API abuse",
		"Add full-text search with Elasticsearch integration",
	}

	errorMessages = []string{
		"Failed to connect to database",
		"Authentication token expired",
		"Rate limit exceeded",
		"Invalid input parameters",
		"Service temporarily unavailable",
		"Insufficient permissions",
		"Resource not found",
		"Timeout exceeded",
		"Memory allocation failed",
		"Network connection error",
	}

	commitMessages = []string{
		"feat: add user authentication with JWT tokens",
		"fix: resolve memory leak in event listeners",
		"feat: integrate Stripe payment processing",
		"perf: optimize database queries with indexing",
		"refactor: modernize codebase with ES6+ features",
		"chore: update dependencies to latest versions",
		"test: add unit tests for user service",
		"feat: implement dark mode theme toggle",
		"fix: responsive design issues on mobile",
		"ci: add GitHub Actions workflow",
		"refactor: migrate to TypeScript",
		"perf: implement Redis caching layer",
		"fix: add error boundary components",
		"fix: resolve accessibility violations",
		"feat: add WebSocket notifications",
		"feat: add i18n support",
		"perf: reduce bundle size by 35%",
		"security: fix critical vulnerabilities",
		"feat: add API rate limiting",
		"feat: implement Elasticsearch search",
	}
)

// GenerateMockRuns creates a large set of mock runs for testing
func GenerateMockRuns(numRepos, runsPerRepo int) []*models.RunResponse {
	// rand.Seed is deprecated in Go 1.20+, no longer needed
	runs := make([]*models.RunResponse, 0, numRepos*runsPerRepo)

	fmt.Printf("[MOCK DEBUG] Generating runs for %d repos (max available: %d)\n", numRepos, len(repositories))

	// Generate runs for each repository
	for i := 0; i < numRepos && i < len(repositories); i++ {
		repo := repositories[i]

		// Vary the number of runs per repo for more realistic data
		var actualRunsForRepo int
		switch i {
		case 0:
			// First repo gets 30 runs
			actualRunsForRepo = 30
		case 1:
			// Second repo gets only 5 runs
			actualRunsForRepo = 5
		default:
			// All other repos get random number of runs between 1 and 60
			actualRunsForRepo = rand.Intn(60) + 1
		}

		fmt.Printf("[MOCK DEBUG] Repo[%d] %s: generating %d runs\n", i, repo, actualRunsForRepo)

		for j := 0; j < actualRunsForRepo; j++ {
			runID := fmt.Sprintf("run_%d_%d_%d", i+1, j+1, rand.Intn(10000))
			status := statusOptions[rand.Intn(len(statusOptions))]
			runType := runTypes[rand.Intn(len(runTypes))]

			// Create base run
			run := &models.RunResponse{
				ID:         runID,
				Status:     status,
				Repository: repo,
				Source:     branches[rand.Intn(len(branches))],
				Target:     fmt.Sprintf("repobird/%s-%d", runType, j+1),
				RunType:    runType,
				Title:      titles[rand.Intn(len(titles))],
				Prompt:     prompts[rand.Intn(len(prompts))],
				CreatedAt:  time.Now().Add(-time.Duration(rand.Intn(72)) * time.Hour),
				UpdatedAt:  time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour),
			}

			// Add status-specific details
			switch status {
			case models.StatusDone:
				prURL := fmt.Sprintf("https://github.com/%s/pull/%d", repo, rand.Intn(1000)+1)
				run.PrURL = &prURL
				run.Description = fmt.Sprintf("Successfully completed: %s", titles[rand.Intn(len(titles))])

			case models.StatusFailed:
				run.Error = errorMessages[rand.Intn(len(errorMessages))]

			case models.StatusProcessing, models.StatusInitializing:
				progress := rand.Intn(90) + 10
				run.Description = fmt.Sprintf("Processing... (%d%%)", progress)
				run.Plan = fmt.Sprintf("Step %d of 5: %s", rand.Intn(5)+1, prompts[rand.Intn(len(prompts))])

			case models.StatusPostProcess:
				prURL := fmt.Sprintf("https://github.com/%s/pull/%d", repo, rand.Intn(1000)+1)
				run.PrURL = &prURL
				run.Description = "Creating pull request and running tests..."

			case models.StatusQueued:
				run.Description = "Waiting in queue..."
			}

			// Add context for variety
			if rand.Float32() > 0.5 {
				run.Context = fmt.Sprintf("Additional context: %s", prompts[rand.Intn(len(prompts))])
			}

			runs = append(runs, run)
		}
	}

	fmt.Printf("[MOCK DEBUG] Total runs generated: %d\n", len(runs))

	// Debug: Count runs per repository
	runCounts := make(map[string]int)
	for _, run := range runs {
		runCounts[run.Repository]++
	}
	fmt.Printf("[MOCK DEBUG] Runs per repository:\n")
	for repo, count := range runCounts {
		fmt.Printf("  %s: %d runs\n", repo, count)
	}

	return runs
}

// MockClient wraps a real client but intercepts certain calls for mock data
type MockClient struct {
	realClient interface{}
	mockRuns   []*models.RunResponse
}

// NewMockClient creates a new mock client
func NewMockClient(realClient interface{}) *MockClient {
	return &MockClient{
		realClient: realClient,
		mockRuns:   GenerateMockRuns(30, 20), // 30 repos with 20 runs each
	}
}

// ListRuns returns mock runs data
func (m *MockClient) ListRuns(ctx context.Context, page, limit int) (*models.ListRunsResponse, error) {
	// Calculate pagination
	start := (page - 1) * limit
	end := start + limit

	if start >= len(m.mockRuns) {
		return &models.ListRunsResponse{
			Data: []*models.RunResponse{},
			Metadata: &models.PaginationMetadata{
				CurrentPage: page,
				Total:       len(m.mockRuns),
				TotalPages:  (len(m.mockRuns) + limit - 1) / limit,
			},
		}, nil
	}

	if end > len(m.mockRuns) {
		end = len(m.mockRuns)
	}

	return &models.ListRunsResponse{
		Data: m.mockRuns[start:end],
		Metadata: &models.PaginationMetadata{
			CurrentPage: page,
			Total:       len(m.mockRuns),
			TotalPages:  (len(m.mockRuns) + limit - 1) / limit,
		},
	}, nil
}

// GetUserInfoWithContext returns mock user info
func (m *MockClient) GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error) {
	return &models.UserInfo{
		Email:          "debug-user@repobird.ai",
		Name:           "Debug User",
		ID:             -1, // Use negative ID to ensure separate cache directory
		GithubUsername: "debug-user",
		RemainingRuns:  100,
		TotalRuns:      500,
		Tier:           "premium",
	}, nil
}

// GetRun returns a specific mock run
func (m *MockClient) GetRun(id string) (*models.RunResponse, error) {
	for _, run := range m.mockRuns {
		if run.ID == id {
			return run, nil
		}
	}
	// If not found in mock data, generate a new one
	return &models.RunResponse{
		ID:         id,
		Status:     models.StatusDone,
		Repository: "debug/test-repo",
		Source:     "main",
		Target:     "feature/test",
		RunType:    "run",
		Title:      "Mock Run",
		Prompt:     "This is a mock run for testing",
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		UpdatedAt:  time.Now().Add(-1 * time.Hour),
	}, nil
}

// GetUserInfo returns mock user info (without context for backward compatibility)
func (m *MockClient) GetUserInfo() (*models.UserInfo, error) {
	return m.GetUserInfoWithContext(context.Background())
}

// ListRunsLegacy returns mock runs data using the legacy method
func (m *MockClient) ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error) {
	// Calculate slice bounds
	end := offset + limit
	if offset >= len(m.mockRuns) {
		return []*models.RunResponse{}, nil
	}
	if end > len(m.mockRuns) {
		end = len(m.mockRuns)
	}
	return m.mockRuns[offset:end], nil
}

// GetAPIEndpoint returns the mock API endpoint
func (m *MockClient) GetAPIEndpoint() string {
	return "https://repobird.ai"
}

// VerifyAuth returns mock user info
func (m *MockClient) VerifyAuth() (*models.UserInfo, error) {
	return m.GetUserInfo()
}

// CreateRunAPI creates a new mock run
func (m *MockClient) CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error) {
	runID := fmt.Sprintf("run_new_%d", rand.Intn(10000))
	return &models.RunResponse{
		ID:             runID,
		Status:         models.StatusQueued,
		RepositoryName: request.RepositoryName,
		Source:         request.SourceBranch,
		Target:         request.TargetBranch,
		RunType:        string(request.RunType),
		Title:          request.Title,
		Prompt:         request.Prompt,
		Context:        request.Context,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

// ListRepositories returns mock repositories
func (m *MockClient) ListRepositories(ctx context.Context) ([]models.APIRepository, error) {
	repos := make([]models.APIRepository, len(repositories))
	for i, repo := range repositories {
		parts := strings.Split(repo, "/")
		owner := parts[0]
		name := parts[1]
		repos[i] = models.APIRepository{
			ID:            i + 1,
			Name:          repo,
			RepoName:      name,
			RepoOwner:     owner,
			RepoURL:       fmt.Sprintf("https://github.com/%s", repo),
			DefaultBranch: "main",
			IsEnabled:     true,
		}
	}
	return repos, nil
}

// GetFileHashes returns mock file hashes
func (m *MockClient) GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error) {
	// Return some mock file hashes for testing
	hashes := []models.FileHashEntry{
		{IssueRunID: 1, FileHash: "abc123def456789"},
		{IssueRunID: 2, FileHash: "xyz789ghi012345"},
		{IssueRunID: 3, FileHash: "qwe456rty789012"},
	}
	return hashes, nil
}
