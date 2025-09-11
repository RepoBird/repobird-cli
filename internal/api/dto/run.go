package dto

import (
	"encoding/json"
	"fmt"
	"time"
)

// RunID handles the API's flexible ID type (can be string or int)
type RunID struct {
	value interface{}
}

// UnmarshalJSON implements custom unmarshaling for RunID
func (r *RunID) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		r.value = s
		return nil
	}

	// Try to unmarshal as int
	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		r.value = i
		return nil
	}

	return fmt.Errorf("RunID must be string or int")
}

// String returns the string representation of the RunID
func (r RunID) String() string {
	switch v := r.value.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	default:
		return ""
	}
}

// CreateRunRequest represents the API request to create a run
type CreateRunRequest struct {
	Prompt         string   `json:"prompt"`
	RepositoryName string   `json:"repositoryName"`
	SourceBranch   string   `json:"sourceBranch"`
	TargetBranch   string   `json:"targetBranch"`
	RunType        string   `json:"runType"`
	Title          string   `json:"title,omitempty"`
	Context        string   `json:"context,omitempty"`
	Files          []string `json:"files,omitempty"`
}

// RunResponse represents the API response for a run
type RunResponse struct {
	ID             RunID      `json:"id"`
	Status         string     `json:"status"`
	StatusMessage  string     `json:"statusMessage,omitempty"`
	Prompt         string     `json:"prompt"`
	RepositoryName string     `json:"repositoryName"`
	SourceBranch   string     `json:"sourceBranch"`
	TargetBranch   string     `json:"targetBranch"`
	PullRequestURL string     `json:"prUrl,omitempty"`
	RunType        string     `json:"runType"`
	Title          string     `json:"title,omitempty"`
	Context        string     `json:"context,omitempty"`
	Files          []string   `json:"files,omitempty"`
	UserID         int        `json:"userId"`
	RepositoryID   int        `json:"repositoryId"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	Cost           float64    `json:"cost,omitempty"`
	InputTokens    int        `json:"inputTokens,omitempty"`
	OutputTokens   int        `json:"outputTokens,omitempty"`
	FileCount      int        `json:"fileCount,omitempty"`
	FilesChanged   []string   `json:"filesChanged,omitempty"`
	Summary        string     `json:"summary,omitempty"`
	Error          string     `json:"error,omitempty"`
}

// CreateRunResponse represents the wrapped API response for create operations
type CreateRunResponse struct {
	Data struct {
		ID      RunID  `json:"id"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"data"`
}

// SingleRunResponse represents the wrapped API response for get operations
type SingleRunResponse struct {
	Data *RunResponse `json:"data"`
}

// ListRunsResponse represents the paginated API response for list operations
type ListRunsResponse struct {
	Data    []*RunResponse `json:"data"`
	Total   int            `json:"total"`
	Limit   int            `json:"limit"`
	Offset  int            `json:"offset"`
	HasMore bool           `json:"hasMore"`
}
