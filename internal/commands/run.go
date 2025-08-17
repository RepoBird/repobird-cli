// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"context"
	"encoding/json"
	netstderrors "errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/bulk"
	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/prompts"
	"github.com/repobird/repobird-cli/internal/utils"
)

var (
	dryRun bool
	follow bool
)

var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Create runs from a JSON, YAML, Markdown, or bulk configuration file",
	Long: `Create one or more runs from a configuration file.

Supports single run or bulk run configurations in JSON, YAML, or Markdown format.

Examples:
  repobird run task.json                    # Run from file (single or bulk)
  repobird run tasks.yaml --follow           # Run and follow status
  repobird run task.md --dry-run            # Validate without running
  cat task.json | repobird run              # Pipe JSON from stdin

For configuration examples and field descriptions:
  repobird examples                         # View all examples
  repobird examples generate run -o task.json
  repobird examples generate bulk -o tasks.json`,
	Args:          cobra.MaximumNArgs(1),
	RunE:          runCommand,
	SilenceErrors: true,
	SilenceUsage:  false, // Let Cobra show usage for arg/flag errors
}

func init() {
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate input without creating a run")
	runCmd.Flags().BoolVar(&follow, "follow", false, "follow the run status after creation")
}

func runCommand(cmd *cobra.Command, args []string) error {
	// For execution errors (not arg/flag errors), suppress usage
	cmd.SilenceUsage = true

	if cfg.APIKey == "" {
		return errors.NoAPIKeyError()
	}

	// Check if it's stdin input
	if len(args) == 0 {
		// Check if stdin has data
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			// No stdin data and no file argument - show help
			_ = cmd.Help()
			return nil
		}

		// Read JSON from stdin with unknown field handling
		runConfig, promptHandler, err := utils.ParseJSONFromStdinWithPrompts()
		if err != nil {
			// The error already contains helpful hints, don't add more
			return err
		}

		// Show informational messages about unknown fields (not prompts)
		if promptHandler != nil && promptHandler.HasUnknownFields() {
			unknownFields := promptHandler.GetUnknownFields()
			if len(unknownFields) > 0 {
				fmt.Fprintf(os.Stderr, "Note: Ignoring unknown fields in configuration: %s\n", strings.Join(unknownFields, ", "))

				// Show suggestions if available
				suggestions := promptHandler.GetFieldSuggestions()
				for field, suggestion := range suggestions {
					if suggestion != "" {
						fmt.Fprintf(os.Stderr, "      Did you mean '%s' instead of '%s'?\n", suggestion, field)
					}
				}
			}
		}

		return processSingleRun(runConfig, "")
	}

	// Load configuration from file
	filename := args[0]

	// Check if it's a bulk configuration FIRST, before trying to parse it
	isBulk, err := bulk.IsBulkConfig(filename)
	if err != nil {
		// If we can't read the file at all, return the error
		return fmt.Errorf("failed to read configuration file: %w", err)
	}

	if isBulk {
		// Process as bulk configuration - no need for validation prompts
		return processBulkRuns(filename)
	}

	// Process as single run configuration
	var runConfig *models.RunConfig
	var additionalContext string
	var promptHandler *prompts.ValidationPromptHandler

	runConfig, additionalContext, promptHandler, err = utils.LoadConfigFromFileWithPrompts(filename)
	if err != nil {
		return fmt.Errorf("failed to load configuration file: %w", err)
	}

	// Show informational messages about unknown fields (not prompts)
	if promptHandler != nil && promptHandler.HasUnknownFields() {
		unknownFields := promptHandler.GetUnknownFields()
		if len(unknownFields) > 0 {
			fmt.Fprintf(os.Stderr, "Note: Ignoring unknown fields in configuration: %s\n", strings.Join(unknownFields, ", "))

			// Show suggestions if available
			suggestions := promptHandler.GetFieldSuggestions()
			for field, suggestion := range suggestions {
				if suggestion != "" {
					fmt.Fprintf(os.Stderr, "      Did you mean '%s' instead of '%s'?\n", suggestion, field)
				}
			}
		}
	}

	return processSingleRun(runConfig, additionalContext)
}

func processSingleRun(runConfig *models.RunConfig, additionalContext string) error {
	// Validate the configuration
	if err := utils.ValidateRunConfig(runConfig); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Convert to domain request
	createReq := domain.CreateRunRequest{
		Prompt:         runConfig.Prompt,
		RepositoryName: runConfig.Repository,
		SourceBranch:   runConfig.Source,
		TargetBranch:   runConfig.Target,
		RunType:        runConfig.RunType,
		Title:          runConfig.Title,
		Context:        runConfig.Context,
		Files:          runConfig.Files,
	}

	// Append additional markdown context if present
	if additionalContext != "" {
		if createReq.Context != "" {
			createReq.Context = createReq.Context + "\n\n" + additionalContext
		} else {
			createReq.Context = additionalContext
		}
	}

	// Auto-detection disabled for now
	// TODO: Enable when feature is ready
	// container := getContainer()
	// gitService := container.GitService()
	//
	// if createReq.RepositoryName == "" && gitService.IsGitRepository() {
	// 	repo, err := gitService.GetRepositoryName()
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Warning: Could not auto-detect repository: %v\n", err)
	// 	} else {
	// 		createReq.RepositoryName = repo
	// 		fmt.Printf("Auto-detected repository: %s\n", repo)
	// 	}
	// }
	//
	// if createReq.SourceBranch == "" && gitService.IsGitRepository() {
	// 	branch, err := gitService.GetCurrentBranch()
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Warning: Could not auto-detect branch: %v\n", err)
	// 	} else {
	// 		createReq.SourceBranch = branch
	// 		fmt.Printf("Auto-detected source branch: %s\n", branch)
	// 	}
	// }

	if dryRun {
		fmt.Println("Validation successful. Run would be created with:")
		b, _ := json.MarshalIndent(createReq, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	// Use service layer to create run
	container := getContainer()
	runService := container.RunService()
	ctx := context.Background()

	fmt.Println("Creating run...")
	run, err := runService.CreateRun(ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to create run: %s", errors.FormatUserError(err))
	}

	fmt.Printf("Run created successfully!\n")
	fmt.Printf("ID: %s\n", run.ID)
	fmt.Printf("Status: %s\n", run.Status)
	fmt.Printf("Repository: %s\n", run.RepositoryName)
	fmt.Printf("Source: %s → Target: %s\n", run.SourceBranch, run.TargetBranch)

	if follow {
		fmt.Println("\nFollowing run status...")
		return followRunStatus(runService, run.ID)
	}

	return nil
}

func processBulkRuns(filename string) error {
	// Load bulk configuration
	bulkConfig, err := bulk.ParseBulkConfig(filename)
	if err != nil {
		return fmt.Errorf("failed to load bulk configuration: %w", err)
	}

	// Auto-detection disabled for now
	// TODO: Enable when feature is ready
	// container := getContainer()
	// gitService := container.GitService()
	//
	// if bulkConfig.Repository == "" && gitService.IsGitRepository() {
	// 	repo, err := gitService.GetRepositoryName()
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Warning: Could not auto-detect repository: %v\n", err)
	// 	} else {
	// 		bulkConfig.Repository = repo
	// 		fmt.Printf("Auto-detected repository: %s\n", repo)
	// 	}
	// }
	//
	// if bulkConfig.Source == "" && gitService.IsGitRepository() {
	// 	branch, err := gitService.GetCurrentBranch()
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Warning: Could not auto-detect branch: %v\n", err)
	// 	} else {
	// 		bulkConfig.Source = branch
	// 		fmt.Printf("Auto-detected source branch: %s\n", branch)
	// 	}
	// }

	if dryRun {
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("✓ Configuration valid"))
		fmt.Printf("Repository: %s\n", bulkConfig.Repository)
		fmt.Printf("Source: %s\n", bulkConfig.Source)
		fmt.Printf("RunType: %s\n", bulkConfig.RunType)
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

	// Process bulk runs using the bulk command's logic
	return executeBulkRuns(bulkConfig)
}

func followRunStatus(runService domain.RunService, runID string) error {
	ctx := context.Background()
	startTime := time.Now()
	lastStatus := ""

	callback := func(status string, message string) {
		if status != lastStatus {
			fmt.Printf("\r\033[K") // Clear line
			fmt.Printf("[%s] Status: %s\n", time.Now().Format("15:04:05"), status)
			lastStatus = status
		} else {
			elapsed := time.Since(startTime)
			if message != "" {
				fmt.Printf("\r[%s] %s - %s", formatDuration(elapsed), status, message)
			} else {
				fmt.Printf("\r[%s] %s", formatDuration(elapsed), status)
			}
		}
	}

	finalRun, err := runService.WaitForCompletion(ctx, runID, callback)
	if err != nil {
		return fmt.Errorf("failed to follow run status: %s", errors.FormatUserError(err))
	}

	fmt.Printf("\r\033[K") // Clear line
	if finalRun.Status == domain.StatusFailed && finalRun.Error != "" {
		fmt.Printf("Run failed: %s\n", finalRun.Error)
	} else {
		fmt.Printf("Run completed with status: %s\n", finalRun.Status)
	}
	return nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func executeBulkRuns(bulkConfig *bulk.BulkConfig) error {
	// Create API client
	apiURL := utils.GetAPIURL(cfg.APIURL)
	client := api.NewClient(cfg.APIKey, apiURL, debug)

	// Generate file hashes for tracking purposes
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
		fileHashCache.Set(fmt.Sprintf("bulk-run-%d", i), hash)
	}

	// Convert to API request format
	bulkRequest := &dto.BulkRunRequest{
		RepositoryName: bulkConfig.Repository,
		RepoID:         bulkConfig.RepoID,
		RunType:        bulkConfig.RunType,
		SourceBranch:   bulkConfig.Source,
		BatchTitle:     bulkConfig.BatchTitle,
		Force:          false,
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
		// Always include file hash for tracking purposes
		if i < len(runHashes) {
			item.FileHash = runHashes[i]
		}
		bulkRequest.Runs[i] = item
	}

	// Submit bulk runs
	ctx := context.Background()

	// Display submission info
	fmt.Println(lipgloss.NewStyle().Bold(true).Render("Submitting bulk runs..."))
	fmt.Printf("Repository: %s\n", bulkConfig.Repository)
	fmt.Printf("Total runs: %d\n", len(bulkConfig.Runs))
	fmt.Println("\nThis may take up to 5 minutes. Please wait...")

	// Show a progress indicator with elapsed time
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
				_ = os.Stdout.Sync() // Force flush to ensure animation
				spinnerIdx = (spinnerIdx + 1) % len(spinner)
			}
		}
	}()

	bulkResp, err := client.CreateBulkRuns(ctx, bulkRequest)
	done <- true // Stop spinner
	close(done)  // Clean up channel

	if err != nil {
		// Check for timeout error
		if ctx.Err() == context.DeadlineExceeded || netstderrors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("bulk submission timed out after 5 minutes. The server may still be processing your runs.\nTry checking the status later with 'repobird status'")
		}

		// Check if this is a 403 error which might indicate duplicate runs or quota issues
		var authErr *errors.AuthError
		if netstderrors.As(err, &authErr) {
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

	// Handle different status codes
	if bulkResp.StatusCode == http.StatusMultiStatus {
		// 207 Multi-Status: Some runs still processing
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
	} else if len(bulkResp.Data.Failed) > 0 {
		// Some runs failed
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render("\n⚠ Partial success:"))
		fmt.Printf("Created: %d/%d runs\n", bulkResp.Data.Metadata.TotalSuccessful, bulkResp.Data.Metadata.TotalRequested)

		// Check if failures are due to duplicates
		for _, runErr := range bulkResp.Data.Failed {
			fmt.Printf("  ✗ Run %d: %s\n", runErr.RequestIndex+1, runErr.Message)
			// Note: Duplicates are no longer blocked as --force is deprecated
		}
	} else {
		// All runs created successfully
		fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("\n✓ All runs created successfully"))
	}

	// Display created runs
	if len(bulkResp.Data.Successful) > 0 {
		fmt.Println("\nCreated runs:")
		for _, run := range bulkResp.Data.Successful {
			fmt.Printf("  • %s (ID: %d)\n", run.Title, run.ID)
		}
	}

	// Follow progress if requested
	if follow && len(bulkResp.Data.Successful) > 0 {
		fmt.Println("\nFollowing batch progress...")
		return followBulkProgress(ctx, client, bulkResp.Data.BatchID)
	}

	fmt.Printf("\nBatch ID: %s\n", bulkResp.Data.BatchID)
	fmt.Println("Use 'repobird bulk status " + bulkResp.Data.BatchID + "' to check progress")

	return nil
}
