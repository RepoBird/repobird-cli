// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package domain

import (
	"context"
)

// RunService defines the business logic for run operations
type RunService interface {
	CreateRun(ctx context.Context, req CreateRunRequest) (*Run, error)
	GetRun(ctx context.Context, id string) (*Run, error)
	ListRuns(ctx context.Context, opts ListOptions) ([]*Run, error)
	WaitForCompletion(ctx context.Context, id string, callback ProgressCallback) (*Run, error)
}

// RunRepository defines the data access interface for runs
type RunRepository interface {
	Create(ctx context.Context, req CreateRunRequest) (*Run, error)
	Get(ctx context.Context, id string) (*Run, error)
	List(ctx context.Context, opts ListOptions) ([]*Run, error)
}

// CacheService defines the caching interface
type CacheService interface {
	GetRun(id string) (*Run, bool)
	SetRun(id string, run *Run)
	GetRunList() ([]*Run, bool)
	SetRunList(runs []*Run)
	InvalidateRun(id string)
	Clear()
}

// GitService defines git operations interface
type GitService interface {
	GetCurrentBranch() (string, error)
	GetRepositoryName() (string, error)
	IsGitRepository() bool
}

// ProgressCallback is called with status updates during polling
type ProgressCallback func(status string, message string)
