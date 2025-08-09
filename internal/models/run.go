package models

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	ID          string    `json:"id"` // Now stored as string internally
	Status      RunStatus `json:"status"`
	Repository  string    `json:"repository"`
	RepoID      int       `json:"repoId,omitempty"`
	Source      string    `json:"source"`
	Target      string    `json:"target"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	Prompt      string    `json:"prompt"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Context     string    `json:"context,omitempty"`
	Error       string    `json:"error,omitempty"`
	PrURL       *string   `json:"prUrl,omitempty"`
	RunType     string    `json:"runType,omitempty"`
}

// GetIDString returns the ID as a string
func (r *RunResponse) GetIDString() string {
	if r.ID == "" || r.ID == "null" {
		return ""
	}
	return r.ID
}

// UnmarshalJSON custom unmarshaler to handle ID field that can be string or number
func (r *RunResponse) UnmarshalJSON(data []byte) error {
	type Alias RunResponse
	aux := &struct {
		ID interface{} `json:"id"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// Convert ID to string regardless of its type
	if aux.ID != nil {
		switch v := aux.ID.(type) {
		case string:
			r.ID = v
		case float64:
			r.ID = strconv.FormatFloat(v, 'f', 0, 64)
		case int:
			r.ID = strconv.Itoa(v)
		default:
			r.ID = fmt.Sprintf("%v", v)
		}
	}

	return nil
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
