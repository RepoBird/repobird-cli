package views

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/bulk"
	"github.com/repobird/repobird-cli/internal/cache"
)

// Command functions for bulk view operations

// loadFiles loads bulk configuration from selected files
func (v *BulkView) loadFiles(files []string) (tea.Model, tea.Cmd) {
	return v, func() tea.Msg {
		// Load bulk configuration from files
		bulkConfig, err := bulk.LoadBulkConfig(files)
		if err != nil {
			return errMsg{err}
		}

		// Convert to BulkRunItems
		var runs []BulkRunItem
		for _, run := range bulkConfig.Runs {
			runs = append(runs, BulkRunItem{
				Prompt:   run.Prompt,
				Title:    run.Title,
				Target:   run.Target,
				Context:  run.Context,
				Selected: true,
				Status:   StatusPending,
			})
		}

		return bulkRunsLoadedMsg{
			runs:       runs,
			repository: bulkConfig.Repository,
			repoID:     bulkConfig.RepoID,
			source:     bulkConfig.Source,
			runType:    bulkConfig.RunType,
			batchTitle: bulkConfig.BatchTitle,
		}
	}
}

// submitBulkRuns submits selected runs to the API
func (v *BulkView) submitBulkRuns() tea.Cmd {
	v.submitting = true

	return func() tea.Msg {
		// Filter selected runs
		var selectedRuns []BulkRunItem
		for _, run := range v.runs {
			if run.Selected {
				selectedRuns = append(selectedRuns, run)
			}
		}

		if len(selectedRuns) == 0 {
			return errMsg{fmt.Errorf("no runs selected")}
		}

		// Generate file hashes
		var runItems []dto.RunItem

		for i, run := range selectedRuns {
			// Create hash from run content
			hashContent := fmt.Sprintf("%s-%s-%s-%s-%d",
				v.repository,
				run.Prompt,
				run.Target,
				run.Context,
				i,
			)
			hash := cache.CalculateStringHash(hashContent)
			run.FileHash = hash

			runItems = append(runItems, dto.RunItem{
				Prompt:   run.Prompt,
				Title:    run.Title,
				Target:   run.Target,
				Context:  run.Context,
				FileHash: hash,
			})
		}

		// Create bulk request
		req := &dto.BulkRunRequest{
			RepositoryName: v.repository,
			RepoID:         v.repoID,
			RunType:        v.runType,
			SourceBranch:   v.sourceBranch,
			BatchTitle:     v.batchTitle,
			Force:          v.force,
			Runs:           runItems,
			Options: dto.BulkOptions{
				Parallel: 5,
			},
		}

		// Submit to API
		ctx := context.Background()
		resp, err := v.client.CreateBulkRuns(ctx, req)
		if err != nil {
			return bulkSubmittedMsg{err: err}
		}

		// Convert response to results
		var results []BulkRunResult
		for _, run := range resp.Data.Successful {
			results = append(results, BulkRunResult{
				ID:     run.ID,
				Title:  run.Title,
				Status: run.Status,
				URL:    "",
			})
		}

		for _, runErr := range resp.Data.Failed {
			results = append(results, BulkRunResult{
				Title:  runErr.Prompt,
				Status: "failed",
				Error:  runErr.Message,
			})
		}

		return bulkSubmittedMsg{
			batchID: resp.Data.BatchID,
			results: results,
			err:     nil,
		}
	}
}

// pollProgress polls for bulk operation progress updates
func (v *BulkView) pollProgress() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		statusChan, err := v.client.PollBulkStatus(ctx, v.batchID, 2*time.Second)
		if err != nil {
			return errMsg{err}
		}

		// Get first status update
		status := <-statusChan

		// Check if completed
		completed := status.Status == "completed" ||
			status.Status == "failed" ||
			status.Status == "cancelled"

		return bulkProgressMsg{
			batchID:    v.batchID,
			statistics: status.Statistics,
			runs:       status.Runs,
			completed:  completed,
		}
	}
}

// cancelBatch cancels a bulk operation
func (v *BulkView) cancelBatch() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := v.client.CancelBulkRuns(ctx, v.batchID)
		if err != nil {
			return errMsg{err}
		}
		return bulkCancelledMsg{}
	}
}