package models

import "time"

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
	Repository string   `json:"repository"`
	Source     string   `json:"source"`
	Target     string   `json:"target"`
	RunType    RunType  `json:"runType"`
	Title      string   `json:"title,omitempty"`
	Context    string   `json:"context,omitempty"`
	Files      []string `json:"files,omitempty"`
}

type RunResponse struct {
	ID         string    `json:"id"`
	Status     RunStatus `json:"status"`
	Repository string    `json:"repository"`
	Source     string    `json:"source"`
	Target     string    `json:"target"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	Prompt     string    `json:"prompt"`
	Title      string    `json:"title,omitempty"`
	Context    string    `json:"context,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type UserInfo struct {
	Email         string `json:"email"`
	RemainingRuns int    `json:"remainingRuns"`
	TotalRuns     int    `json:"totalRuns"`
	Tier          string `json:"tier"`
}
