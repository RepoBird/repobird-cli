package commands

import (
	"context"
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
	tuiviews "github.com/repobird/repobird-cli/internal/tui/views"
	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	bulkFollow      bool
	bulkDryRun      bool
	bulkForce       bool
	bulkInteractive bool
	bulkParallel    int
)

// NewBulkCommand creates the bulk command
func NewBulkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bulk [files...]",
		Short: "Submit multiple runs in parallel from configuration files",
		Long: `Submit multiple runs in parallel from configuration files.
		
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
	cmd.Flags().BoolVar(&bulkForce, "force", false, "Override duplicate detection")
	cmd.Flags().BoolVarP(&bulkInteractive, "interactive", "i", false, "Interactive bulk mode")
	cmd.Flags().IntVarP(&bulkParallel, "parallel", "p", 5, "Max concurrent runs")

	return cmd
}

func runBulk(cmd *cobra.Command, args []string) error {
	// Interactive mode
	if bulkInteractive || len(args) == 0 {
		return runBulkInteractive()
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("%s", errors.FormatUserError(err))
	}

	// Validate API key
	if cfg.APIKey == "" {
		return fmt.Errorf("API key not configured. Run 'repobird config set api-key YOUR_KEY' first")
	}

	// Expand glob patterns and resolve paths
	var files []string
	for _, pattern := range args {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return fmt.Errorf("invalid glob pattern %s: %w", pattern, err)
		}
		if len(matches) == 0 {
			// If no glob matches, treat as literal file
			files = append(files, pattern)
		} else {
			files = append(files, matches...)
		}
	}

	// Load bulk configuration
	bulkConfig, err := bulk.LoadBulkConfig(files)
	if err != nil {
		return fmt.Errorf("%s", errors.FormatUserError(err))
	}

	// Validate configuration
	if bulkDryRun {
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

	// Apply force flag if set
	if bulkForce {
		bulkConfig.Force = true
	}

	// Create API client
	apiURL := os.Getenv("REPOBIRD_API_URL")
	if apiURL == "" {
		apiURL = api.DefaultAPIURL
	}
	client := api.NewClient(cfg.APIKey, apiURL, debug)

	// Generate file hashes for duplicate detection
	fileHashCache := cache.NewFileHashCache()
	var runHashes []string
	for i, run := range bulkConfig.Runs {
		// Create a temporary hash based on the run content
		hashContent := fmt.Sprintf("%s-%s-%s-%s-%d",
			bulkConfig.Repository,
			run.Prompt,
			run.Target,
			run.Context,
			i,
		)
		hash := cache.CalculateStringHash(hashContent)
		runHashes = append(runHashes, hash)

		// Cache the hash
		fileHashCache.Set(fmt.Sprintf("bulk-run-%d", i), hash)
	}

	// Convert to API request format
	bulkRequest := &dto.BulkRunRequest{
		RepositoryName: bulkConfig.Repository,
		RepoID:         bulkConfig.RepoID,
		RunType:        bulkConfig.RunType,
		SourceBranch:   bulkConfig.Source,
		BatchTitle:     bulkConfig.BatchTitle,
		Force:          bulkConfig.Force,
		Runs:           make([]dto.RunItem, len(bulkConfig.Runs)),
		Options: dto.BulkOptions{
			Parallel: bulkParallel,
		},
	}

	for i, run := range bulkConfig.Runs {
		bulkRequest.Runs[i] = dto.RunItem{
			Prompt:   run.Prompt,
			Title:    run.Title,
			Target:   run.Target,
			Context:  run.Context,
			FileHash: runHashes[i],
		}
	}

	// Submit bulk runs
	ctx := context.Background()

	// Display submission info
	fmt.Println(lipgloss.NewStyle().Bold(true).Render("Submitting bulk runs..."))
	fmt.Printf("Repository: %s\n", bulkConfig.Repository)
	fmt.Printf("Total runs: %d\n", len(bulkConfig.Runs))
	fmt.Println("\nThis may take a few minutes for large batches. Please wait...")

	// Show a simple spinner while waiting
	spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIdx := 0
	done := make(chan bool)

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
				fmt.Printf("\r%s Processing...", spinner[spinnerIdx])
				spinnerIdx = (spinnerIdx + 1) % len(spinner)
			}
		}
	}()

	bulkResp, err := client.CreateBulkRuns(ctx, bulkRequest)
	done <- true // Stop spinner

	if err != nil {
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

		for _, runErr := range bulkResp.Data.Failed {
			fmt.Printf("  ✗ Run %d: %s\n", runErr.RequestIndex+1, runErr.Message)
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
	if bulkFollow && len(bulkResp.Data.Successful) > 0 {
		fmt.Println("\nFollowing batch progress...")
		return followBulkProgress(ctx, client, bulkResp.Data.BatchID)
	}

	fmt.Printf("\nBatch ID: %s\n", bulkResp.Data.BatchID)
	fmt.Println("Use 'repobird bulk status " + bulkResp.Data.BatchID + "' to check progress")

	return nil
}

func runBulkInteractive() error {
	// Load configuration for API key
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("%s", errors.FormatUserError(err))
	}

	if cfg.APIKey == "" {
		return fmt.Errorf("API key not configured. Run 'repobird config set api-key YOUR_KEY' first")
	}

	// Create API client
	apiURL := os.Getenv("REPOBIRD_API_URL")
	if apiURL == "" {
		apiURL = api.DefaultAPIURL
	}
	client := api.NewClient(cfg.APIKey, apiURL, debug)

	// Launch bulk TUI view
	bulkView := tuiviews.NewBulkView(client)
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

	for status := range statusChan {
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
			break
		}
	}

	return nil
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
