// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package domain

import (
	"time"
)

// Run represents a RepoBird run in the domain layer
type Run struct {
	ID             string
	Status         string
	StatusMessage  string
	Prompt         string
	RepositoryName string
	SourceBranch   string
	TargetBranch   string
	PullRequestURL string
	RunType        string
	Title          string
	Context        string
	Files          []string
	UserID         int
	RepositoryID   int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
	Cost           float64
	InputTokens    int
	OutputTokens   int
	FileCount      int
	FilesChanged   []string
	Summary        string
	Error          string
}

// CreateRunRequest represents a request to create a new run
type CreateRunRequest struct {
	Prompt         string
	RepositoryName string
	SourceBranch   string
	TargetBranch   string
	RunType        string
	Title          string
	Context        string
	Files          []string
}

// ListOptions represents options for listing runs
type ListOptions struct {
	Limit  int
	Offset int
	UserID int
}

// RunStatus constants
const (
	StatusCreated   = "created"
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// RunType constants
const (
	RunTypeRun  = "run"
	RunTypePlan = "plan"
)

// IsTerminal returns true if the run status is terminal
func (r *Run) IsTerminal() bool {
	switch r.Status {
	case StatusCompleted, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

// IsSuccess returns true if the run completed successfully
func (r *Run) IsSuccess() bool {
	return r.Status == StatusCompleted
}
