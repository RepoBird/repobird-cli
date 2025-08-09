package models

import (
	"fmt"
	"time"
)

type RunType string

const (
	RunTypeRun      RunType = "run"
	RunTypeApproval RunType = "approval"
)

type RunStatus string

const (
	StatusQueued       RunStatus = "QUEUED"
	StatusInitializing RunStatus = "INITIALIZING"
	StatusProcessing   RunStatus = "PROCESSING"
	StatusPostProcess  RunStatus = "POST_PROCESS"
	StatusDone         RunStatus = "DONE"
	StatusFailed       RunStatus = "FAILED"
)

type RunRequest struct {
	Prompt     string   `json:"prompt"`
	Repository string   `json:"repository"` // User-facing field name
	Source     string   `json:"source"`     // User-facing field name
	Target     string   `json:"target"`     // User-facing field name
	RunType    RunType  `json:"runType"`
	Title      string   `json:"title,omitempty"`
	Context    string   `json:"context,omitempty"`
	Files      []string `json:"files,omitempty"`
}

// APIRunRequest is the structure that matches the actual API expectations
type APIRunRequest struct {
	Prompt         string   `json:"prompt"`
	RepositoryName string   `json:"repositoryName"`
	SourceBranch   string   `json:"sourceBranch"`
	TargetBranch   string   `json:"targetBranch"`
	RunType        RunType  `json:"runType"`
	Title          string   `json:"title,omitempty"`
	Context        string   `json:"context,omitempty"`
	Files          []string `json:"files,omitempty"`
}

// ToAPIRequest converts user-facing RunRequest to API-compatible structure
func (r *RunRequest) ToAPIRequest() *APIRunRequest {
	return &APIRunRequest{
		Prompt:         r.Prompt,
		RepositoryName: r.Repository,
		SourceBranch:   r.Source,
		TargetBranch:   r.Target,
		RunType:        r.RunType,
		Title:          r.Title,
		Context:        r.Context,
		Files:          r.Files,
	}
}

type RunResponse struct {
	ID          interface{} `json:"id"` // Can be string or int from API
	Status      RunStatus   `json:"status"`
	Repository  string      `json:"repository"`
	RepoId      int         `json:"repoId,omitempty"`
	Source      string      `json:"source"`
	Target      string      `json:"target"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	Prompt      string      `json:"prompt"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Context     string      `json:"context,omitempty"`
	Error       string      `json:"error,omitempty"`
	PrUrl       *string     `json:"prUrl,omitempty"`
	RunType     string      `json:"runType,omitempty"`
}

// GetIDString returns the ID as a string regardless of its actual type
func (r *RunResponse) GetIDString() string {
	if r.ID == nil {
		return ""
	}
	switch v := r.ID.(type) {
	case string:
		if v == "null" {
			return ""
		}
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	case int:
		return fmt.Sprintf("%d", v)
	default:
		s := fmt.Sprintf("%v", v)
		if s == "<nil>" || s == "null" {
			return ""
		}
		return s
	}
}

type UserInfo struct {
	Email         string `json:"email"`
	RemainingRuns int    `json:"remainingRuns"`
	TotalRuns     int    `json:"totalRuns"`
	Tier          string `json:"tier"`
}

type ListRunsResponse struct {
	Data     []*RunResponse      `json:"data"`
	Metadata *PaginationMetadata `json:"metadata"`
}

type SingleRunResponse struct {
	Data     *RunResponse        `json:"data"`
	Metadata *PaginationMetadata `json:"metadata"`
}

type PaginationMetadata struct {
	CurrentPage int `json:"currentPage"`
	Total       int `json:"total"`
	TotalPages  int `json:"totalPages"`
}
