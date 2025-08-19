// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/bulk"
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/errors"
	tuicache "github.com/repobird/repobird-cli/internal/tui/cache"
	tuiviews "github.com/repobird/repobird-cli/internal/tui/views"
	"github.com/repobird/repobird-cli/internal/utils"
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	bulkFollow      bool
	bulkDryRun      bool
	bulkForce       bool
	bulkInteractive bool
)

// NewBulkCommand creates the bulk command
func NewBulkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk [files...]",
		Short: "Submit multiple runs from configuration files",
		Long: `Submit multiple runs from configuration files.
		
Supports multiple formats:
- Bulk JSON/YAML files with multiple runs
- Multiple single-run configuration files
- JSONL format for streaming large batches
- Markdown format with front matter

Examples:
  # Single bulk config file
  repobird bulk tasks.json
  
  # Multiple files merged into batch
  repobird bulk auth.yaml payment.yaml user.json
  
  # JSONL format
  repobird bulk runs.jsonl
  
  # Interactive mode
  repobird bulk --interactive
  
  # Follow batch progress
  repobird bulk tasks.json --follow`,
		RunE: runBulk,
	}

	cmd.Flags().BoolVarP(&bulkFollow, "follow", "f", false, "Follow batch progress")
	cmd.Flags().BoolVar(&bulkDryRun, "dry-run", false, "Validate without submitting")
	cmd.Flags().BoolVar(&bulkForce, "force", false, "Deprecated - has no effect (kept for backwards compatibility)")
	cmd.Flags().BoolVarP(&bulkInteractive, "interactive", "i", false, "Interactive bulk mode")

	// Mark force flag as deprecated
	_ = cmd.Flags().MarkDeprecated("force", "file hashes are now for tracking only and won't block runs")

	return cmd
}

func runBulk(cmd *cobra.Command, args []string) error {
	// Interactive mode
	if bulkInteractive || len(args) == 0 {
		return runBulkInteractive()
	}

	// Load and validate configuration
	cfg, err := loadAndValidateConfig()
	if err != nil {
		return err
	}

	// Expand file paths from arguments
	files, err := expandFilePaths(args)
	if err != nil {
		return err
	}

	// Load bulk configuration
	bulkConfig, err := bulk.LoadBulkConfig(files)
	if err != nil {
		return fmt.Errorf("%s", errors.FormatUserError(err))
	}

	// Handle dry run
	if bulkDryRun {
		return printDryRunSummary(bulkConfig)
	}

	// Create API client
	apiURL := utils.GetAPIURL(cfg.APIURL)
	client := api.NewClient(cfg.APIKey, apiURL, debug)

	// Prepare bulk request
	bulkRequest := prepareBulkRequest(bulkConfig)

	// Submit with progress indicator
	bulkResp, err := submitBulkRunsWithProgress(client, bulkRequest, bulkConfig)
	if err != nil {
		return err
	}

	// Display results
	displayBulkSubmissionResults(bulkResp)

	// Follow progress if requested
	if bulkFollow && len(bulkResp.Data.Successful) > 0 {
		fmt.Println("\nFollowing batch progress...")
		// Create context with 1h 30m timeout
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Minute)
		defer cancel()
		return followBulkProgress(ctx, client, bulkResp.Data.BatchID)
	}

	fmt.Printf("\nBatch ID: %s\n", bulkResp.Data.BatchID)
	fmt.Println("Use 'repobird bulk status " + bulkResp.Data.BatchID + "' to check progress")

	return nil
}

func loadAndValidateConfig() (*config.Config, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("%s", errors.FormatUserError(err))
	}

	if cfg.APIKey == "" {
		return nil, errors.NoAPIKeyError()
	}

	return cfg, nil
}

func expandFilePaths(args []string) ([]string, error) {
	var files []string
	for _, pattern := range args {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}
		if len(matches) == 0 {
			// If no glob matches, treat as literal file
			files = append(files, pattern)
		} else {
			files = append(files, matches...)
		}
	}
	return files, nil
}

func printDryRunSummary(bulkConfig *bulk.BulkConfig) error {
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓ Configuration valid"))
	fmt.Printf("Repository: %s\n", bulkConfig.Repository)
	fmt.Printf("Total runs: %d\n", len(bulkConfig.Runs))
	for i, run := range bulkConfig.Runs {
		title := run.Title
		if title == "" {
			title = fmt.Sprintf("Run %d", i+1)
		}
		fmt.Printf("  - %s\n", title)
	}
	return nil
}

func prepareBulkRequest(bulkConfig *bulk.BulkConfig) *dto.BulkRunRequest {
	// Generate file hashes for tracking purposes
	runHashes := generateRunHashes(bulkConfig)

	// Convert to API request format
	bulkRequest := &dto.BulkRunRequest{
		RepositoryName: bulkConfig.Repository,
		RepoID:         bulkConfig.RepoID,
		RunType:        bulkConfig.RunType,
		SourceBranch:   bulkConfig.Source,
		BatchTitle:     bulkConfig.BatchTitle,
		Force:          false, // Deprecated
		Runs:           make([]dto.RunItem, len(bulkConfig.Runs)),
		Options:        dto.BulkOptions{},
	}

	for i, run := range bulkConfig.Runs {
		item := dto.RunItem{
			Prompt:  run.Prompt,
			Title:   run.Title,
			Target:  run.Target,
			Context: run.Context,
		}
		if i < len(runHashes) {
			item.FileHash = runHashes[i]
		}
		bulkRequest.Runs[i] = item
	}

	return bulkRequest
}

func generateRunHashes(bulkConfig *bulk.BulkConfig) []string {
	var runHashes []string
	fileHashCache := cache.NewFileHashCache()

	for i, run := range bulkConfig.Runs {
		// Create a hash based on the run content for tracking
		hashContent := fmt.Sprintf("%s-%s-%s-%s",
			bulkConfig.Repository,
			run.Prompt,
			run.Target,
			run.Context,
		)
		hash := cache.CalculateStringHash(hashContent)
		runHashes = append(runHashes, hash)

		// Cache the hash for tracking
		fileHashCache.Set(fmt.Sprintf("bulk-run-%d", i), hash)
	}

	return runHashes
}

func submitBulkRunsWithProgress(client *api.Client, bulkRequest *dto.BulkRunRequest, bulkConfig *bulk.BulkConfig) (*dto.BulkRunResponse, error) {
	ctx := context.Background()

	// Display submission info
	fmt.Println(lipgloss.NewStyle().Bold(true).Render("Submitting bulk runs..."))
	fmt.Printf("Repository: %s\n", bulkConfig.Repository)
	fmt.Printf("Total runs: %d\n", len(bulkConfig.Runs))
	fmt.Println("\nThis may take up to 5 minutes. Please wait...")

	// Show progress spinner
	done := showProgressSpinner()

	bulkResp, err := client.CreateBulkRuns(ctx, bulkRequest)
	done <- true // Stop spinner
	close(done)  // Clean up channel

	if err != nil {
		return nil, handleBulkSubmissionError(err, ctx, bulkConfig)
	}

	return bulkResp, nil
}

func showProgressSpinner() chan bool {
	startTime := time.Now()
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIdx := 0
	done := make(chan bool, 1) // Buffered to prevent goroutine leak

	// Start spinner in background
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				fmt.Print("\r\033[K") // Clear the spinner line
				return
			case <-ticker.C:
				elapsed := time.Since(startTime)
				fmt.Printf("\r%s Processing... (%.0fs)", spinner[spinnerIdx], elapsed.Seconds())
				_ = os.Stdout.Sync()
				spinnerIdx = (spinnerIdx + 1) % len(spinner)
			}
		}
	}()

	return done
}

func handleBulkSubmissionError(err error, ctx context.Context, bulkConfig *bulk.BulkConfig) error {
	// Check for timeout error
	if ctx.Err() == context.DeadlineExceeded || stderrors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("bulk submission timed out after 5 minutes. The server may still be processing your runs.\nTry checking the status later with 'repobird status'")
	}

	// Check if this is a 403 error which might indicate quota issues
	var authErr *errors.AuthError
	if stderrors.As(err, &authErr) {
		errMsg := errors.FormatUserError(err)
		// Check for quota-related messages
		if strings.Contains(strings.ToLower(errMsg), "insufficient run") ||
			strings.Contains(strings.ToLower(errMsg), "no runs remaining") {
			return fmt.Errorf("%s\n\nUpgrade your plan at %s", errMsg, config.GetPricingURL())
		}
		// For other 403 errors, just return the error message
		if !bulkConfig.Force {
			return fmt.Errorf("%s", errMsg)
		}
	}
	return fmt.Errorf("%s", errors.FormatUserError(err))
}

func displayBulkSubmissionResults(bulkResp *dto.BulkRunResponse) {
	if bulkResp.StatusCode == http.StatusMultiStatus {
		displayMultiStatusResult(bulkResp)
	} else if len(bulkResp.Data.Failed) > 0 {
		displayPartialSuccessResult(bulkResp)
	} else {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("\n✓ All runs created successfully"))
	}

	// Display created runs
	if len(bulkResp.Data.Successful) > 0 {
		fmt.Println("\nCreated runs:")
		for _, run := range bulkResp.Data.Successful {
			fmt.Printf("  • %s (ID: %d)\n", run.Title, run.ID)
		}
	}
}

func displayMultiStatusResult(bulkResp *dto.BulkRunResponse) {
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("\n⚠ Bulk submission in progress:"))
	fmt.Printf("The server is still processing your runs. This is normal for large batches.\n")
	fmt.Printf("Created: %d/%d runs so far\n", bulkResp.Data.Metadata.TotalSuccessful, bulkResp.Data.Metadata.TotalRequested)

	if len(bulkResp.Data.Failed) > 0 {
		fmt.Println("\nFailed runs:")
		for _, runErr := range bulkResp.Data.Failed {
			fmt.Printf("  ✗ Run %d: %s\n", runErr.RequestIndex+1, runErr.Message)
		}
	}

	fmt.Println("\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Render("ℹ  The remaining runs are being processed in the background."))
	fmt.Println("Use --follow or check status to monitor progress.")
}

func displayPartialSuccessResult(bulkResp *dto.BulkRunResponse) {
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("\n⚠ Partial success:"))
	fmt.Printf("Created: %d/%d runs\n", bulkResp.Data.Metadata.TotalSuccessful, bulkResp.Data.Metadata.TotalRequested)

	for _, runErr := range bulkResp.Data.Failed {
		fmt.Printf("  ✗ Run %d: %s\n", runErr.RequestIndex+1, runErr.Message)
	}
}

func runBulkInteractive() error {
	// Load configuration for API key
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("%s", errors.FormatUserError(err))
	}

	if cfg.APIKey == "" {
		return errors.NoAPIKeyError()
	}

	// Create API client
	apiURL := utils.GetAPIURL(cfg.APIURL)
	client := api.NewClient(cfg.APIKey, apiURL, debug)

	// Launch bulk TUI view with a cache instance
	cache := tuicache.NewSimpleCache()
	bulkView := tuiviews.NewBulkView(client, cache)
	p := tea.NewProgram(bulkView, tea.WithAltScreen())

	_, err = p.Run()
	return err
}

func followBulkProgress(ctx context.Context, client *api.Client, batchID string) error {
	// Poll for status updates
	statusChan, err := client.PollBulkStatus(ctx, batchID, 2*time.Second)
	if err != nil {
		return err
	}

	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIdx := 0
	startTime := time.Now()

	for {
		select {
		case status, ok := <-statusChan:
			if !ok {
				// Channel closed - check if it was due to timeout
				if ctx.Err() == context.DeadlineExceeded {
					fmt.Print("\033[2K\r") // Clear line
					return fmt.Errorf("polling timeout exceeded (maximum wait time: 1h 30m). The batch may still be processing on the server")
				}
				return nil
			}

			// Clear previous line
			fmt.Print("\033[2K\r")

			// Display progress
			progressBar := makeProgressBar(status.Statistics)
			fmt.Printf("%s %s", spinner[spinnerIdx], progressBar)

			spinnerIdx = (spinnerIdx + 1) % len(spinner)

			// Check for completion
			if status.Status == "completed" || status.Status == "failed" || status.Status == "cancelled" {
				fmt.Println("\n\nBatch completed!")
				displayBulkResults(status)
				return nil
			}

		case <-ctx.Done():
			fmt.Print("\033[2K\r") // Clear line
			elapsed := time.Since(startTime)
			return fmt.Errorf("polling timeout exceeded after %v (maximum wait time: 1h 30m). The batch may still be processing on the server", elapsed)
		}
	}
}

func makeProgressBar(stats dto.BulkStatistics) string {
	width := 40
	completed := stats.Completed + stats.Failed + stats.Cancelled
	total := stats.Total

	if total == 0 {
		return ""
	}

	progress := int(float64(completed) / float64(total) * float64(width))
	bar := strings.Repeat("█", progress) + strings.Repeat("░", width-progress)

	return fmt.Sprintf("[%s] %d/%d (Queued: %d, Processing: %d, Completed: %d, Failed: %d)",
		bar, completed, total, stats.Queued, stats.Processing, stats.Completed, stats.Failed)
}

func displayBulkResults(status dto.BulkStatusResponse) {
	fmt.Println("\nResults:")

	for _, run := range status.Runs {
		statusIcon := "○"
		statusColor := lipgloss.Color("7")

		switch run.Status {
		case "completed":
			statusIcon = "✓"
			statusColor = lipgloss.Color("10")
		case "failed":
			statusIcon = "✗"
			statusColor = lipgloss.Color("9")
		case "processing":
			statusIcon = "●"
			statusColor = lipgloss.Color("11")
		case "queued":
			statusIcon = "○"
			statusColor = lipgloss.Color("8")
		}

		style := lipgloss.NewStyle().Foreground(statusColor)
		fmt.Printf("  %s %s (ID: %d)\n",
			style.Render(statusIcon),
			run.Title,
			run.ID,
		)

		if run.Error != "" {
			fmt.Printf("    Error: %s\n", run.Error)
		}
		if run.RunURL != "" {
			fmt.Printf("    URL: %s\n", run.RunURL)
		}
	}

	// Summary
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total: %d\n", status.Statistics.Total)
	fmt.Printf("  Completed: %d\n", status.Statistics.Completed)
	fmt.Printf("  Failed: %d\n", status.Statistics.Failed)
	fmt.Printf("  Cancelled: %d\n", status.Statistics.Cancelled)
}
