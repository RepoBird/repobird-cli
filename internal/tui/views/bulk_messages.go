package views

import (
	"github.com/repobird/repobird-cli/internal/api/dto"
)

// Message types for bulk view operations

// BulkRunResult represents the result of a bulk run submission
type BulkRunResult struct {
	ID     int
	Title  string
	Status string
	Error  string
	URL    string
}

// fileSelectedMsg is sent when files are selected for bulk processing
type fileSelectedMsg struct {
	files []string
}

// bulkRunsLoadedMsg is sent when bulk runs are loaded from configuration files
type bulkRunsLoadedMsg struct {
	runs       []BulkRunItem
	repository string
	repoID     int
	source     string
	runType    string
	batchTitle string
}

// bulkSubmittedMsg is sent when bulk runs have been submitted
type bulkSubmittedMsg struct {
	batchID string
	results []BulkRunResult
	err     error
}

// bulkProgressMsg is sent with progress updates for bulk operations
type bulkProgressMsg struct {
	batchID    string
	statistics dto.BulkStatistics
	runs       []dto.RunStatusItem
	completed  bool
}

// bulkCancelledMsg is sent when a bulk operation is cancelled
type bulkCancelledMsg struct{}

// errMsg is sent when an error occurs
type errMsg struct {
	err error
}
