package dto

import "time"

// BulkRunRequest represents a request to create multiple runs
type BulkRunRequest struct {
	RepositoryName string      `json:"repositoryName,omitempty"`
	RepoID         int         `json:"repoId,omitempty"`
	RunType        string      `json:"runType"`
	SourceBranch   string      `json:"sourceBranch,omitempty"`
	BatchTitle     string      `json:"batchTitle,omitempty"`
	Force          bool        `json:"force,omitempty"`
	Runs           []RunItem   `json:"runs"`
	Options        BulkOptions `json:"options,omitempty"`
}

// RunItem represents a single run within a bulk request
type RunItem struct {
	Prompt   string `json:"prompt"`
	Title    string `json:"title,omitempty"`
	Target   string `json:"target,omitempty"`
	Context  string `json:"context,omitempty"`
	FileHash string `json:"fileHash,omitempty"`
}

// BulkOptions represents options for bulk run execution
type BulkOptions struct {
	Parallel      int  `json:"parallel,omitempty"`
	StopOnFailure bool `json:"stopOnFailure,omitempty"`
}

// BulkRunResponse represents the response after creating bulk runs
type BulkRunResponse struct {
	Data       BulkRunData `json:"data"`
	StatusCode int         `json:"-"` // HTTP status code from the response (not part of JSON)
}

// BulkRunData contains the actual bulk run response data
type BulkRunData struct {
	BatchID    string               `json:"batchId"`
	BatchTitle string               `json:"batchTitle,omitempty"`
	Successful []RunCreatedItem     `json:"successful"`
	Failed     []RunError           `json:"failed,omitempty"`
	Metadata   BulkResponseMetadata `json:"metadata"`
}

// RunCreatedItem represents a successfully created run
type RunCreatedItem struct {
	ID             int    `json:"id"`
	Status         string `json:"status"`
	RepositoryName string `json:"repositoryName"`
	Title          string `json:"title"`
	RequestIndex   int    `json:"requestIndex"`
}

// RunError represents an error creating a specific run
type RunError struct {
	RequestIndex  int    `json:"requestIndex"`
	Prompt        string `json:"prompt"`
	Error         string `json:"error"`
	Message       string `json:"message"`
	ExistingRunId int    `json:"existingRunId,omitempty"`
}

// BulkResponseMetadata contains metadata about the bulk operation response
type BulkResponseMetadata struct {
	TotalRequested  int `json:"totalRequested"`
	TotalSuccessful int `json:"totalSuccessful"`
	TotalFailed     int `json:"totalFailed"`
}

// BulkMetadata contains metadata about the bulk operation (keeping for backwards compatibility)
type BulkMetadata struct {
	TotalRequested int       `json:"totalRequested"`
	TotalCreated   int       `json:"totalCreated"`
	TotalFailed    int       `json:"totalFailed"`
	CreatedAt      time.Time `json:"createdAt"`
}

// BulkStatusResponse represents the API response for bulk status
type BulkStatusResponse struct {
	Data BulkStatusData `json:"data"`
}

// BulkStatusData contains the actual bulk status data
type BulkStatusData struct {
	BatchID    string             `json:"batchId"`
	BatchTitle *string            `json:"batchTitle,omitempty"`
	Status     string             `json:"status"`
	Runs       []RunStatusItem    `json:"runs"`
	Metadata   BulkStatusMetadata `json:"metadata"`
}

// RunStatusItem represents the status of a single run in a batch
type RunStatusItem struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	Progress    int     `json:"progress,omitempty"`
	CompletedAt *string `json:"completedAt,omitempty"`
	PRURL       *string `json:"prUrl,omitempty"`
}

// BulkStatusMetadata contains metadata about the bulk batch status
type BulkStatusMetadata struct {
	TotalRuns               int     `json:"totalRuns"`
	Completed               int     `json:"completed"`
	Processing              int     `json:"processing"`
	Queued                  int     `json:"queued"`
	Failed                  int     `json:"failed"`
	StartedAt               string  `json:"startedAt"`
	EstimatedCompletionTime *string `json:"estimatedCompletionTime,omitempty"`
}

// BulkStatistics contains statistics about a bulk batch (deprecated, kept for compatibility)
type BulkStatistics struct {
	Total      int `json:"total"`
	Queued     int `json:"queued"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Cancelled  int `json:"cancelled"`
}
