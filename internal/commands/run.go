package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
)

var (
	dryRun bool
	follow bool
)

var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Create a new run from a JSON file",
	Long: `Create a new run from a JSON file containing the task details.
If no file is specified, reads from stdin.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCommand,
}

func init() {
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate input without creating a run")
	runCmd.Flags().BoolVar(&follow, "follow", false, "follow the run status after creation")
}

func runCommand(cmd *cobra.Command, args []string) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("API key not configured. Set REPOBIRD_API_KEY or run 'repobird config set api-key'")
	}

	var input io.Reader
	if len(args) > 0 {
		file, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer func() { _ = file.Close() }()
		input = file
	} else {
		input = os.Stdin
	}

	var runReq models.RunRequest
	if err := json.NewDecoder(input).Decode(&runReq); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	if err := validateRunRequest(&runReq); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Convert to domain request
	createReq := domain.CreateRunRequest{
		Prompt:         runReq.Prompt,
		RepositoryName: runReq.Repository,
		SourceBranch:   runReq.Source,
		TargetBranch:   runReq.Target,
		RunType:        string(runReq.RunType),
		Title:          runReq.Title,
		Context:        runReq.Context,
		Files:          runReq.Files,
	}

	// Auto-detect git info if needed
	container := getContainer()
	gitService := container.GitService()

	if createReq.RepositoryName == "" && gitService.IsGitRepository() {
		repo, err := gitService.GetRepositoryName()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not auto-detect repository: %v\n", err)
		} else {
			createReq.RepositoryName = repo
			fmt.Printf("Auto-detected repository: %s\n", repo)
		}
	}

	if createReq.SourceBranch == "" && gitService.IsGitRepository() {
		branch, err := gitService.GetCurrentBranch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not auto-detect branch: %v\n", err)
		} else {
			createReq.SourceBranch = branch
			fmt.Printf("Auto-detected source branch: %s\n", branch)
		}
	}

	if dryRun {
		fmt.Println("Validation successful. Run would be created with:")
		b, _ := json.MarshalIndent(createReq, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	// Use service layer to create run
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

func validateRunRequest(req *models.RunRequest) error {
	if req.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if req.RunType == "" {
		req.RunType = models.RunTypeRun
	}

	if req.RunType != models.RunTypeRun && req.RunType != models.RunTypeApproval {
		return fmt.Errorf("invalid runType: %s (must be 'run' or 'approval')", req.RunType)
	}

	if req.Source == "" && req.Target != "" {
		return fmt.Errorf("source branch is required when target is specified")
	}

	return nil
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
