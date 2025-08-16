package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

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
	Short: "Create a new run from a JSON, YAML, or Markdown file",
	Long: `Create a new run from a configuration file containing the task details.

SUPPORTED FORMATS:
  • JSON (.json)                 - Standard JSON configuration
  • YAML (.yaml, .yml)           - YAML configuration
  • Markdown (.md, .markdown)    - Markdown with YAML frontmatter
  • Stdin                        - Pipe JSON directly (no file needed)

CONFIGURATION FIELDS:

Required:
  • prompt      (string)  - The task description/instructions for the AI
  • repository  (string)  - Repository name in format "owner/repo"
  • target      (string)  - Target branch for the changes
  • title       (string)  - Title for the run

Optional:
  • source      (string)  - Source branch (defaults to "main")
  • runType     (string)  - Type: "run" or "plan" (defaults to "run")
  • context     (string)  - Additional context or instructions
  • files       (array)   - List of specific files to include

EXAMPLES:

JSON file (task.json):
  {
    "prompt": "Fix the login bug in auth.js",
    "repository": "myorg/webapp",
    "source": "main",
    "target": "fix/login-bug",
    "title": "Fix authentication issue",
    "runType": "run",
    "context": "Users report login fails after 5 attempts",
    "files": ["src/auth.js", "src/utils/validation.js"]
  }

YAML file (task.yaml):
  prompt: Fix the login bug in auth.js
  repository: myorg/webapp
  source: main
  target: fix/login-bug
  title: Fix authentication issue
  runType: run
  context: Users report login fails after 5 attempts
  files:
    - src/auth.js
    - src/utils/validation.js

Markdown with frontmatter (task.md):
  ---
  prompt: Fix the login bug
  repository: myorg/webapp
  target: fix/login-bug
  title: Fix authentication issue
  ---
  # Additional Context
  
  Users are experiencing login failures after 5 attempts.
  The issue seems to be in the rate limiting logic.

Stdin (pipe JSON):
  echo '{"prompt":"Fix bug","repository":"org/repo","target":"fix","title":"Bug fix"}' | repobird run

AUTO-DETECTION:
  If running from a git repository:
  • Repository name auto-detected from git remote
  • Source branch auto-detected from current branch

USAGE:
  repobird run task.json                    # Run from file
  repobird run task.yaml --follow           # Run and follow status
  repobird run task.md --dry-run            # Validate without running
  cat task.json | repobird run              # Pipe from stdin
  repobird run                              # Error: requires file or stdin`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCommand,
}

func init() {
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate input without creating a run")
	runCmd.Flags().BoolVar(&follow, "follow", false, "follow the run status after creation")
}

func runCommand(cmd *cobra.Command, args []string) error {
	if cfg.APIKey == "" {
		return errors.NoAPIKeyError()
	}

	var runConfig *models.RunConfig
	var additionalContext string
	var err error

	if len(args) > 0 {
		// Load configuration from file (supports JSON, YAML, and Markdown)
		filename := args[0]
		var promptHandler *prompts.ValidationPromptHandler
		runConfig, additionalContext, promptHandler, err = utils.LoadConfigFromFileWithPrompts(filename)
		if err != nil {
			return fmt.Errorf("failed to load configuration file: %w", err)
		}

		// Process any validation prompts before proceeding
		if promptHandler != nil && promptHandler.HasPrompts() {
			shouldContinue, err := promptHandler.ProcessPrompts()
			if err != nil {
				return fmt.Errorf("failed to process validation prompts: %w", err)
			}
			if !shouldContinue {
				return fmt.Errorf("operation cancelled by user")
			}
		}
	} else {
		// Read JSON from stdin with unknown field handling
		runConfig, err = utils.ParseJSONFromStdin()
		if err != nil {
			return fmt.Errorf("failed to parse input: %w\nHint: Run 'repobird examples' to see configuration formats and schemas", err)
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
