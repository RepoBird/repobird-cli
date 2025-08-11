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
	BatchID  string           `json:"batchId"`
	Runs     []RunCreatedItem `json:"runs"`
	Errors   []RunError       `json:"errors,omitempty"`
	Metadata BulkMetadata     `json:"metadata"`
}

// RunCreatedItem represents a successfully created run
type RunCreatedItem struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Target   string `json:"target"`
	Status   string `json:"status"`
	URL      string `json:"url,omitempty"`
	RunURL   string `json:"runUrl,omitempty"`
	QueuedAt string `json:"queuedAt,omitempty"`
}

// RunError represents an error creating a specific run
type RunError struct {
	Index   int    `json:"index"`
	Title   string `json:"title,omitempty"`
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// BulkMetadata contains metadata about the bulk operation
type BulkMetadata struct {
	TotalRequested int       `json:"totalRequested"`
	TotalCreated   int       `json:"totalCreated"`
	TotalFailed    int       `json:"totalFailed"`
	CreatedAt      time.Time `json:"createdAt"`
}

// BulkStatusResponse represents the status of a bulk run batch
type BulkStatusResponse struct {
	BatchID    string          `json:"batchId"`
	Status     string          `json:"status"`
	Runs       []RunStatusItem `json:"runs"`
	Statistics BulkStatistics  `json:"statistics"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

// RunStatusItem represents the status of a single run in a batch
type RunStatusItem struct {
	ID          int        `json:"id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	Progress    int        `json:"progress,omitempty"`
	Message     string     `json:"message,omitempty"`
	Error       string     `json:"error,omitempty"`
	StartedAt   *time.Time `json:"startedAt,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	URL         string     `json:"url,omitempty"`
	RunURL      string     `json:"runUrl,omitempty"`
}

// BulkStatistics contains statistics about a bulk batch
type BulkStatistics struct {
	Total      int `json:"total"`
	Queued     int `json:"queued"`
	Processing int `json:"processing"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
	Cancelled  int `json:"cancelled"`
}
