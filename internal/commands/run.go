package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/utils"
)

var (
	dryRun bool
	follow bool
)

var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Create a new run from a JSON or Markdown file",
	Long: `Create a new run from a JSON or Markdown file containing the task details.
Supports both JSON format and Markdown files with YAML frontmatter.
If no file is specified, reads JSON from stdin.`,
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

	var runConfig *models.RunConfig
	var additionalContext string
	var err error

	if len(args) > 0 {
		// Check if it's a markdown or JSON file
		filename := args[0]
		if strings.HasSuffix(strings.ToLower(filename), ".md") ||
			strings.HasSuffix(strings.ToLower(filename), ".markdown") {
			// Parse markdown file with frontmatter
			runConfig, additionalContext, err = utils.ParseMarkdownConfig(filename)
			if err != nil {
				return fmt.Errorf("failed to parse markdown file: %w", err)
			}
		} else {
			// Parse JSON file
			file, err := os.Open(filename)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer func() { _ = file.Close() }()

			var runReq models.RunRequest
			if err := json.NewDecoder(file).Decode(&runReq); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}

			// Convert RunRequest to RunConfig
			runConfig = &models.RunConfig{
				Prompt:     runReq.Prompt,
				Repository: runReq.Repository,
				Source:     runReq.Source,
				Target:     runReq.Target,
				RunType:    string(runReq.RunType),
				Title:      runReq.Title,
				Context:    runReq.Context,
				Files:      runReq.Files,
			}
		}
	} else {
		// Read JSON from stdin
		var runReq models.RunRequest
		if err := json.NewDecoder(os.Stdin).Decode(&runReq); err != nil {
			return fmt.Errorf("failed to parse JSON from stdin: %w", err)
		}

		// Convert RunRequest to RunConfig
		runConfig = &models.RunConfig{
			Prompt:     runReq.Prompt,
			Repository: runReq.Repository,
			Source:     runReq.Source,
			Target:     runReq.Target,
			RunType:    string(runReq.RunType),
			Title:      runReq.Title,
			Context:    runReq.Context,
			Files:      runReq.Files,
		}
	}

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
