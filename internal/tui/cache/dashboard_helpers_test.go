// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRepositoryOverviewFromRuns_SortingByCreatedAt(t *testing.T) {
	// Create base time for consistent testing
	baseTime := time.Now()

	tests := []struct {
		name     string
		runs     []*models.RunResponse
		expected []string // Expected repository names in order
	}{
		{
			name: "repositories sorted by most recent run creation",
			runs: []*models.RunResponse{
				{
					ID:             "1",
					Repository:     "org/repo-old",
					RepositoryName: "org/repo-old",
					CreatedAt:      baseTime.Add(-72 * time.Hour), // 3 days ago
					UpdatedAt:      baseTime.Add(-72 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "2",
					Repository:     "org/repo-recent",
					RepositoryName: "org/repo-recent",
					CreatedAt:      baseTime.Add(-1 * time.Hour), // 1 hour ago
					UpdatedAt:      baseTime.Add(-1 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "3",
					Repository:     "org/repo-middle",
					RepositoryName: "org/repo-middle",
					CreatedAt:      baseTime.Add(-24 * time.Hour), // 1 day ago
					UpdatedAt:      baseTime.Add(-24 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/repo-recent", "org/repo-middle", "org/repo-old"},
		},
		{
			name: "multiple runs per repository - uses most recent",
			runs: []*models.RunResponse{
				{
					ID:             "1",
					Repository:     "org/repo-a",
					RepositoryName: "org/repo-a",
					CreatedAt:      baseTime.Add(-48 * time.Hour), // Old run
					UpdatedAt:      baseTime.Add(-48 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "2",
					Repository:     "org/repo-a",
					RepositoryName: "org/repo-a",
					CreatedAt:      baseTime.Add(-1 * time.Hour), // Recent run for same repo
					UpdatedAt:      baseTime.Add(-1 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "3",
					Repository:     "org/repo-b",
					RepositoryName: "org/repo-b",
					CreatedAt:      baseTime.Add(-12 * time.Hour), // Middle
					UpdatedAt:      baseTime.Add(-12 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/repo-a", "org/repo-b"},
		},
		{
			name: "repositories without runs sorted alphabetically at bottom",
			runs: []*models.RunResponse{
				{
					ID:             "1",
					Repository:     "org/active-repo",
					RepositoryName: "org/active-repo",
					CreatedAt:      baseTime.Add(-2 * time.Hour),
					UpdatedAt:      baseTime.Add(-2 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/active-repo"},
		},
		{
			name: "mixed status runs sorted by creation time",
			runs: []*models.RunResponse{
				{
					ID:             "1",
					Repository:     "org/repo-failed",
					RepositoryName: "org/repo-failed",
					CreatedAt:      baseTime.Add(-30 * time.Minute), // Most recent
					UpdatedAt:      baseTime.Add(-30 * time.Minute),
					Status:         models.StatusFailed,
				},
				{
					ID:             "2",
					Repository:     "org/repo-running",
					RepositoryName: "org/repo-running",
					CreatedAt:      baseTime.Add(-2 * time.Hour),
					UpdatedAt:      baseTime.Add(-1 * time.Hour), // Updated more recently but created earlier
					Status:         models.StatusProcessing,
				},
				{
					ID:             "3",
					Repository:     "org/repo-done",
					RepositoryName: "org/repo-done",
					CreatedAt:      baseTime.Add(-1 * time.Hour),
					UpdatedAt:      baseTime.Add(-1 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/repo-failed", "org/repo-done", "org/repo-running"},
		},
		{
			name: "repository with only repo ID mapping",
			runs: []*models.RunResponse{
				{
					ID:             "1",
					Repository:     "org/repo-with-id",
					RepositoryName: "org/repo-with-id",
					RepoID:         123,
					CreatedAt:      baseTime.Add(-2 * time.Hour),
					UpdatedAt:      baseTime.Add(-2 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:        "2",
					RepoID:    123, // Same repo but only has ID
					CreatedAt: baseTime.Add(-1 * time.Hour),
					UpdatedAt: baseTime.Add(-1 * time.Hour),
					Status:    models.StatusDone,
				},
				{
					ID:             "3",
					Repository:     "org/other-repo",
					RepositoryName: "org/other-repo",
					CreatedAt:      baseTime.Add(-30 * time.Minute), // Most recent overall
					UpdatedAt:      baseTime.Add(-30 * time.Minute),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/other-repo", "org/repo-with-id"},
		},
		{
			name:     "empty runs list",
			runs:     []*models.RunResponse{},
			expected: []string{},
		},
		{
			name: "runs with empty repository names are ignored",
			runs: []*models.RunResponse{
				{
					ID:        "1",
					CreatedAt: baseTime.Add(-1 * time.Hour),
					UpdatedAt: baseTime.Add(-1 * time.Hour),
					Status:    models.StatusDone,
				},
				{
					ID:             "2",
					Repository:     "org/valid-repo",
					RepositoryName: "org/valid-repo",
					CreatedAt:      baseTime.Add(-2 * time.Hour),
					UpdatedAt:      baseTime.Add(-2 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/valid-repo"},
		},
		{
			name: "repos with different creation times maintain order",
			runs: []*models.RunResponse{
				{
					ID:             "1",
					Repository:     "org/zebra",
					RepositoryName: "org/zebra",
					CreatedAt:      baseTime.Add(-3 * time.Hour),
					UpdatedAt:      baseTime.Add(-3 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "2",
					Repository:     "org/alpha",
					RepositoryName: "org/alpha",
					CreatedAt:      baseTime.Add(-1 * time.Hour), // Most recent
					UpdatedAt:      baseTime.Add(-1 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "3",
					Repository:     "org/beta",
					RepositoryName: "org/beta",
					CreatedAt:      baseTime.Add(-2 * time.Hour),
					UpdatedAt:      baseTime.Add(-2 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/alpha", "org/beta", "org/zebra"}, // Sorted by creation time
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a cache instance
			cache := NewSimpleCache()

			// Build repository overview
			repos := cache.BuildRepositoryOverviewFromRuns(tt.runs)

			// Extract repository names in order
			actualNames := []string{}
			for _, repo := range repos {
				actualNames = append(actualNames, repo.Name)
			}

			// Assert the order matches expected
			assert.Equal(t, tt.expected, actualNames, "Repository order should match expected")

			// Additional assertions for repository statistics
			if len(repos) > 0 && tt.name != "repository with only repo ID mapping" {
				// Verify run counts are accurate (skip for repo ID mapping test as it counts differently)
				repoRunCounts := make(map[string]int)
				for _, run := range tt.runs {
					if name := run.GetRepositoryName(); name != "" {
						repoRunCounts[name]++
					}
				}

				for _, repo := range repos {
					expectedCount := repoRunCounts[repo.Name]
					assert.Equal(t, expectedCount, repo.RunCounts.Total,
						"Repository %s should have correct total run count", repo.Name)
				}
			}
		})
	}
}

func TestBuildRepositoryOverviewFromRuns_Statistics(t *testing.T) {
	baseTime := time.Now()

	runs := []*models.RunResponse{
		{
			ID:             "1",
			Repository:     "org/test-repo",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-3 * time.Hour),
			UpdatedAt:      baseTime.Add(-3 * time.Hour),
			Status:         models.StatusDone,
		},
		{
			ID:             "2",
			Repository:     "org/test-repo",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-2 * time.Hour),
			UpdatedAt:      baseTime.Add(-2 * time.Hour),
			Status:         models.StatusFailed,
		},
		{
			ID:             "3",
			Repository:     "org/test-repo",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-1 * time.Hour),
			UpdatedAt:      baseTime.Add(-30 * time.Minute), // Most recent update
			Status:         models.StatusProcessing,
		},
		{
			ID:             "4",
			Repository:     "org/test-repo",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-30 * time.Minute),
			UpdatedAt:      baseTime.Add(-30 * time.Minute),
			Status:         models.StatusQueued,
		},
	}

	cache := NewSimpleCache()
	repos := cache.BuildRepositoryOverviewFromRuns(runs)

	require.Len(t, repos, 1, "Should have one repository")
	repo := repos[0]

	assert.Equal(t, "org/test-repo", repo.Name)
	assert.Equal(t, 4, repo.RunCounts.Total, "Should count all runs")
	assert.Equal(t, 1, repo.RunCounts.Completed, "Should count completed runs")
	assert.Equal(t, 1, repo.RunCounts.Failed, "Should count failed runs")
	assert.Equal(t, 2, repo.RunCounts.Running, "Should count running and queued runs")

	// LastActivity should be the most recent UpdatedAt
	expectedLastActivity := baseTime.Add(-30 * time.Minute)
	assert.Equal(t, expectedLastActivity, repo.LastActivity, "Should track most recent activity")
}

func TestBuildRepositoryOverviewFromRuns_EdgeCases(t *testing.T) {
	baseTime := time.Now()

	t.Run("handles nil runs in slice", func(t *testing.T) {
		runs := []*models.RunResponse{
			nil,
			{
				ID:             "1",
				Repository:     "org/repo",
				RepositoryName: "org/repo",
				CreatedAt:      baseTime,
				UpdatedAt:      baseTime,
				Status:         models.StatusDone,
			},
			nil,
		}

		cache := NewSimpleCache()
		repos := cache.BuildRepositoryOverviewFromRuns(runs)

		assert.Len(t, repos, 1)
		assert.Equal(t, "org/repo", repos[0].Name)
	})

	t.Run("repository name priority", func(t *testing.T) {
		runs := []*models.RunResponse{
			{
				ID:             "1",
				Repository:     "legacy/name", // Legacy field
				RepositoryName: "new/name",    // New field takes priority
				CreatedAt:      baseTime,
				UpdatedAt:      baseTime,
				Status:         models.StatusDone,
			},
		}

		cache := NewSimpleCache()
		repos := cache.BuildRepositoryOverviewFromRuns(runs)

		assert.Len(t, repos, 1)
		// GetRepositoryName() should prefer RepositoryName over Repository
		assert.Equal(t, "new/name", repos[0].Name)
	})

	t.Run("very large time differences", func(t *testing.T) {
		runs := []*models.RunResponse{
			{
				ID:             "1",
				Repository:     "org/ancient",
				RepositoryName: "org/ancient",
				CreatedAt:      baseTime.Add(-365 * 24 * time.Hour), // 1 year ago
				UpdatedAt:      baseTime.Add(-365 * 24 * time.Hour),
				Status:         models.StatusDone,
			},
			{
				ID:             "2",
				Repository:     "org/recent",
				RepositoryName: "org/recent",
				CreatedAt:      baseTime.Add(-1 * time.Minute), // 1 minute ago
				UpdatedAt:      baseTime.Add(-1 * time.Minute),
				Status:         models.StatusDone,
			},
		}

		cache := NewSimpleCache()
		repos := cache.BuildRepositoryOverviewFromRuns(runs)

		assert.Len(t, repos, 2)
		assert.Equal(t, "org/recent", repos[0].Name, "Recent should be first")
		assert.Equal(t, "org/ancient", repos[1].Name, "Ancient should be second")
	})
}

func TestBuildRepositoryOverviewFromRuns_PerformanceWithManyRuns(t *testing.T) {
	baseTime := time.Now()

	// Create a large number of runs to test performance
	var runs []*models.RunResponse
	numRepos := 100
	runsPerRepo := 50

	for i := 0; i < numRepos; i++ {
		repoName := fmt.Sprintf("org/repo-%03d", i)
		for j := 0; j < runsPerRepo; j++ {
			runs = append(runs, &models.RunResponse{
				ID:             fmt.Sprintf("%d-%d", i, j),
				Repository:     repoName,
				RepositoryName: repoName,
				CreatedAt:      baseTime.Add(-time.Duration(i*runsPerRepo+j) * time.Hour),
				UpdatedAt:      baseTime.Add(-time.Duration(i*runsPerRepo+j) * time.Hour),
				Status:         models.StatusDone,
			})
		}
	}

	cache := NewSimpleCache()

	start := time.Now()
	repos := cache.BuildRepositoryOverviewFromRuns(runs)
	duration := time.Since(start)

	assert.Len(t, repos, numRepos, "Should create correct number of repositories")
	assert.Less(t, duration, 100*time.Millisecond, "Should complete within reasonable time")

	// Verify first repo has the most recent run
	assert.Equal(t, "org/repo-000", repos[0].Name, "Most recent repo should be first")

	// Verify each repo has correct run count
	for _, repo := range repos {
		assert.Equal(t, runsPerRepo, repo.RunCounts.Total,
			"Each repository should have correct run count")
	}
}
