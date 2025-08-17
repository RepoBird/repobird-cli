// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package services

import (
	"context"
	"fmt"
	"time"

	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/internal/utils"
)

// runService implements the domain.RunService interface
type runService struct {
	repo  domain.RunRepository
	cache domain.CacheService
	git   domain.GitService
}

// NewRunService creates a new instance of RunService
func NewRunService(
	repo domain.RunRepository,
	cache domain.CacheService,
	git domain.GitService,
) domain.RunService {
	return &runService{
		repo:  repo,
		cache: cache,
		git:   git,
	}
}

// CreateRun creates a new run with validation and auto-detection
func (s *runService) CreateRun(ctx context.Context, req domain.CreateRunRequest) (*domain.Run, error) {
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Auto-detection disabled - experimental feature
	// TODO: Enable when feature is ready and properly tested
	// if req.RepositoryName == "" && s.git.IsGitRepository() {
	// 	repoName, err := s.git.GetRepositoryName()
	// 	if err == nil {
	// 		req.RepositoryName = repoName
	// 	}
	// }

	// if req.SourceBranch == "" && s.git.IsGitRepository() {
	// 	branch, err := s.git.GetCurrentBranch()
	// 	if err == nil {
	// 		req.SourceBranch = branch
	// 	}
	// }

	// Create run via repository
	run, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}

	// Update cache
	s.cache.SetRun(run.ID, run)
	s.cache.InvalidateRun("list") // Invalidate list cache

	return run, nil
}

// GetRun retrieves a run by ID
func (s *runService) GetRun(ctx context.Context, id string) (*domain.Run, error) {
	// Check cache first
	if run, found := s.cache.GetRun(id); found {
		// If run is not terminal, refresh from repository
		if !run.IsTerminal() {
			freshRun, err := s.repo.Get(ctx, id)
			if err == nil {
				s.cache.SetRun(id, freshRun)
				return freshRun, nil
			}
		}
		return run, nil
	}

	// Fetch from repository
	run, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	// Update cache
	s.cache.SetRun(id, run)

	return run, nil
}

// ListRuns retrieves a list of runs
func (s *runService) ListRuns(ctx context.Context, opts domain.ListOptions) ([]*domain.Run, error) {
	// Check cache first
	if runs, found := s.cache.GetRunList(); found && len(runs) > 0 {
		return runs, nil
	}

	// Fetch from repository
	runs, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}

	// Update cache
	s.cache.SetRunList(runs)

	return runs, nil
}

// WaitForCompletion polls a run until it reaches a terminal state
func (s *runService) WaitForCompletion(ctx context.Context, id string, callback domain.ProgressCallback) (*domain.Run, error) {
	poller := utils.NewGenericPoller[*domain.Run](&utils.GenericPollConfig{
		Interval:      5 * time.Second,
		MaxInterval:   30 * time.Second,
		BackoffFactor: 1.5,
		Timeout:       45 * time.Minute,
	})

	pollFunc := func(ctx context.Context) (*domain.Run, error) {
		return s.GetRun(ctx, id)
	}

	onUpdate := func(run *domain.Run) {
		if callback != nil {
			callback(run.Status, run.StatusMessage)
		}
	}

	// Use generic polling
	result, err := poller.PollUntilComplete(
		ctx,
		pollFunc,
		func(run *domain.Run) bool { return run.IsTerminal() },
		onUpdate,
	)

	if err != nil {
		return nil, fmt.Errorf("polling failed: %w", err)
	}

	return result, nil
}

// validateCreateRequest validates a CreateRunRequest
func (s *runService) validateCreateRequest(req domain.CreateRunRequest) error {
	if req.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if req.RunType != "" && req.RunType != domain.RunTypeRun && req.RunType != domain.RunTypePlan {
		return fmt.Errorf("invalid run type: %s", req.RunType)
	}

	return nil
}
