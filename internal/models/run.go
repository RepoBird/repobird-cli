package models

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

type RunType string

const (
	RunTypeRun      RunType = "run"
	RunTypePlan     RunType = "plan"
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

// RunConfig is a unified configuration structure for both JSON and Markdown configs
type RunConfig struct {
	Prompt     string   `json:"prompt" yaml:"prompt"`
	Repository string   `json:"repository" yaml:"repository"`
	Source     string   `json:"source" yaml:"source"`
	Target     string   `json:"target" yaml:"target"`
	RunType    string   `json:"runType" yaml:"runType"`
	Title      string   `json:"title,omitempty" yaml:"title,omitempty"`
	Context    string   `json:"context,omitempty" yaml:"context,omitempty"`
	Files      []string `json:"files,omitempty" yaml:"files,omitempty"`
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
	FileHash       string   `json:"fileHash,omitempty"`
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
	ID             string    `json:"id"` // Now stored as string internally
	Status         RunStatus `json:"status"`
	Repository     string    `json:"repository,omitempty"`     // Legacy field
	RepositoryName string    `json:"repositoryName,omitempty"` // New API field
	RepoID         int       `json:"repoId,omitempty"`
	Source         string    `json:"source"`
	Target         string    `json:"target"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	Prompt         string    `json:"prompt"`
	Title          string    `json:"title,omitempty"`
	Description    string    `json:"description,omitempty"`
	Context        string    `json:"context,omitempty"`
	Error          string    `json:"error,omitempty"`
	PrURL          *string   `json:"prUrl,omitempty"`
	TriggerSource  *string   `json:"triggerSource,omitempty"`
	RunType        string    `json:"runType,omitempty"`
	Plan           string    `json:"plan,omitempty"`
	FileHash       string    `json:"fileHash,omitempty"`
}

// GetIDString returns the ID as a string
func (r *RunResponse) GetIDString() string {
	if r.ID == "" || r.ID == "null" {
		return ""
	}
	return r.ID
}

// GetRepositoryName returns the repository name from either field
func (r *RunResponse) GetRepositoryName() string {
	if r.RepositoryName != "" {
		return r.RepositoryName
	}
	return r.Repository
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
	ID             int    `json:"id,omitempty"`
	Email          string `json:"email"`
	Name           string `json:"name,omitempty"`
	GithubUsername string `json:"githubUsername,omitempty"`
	RemainingRuns  int    `json:"remainingRuns"`
	TotalRuns      int    `json:"totalRuns"`
	Tier           string `json:"tier"`
	TierDetails    *Tier  `json:"tierDetails,omitempty"`
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

// FileHashEntry represents a file hash entry from the API
type FileHashEntry struct {
	IssueRunID int    `json:"issueRunId"`
	FileHash   string `json:"fileHash"`
}

// FileHashesResponse represents the response from /api/v1/runs/hashes
type FileHashesResponse struct {
	Data []FileHashEntry `json:"data"`
}

// LoadRunConfigFromFile loads a RunConfig from a JSON file
func LoadRunConfigFromFile(filepath string) (*RunConfig, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var runReq RunRequest
	if err := json.NewDecoder(file).Decode(&runReq); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Convert RunRequest to RunConfig
	return &RunConfig{
		Prompt:     runReq.Prompt,
		Repository: runReq.Repository,
		Source:     runReq.Source,
		Target:     runReq.Target,
		RunType:    string(runReq.RunType),
		Title:      runReq.Title,
		Context:    runReq.Context,
		Files:      runReq.Files,
	}, nil
}
