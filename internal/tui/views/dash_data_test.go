// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"fmt"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateRepositoryStats_SortingByCreatedAt(t *testing.T) {
	baseTime := time.Now()

	tests := []struct {
		name         string
		repositories []models.Repository
		runs         []*models.RunResponse
		expected     []string // Expected repository names in order
	}{
		{
			name: "sorts repositories by most recent run creation",
			repositories: []models.Repository{
				{Name: "org/repo-old"},
				{Name: "org/repo-recent"},
				{Name: "org/repo-middle"},
			},
			runs: []*models.RunResponse{
				{
					ID:             "1",
					RepositoryName: "org/repo-old",
					CreatedAt:      baseTime.Add(-72 * time.Hour),
					UpdatedAt:      baseTime.Add(-72 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "2",
					RepositoryName: "org/repo-recent",
					CreatedAt:      baseTime.Add(-1 * time.Hour),
					UpdatedAt:      baseTime.Add(-1 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "3",
					RepositoryName: "org/repo-middle",
					CreatedAt:      baseTime.Add(-24 * time.Hour),
					UpdatedAt:      baseTime.Add(-24 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/repo-recent", "org/repo-middle", "org/repo-old"},
		},
		{
			name: "repositories without runs sorted alphabetically at bottom",
			repositories: []models.Repository{
				{Name: "org/zebra-no-runs"},
				{Name: "org/alpha-no-runs"},
				{Name: "org/with-runs"},
			},
			runs: []*models.RunResponse{
				{
					ID:             "1",
					RepositoryName: "org/with-runs",
					CreatedAt:      baseTime.Add(-1 * time.Hour),
					UpdatedAt:      baseTime.Add(-1 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/with-runs", "org/alpha-no-runs", "org/zebra-no-runs"},
		},
		{
			name: "multiple runs per repository uses most recent creation",
			repositories: []models.Repository{
				{Name: "org/repo-a"},
				{Name: "org/repo-b"},
			},
			runs: []*models.RunResponse{
				{
					ID:             "1",
					RepositoryName: "org/repo-a",
					CreatedAt:      baseTime.Add(-48 * time.Hour), // Old run
					UpdatedAt:      baseTime.Add(-48 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "2",
					RepositoryName: "org/repo-a",
					CreatedAt:      baseTime.Add(-1 * time.Hour), // Recent run for same repo
					UpdatedAt:      baseTime.Add(-1 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:             "3",
					RepositoryName: "org/repo-b",
					CreatedAt:      baseTime.Add(-12 * time.Hour),
					UpdatedAt:      baseTime.Add(-12 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"org/repo-a", "org/repo-b"},
		},
		{
			name: "handles repo ID mapping",
			repositories: []models.Repository{
				{Name: "owner/repo-with-id"},
				{Name: "owner/other-repo"},
			},
			runs: []*models.RunResponse{
				{
					ID:             "1",
					RepositoryName: "owner/repo-with-id",
					RepoID:         123,
					CreatedAt:      baseTime.Add(-2 * time.Hour),
					UpdatedAt:      baseTime.Add(-2 * time.Hour),
					Status:         models.StatusDone,
				},
				{
					ID:        "2",
					RepoID:    123,                             // Same repo but only has ID
					CreatedAt: baseTime.Add(-30 * time.Minute), // More recent
					UpdatedAt: baseTime.Add(-30 * time.Minute),
					Status:    models.StatusDone,
				},
				{
					ID:             "3",
					RepositoryName: "owner/other-repo",
					CreatedAt:      baseTime.Add(-1 * time.Hour),
					UpdatedAt:      baseTime.Add(-1 * time.Hour),
					Status:         models.StatusDone,
				},
			},
			expected: []string{"owner/repo-with-id", "owner/other-repo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dashboard view with mock client
			d := &DashboardView{
				cache:           cache.NewSimpleCache(),
				apiRepositories: make(map[int]models.APIRepository),
			}

			// Set up API repositories for ID mapping test
			if tt.name == "handles repo ID mapping" {
				d.apiRepositories[123] = models.APIRepository{
					ID:        123,
					Name:      "owner/repo-with-id",
					RepoOwner: "owner",
					RepoName:  "repo-with-id",
				}
			}

			// Update repository stats
			result := d.updateRepositoryStats(tt.repositories, tt.runs)

			// Extract repository names in order
			var actualNames []string
			for _, repo := range result {
				actualNames = append(actualNames, repo.Name)
			}

			// Assert the order matches expected
			assert.Equal(t, tt.expected, actualNames, "Repository order should match expected")
		})
	}
}

func TestUpdateRepositoryStats_Statistics(t *testing.T) {
	baseTime := time.Now()

	d := &DashboardView{
		cache:           cache.NewSimpleCache(),
		apiRepositories: make(map[int]models.APIRepository),
	}

	repositories := []models.Repository{
		{Name: "org/test-repo", RunCounts: models.RunStats{}},
	}

	runs := []*models.RunResponse{
		{
			ID:             "1",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-3 * time.Hour),
			UpdatedAt:      baseTime.Add(-3 * time.Hour),
			Status:         models.StatusDone,
		},
		{
			ID:             "2",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-2 * time.Hour),
			UpdatedAt:      baseTime.Add(-2 * time.Hour),
			Status:         models.StatusFailed,
		},
		{
			ID:             "3",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-1 * time.Hour),
			UpdatedAt:      baseTime.Add(-30 * time.Minute),
			Status:         models.StatusProcessing,
		},
		{
			ID:             "4",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-30 * time.Minute),
			UpdatedAt:      baseTime.Add(-30 * time.Minute),
			Status:         models.StatusQueued,
		},
		{
			ID:             "5",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-20 * time.Minute),
			UpdatedAt:      baseTime.Add(-20 * time.Minute),
			Status:         models.StatusInitializing,
		},
		{
			ID:             "6",
			RepositoryName: "org/test-repo",
			CreatedAt:      baseTime.Add(-10 * time.Minute),
			UpdatedAt:      baseTime.Add(-10 * time.Minute),
			Status:         models.StatusPostProcess,
		},
	}

	result := d.updateRepositoryStats(repositories, runs)

	require.Len(t, result, 1, "Should have one repository")
	repo := result[0]

	assert.Equal(t, "org/test-repo", repo.Name)
	assert.Equal(t, 6, repo.RunCounts.Total, "Should count all runs")
	assert.Equal(t, 1, repo.RunCounts.Completed, "Should count completed runs")
	assert.Equal(t, 1, repo.RunCounts.Failed, "Should count failed runs")
	assert.Equal(t, 4, repo.RunCounts.Running, "Should count running, queued, initializing, and post-process runs")

	// LastActivity should be the most recent UpdatedAt
	expectedLastActivity := baseTime.Add(-10 * time.Minute)
	assert.Equal(t, expectedLastActivity, repo.LastActivity, "Should track most recent activity")
}

func TestUpdateRepositoryStats_EdgeCases(t *testing.T) {
	baseTime := time.Now()

	t.Run("empty repositories list", func(t *testing.T) {
		d := &DashboardView{
			cache:           cache.NewSimpleCache(),
			apiRepositories: make(map[int]models.APIRepository),
		}

		runs := []*models.RunResponse{
			{
				ID:             "1",
				RepositoryName: "org/repo",
				CreatedAt:      baseTime,
				UpdatedAt:      baseTime,
				Status:         models.StatusDone,
			},
		}

		result := d.updateRepositoryStats([]models.Repository{}, runs)
		assert.Empty(t, result, "Should return empty list when no repositories provided")
	})

	t.Run("empty runs list", func(t *testing.T) {
		d := &DashboardView{
			cache:           cache.NewSimpleCache(),
			apiRepositories: make(map[int]models.APIRepository),
		}

		repositories := []models.Repository{
			{Name: "org/repo1"},
			{Name: "org/repo2"},
		}

		result := d.updateRepositoryStats(repositories, []*models.RunResponse{})

		assert.Len(t, result, 2, "Should return all repositories")
		// When no runs, repositories should be sorted alphabetically
		assert.Equal(t, "org/repo1", result[0].Name)
		assert.Equal(t, "org/repo2", result[1].Name)

		for _, repo := range result {
			assert.Equal(t, 0, repo.RunCounts.Total, "Should have zero run count")
		}
	})

	t.Run("runs with no matching repository", func(t *testing.T) {
		d := &DashboardView{
			cache:           cache.NewSimpleCache(),
			apiRepositories: make(map[int]models.APIRepository),
		}

		repositories := []models.Repository{
			{Name: "org/repo1"},
		}

		runs := []*models.RunResponse{
			{
				ID:             "1",
				RepositoryName: "org/different-repo",
				CreatedAt:      baseTime,
				UpdatedAt:      baseTime,
				Status:         models.StatusDone,
			},
		}

		result := d.updateRepositoryStats(repositories, runs)

		assert.Len(t, result, 1, "Should return original repository")
		assert.Equal(t, 0, result[0].RunCounts.Total, "Should have zero run count for unmatched repo")
	})

	t.Run("handles nil runs in slice", func(t *testing.T) {
		d := &DashboardView{
			cache:           cache.NewSimpleCache(),
			apiRepositories: make(map[int]models.APIRepository),
		}

		repositories := []models.Repository{
			{Name: "org/repo"},
		}

		runs := []*models.RunResponse{
			nil,
			{
				ID:             "1",
				RepositoryName: "org/repo",
				CreatedAt:      baseTime,
				UpdatedAt:      baseTime,
				Status:         models.StatusDone,
			},
			nil,
		}

		result := d.updateRepositoryStats(repositories, runs)

		assert.Len(t, result, 1)
		assert.Equal(t, 1, result[0].RunCounts.Total, "Should count only non-nil runs")
	})
}

func TestUpdateRepositoryStats_PerformanceWithManyRuns(t *testing.T) {
	baseTime := time.Now()

	d := &DashboardView{
		cache:           cache.NewSimpleCache(),
		apiRepositories: make(map[int]models.APIRepository),
	}

	// Create many repositories
	numRepos := 100
	repositories := make([]models.Repository, numRepos)
	for i := 0; i < numRepos; i++ {
		repositories[i] = models.Repository{
			Name: fmt.Sprintf("org/repo-%03d", i),
		}
	}

	// Create many runs
	var runs []*models.RunResponse
	runsPerRepo := 50
	for i := 0; i < numRepos; i++ {
		for j := 0; j < runsPerRepo; j++ {
			runs = append(runs, &models.RunResponse{
				ID:             fmt.Sprintf("%d-%d", i, j),
				RepositoryName: fmt.Sprintf("org/repo-%03d", i),
				CreatedAt:      baseTime.Add(-time.Duration(i*runsPerRepo+j) * time.Hour),
				UpdatedAt:      baseTime.Add(-time.Duration(i*runsPerRepo+j) * time.Hour),
				Status:         models.StatusDone,
			})
		}
	}

	start := time.Now()
	result := d.updateRepositoryStats(repositories, runs)
	duration := time.Since(start)

	assert.Len(t, result, numRepos, "Should return all repositories")
	assert.Less(t, duration, 100*time.Millisecond, "Should complete within reasonable time")

	// Verify first repo has the most recent run
	assert.Equal(t, "org/repo-000", result[0].Name, "Most recent repo should be first")

	// Verify each repo has correct run count
	for _, repo := range result {
		assert.Equal(t, runsPerRepo, repo.RunCounts.Total,
			"Each repository should have correct run count")
	}
}
