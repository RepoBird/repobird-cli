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
	dryRun                bool
	follow                bool
	repo                  string
	prompt                string
	source                string
	target                string
	baseBranch            string
	outputMode            string
	outputBranch          string
	prTargetBranch        string
	outputBranchPolicy    string
	title                 string
	runType               string
	contextFlag           string
	basicRun              bool
	proRun                bool
	branchOnly            bool
	acknowledgePromptRisk bool
)

type runPreset struct {
	RunType  string
	Label    string
	Model    string
	Provider string
}

var runPresets = map[string]runPreset{
	"basic": {
		RunType:  "basic",
		Label:    "Basic",
		Model:    "openrouter/deepseek/deepseek-v4-flash",
		Provider: "openrouter",
	},
	"pro": {
		RunType:  "pro",
		Label:    "Pro",
		Model:    "openrouter/moonshotai/kimi-k2.6",
		Provider: "openrouter",
	},
}

var runCmd = &cobra.Command{
	Use:   "run [file]",
	Short: "Create a run from a JSON, YAML, or Markdown configuration file, or with flags",
	Long: `Create one or more runs from a configuration file or using command-line flags.

Supports single run configurations in JSON, YAML, or Markdown format.
Can also create a single run directly using command-line flags.

Examples:
  # Run from file
  repobird run task.json                    # Run from file
  repobird run tasks.yaml --follow           # Run and follow status
  repobird run task.md --dry-run            # Validate without running
  cat task.json | repobird run              # Pipe JSON from stdin

  # Run with flags
  repobird run -r owner/repo -p "Fix the bug in auth"
  repobird run --basic -r owner/repo -p "Fix a small bug"
  repobird run --pro -r owner/repo -p "Implement OAuth"
  repobird run --repo owner/repo --prompt "Add tests" --follow
  repobird run -r owner/repo -p "Refactor" --source dev --target main
  repobird run -r owner/repo -p "Update generated docs" --branch-only

  # Using prompt from file
  repobird run -r owner/repo -p @prompt.txt         # Read prompt from file
  repobird run -r owner/repo -p @prompt.md          # Markdown file as prompt
  echo "Fix auth bug" | repobird run -r owner/repo -p -  # Prompt from stdin
  repobird run -r owner/repo -p "@@starts with @"   # Use @@ to escape @

  # Using context from file
  repobird run -r owner/repo -p "Refactor" --context @context.md
  repobird run -r owner/repo -p @task.txt --context @requirements.md

For configuration examples and field descriptions:
  repobird examples                         # View all examples
  repobird examples generate run -o task.json`,
	Args:          cobra.MaximumNArgs(1),
	RunE:          runCommand,
	SilenceErrors: true,
	SilenceUsage:  false, // Let Cobra show usage for arg/flag errors
}

//nolint:gochecknoinits // Required for CLI command registration
func init() {
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate input without creating a run")
	runCmd.Flags().BoolVar(&follow, "follow", false, "follow the run status after creation")

	// Flags for direct run creation
	runCmd.Flags().StringVarP(&repo, "repo", "r", "", "repository name (owner/repo or numeric ID)")
	runCmd.Flags().StringVarP(&prompt, "prompt", "p", "", "prompt for the run (use @file to read from file, - for stdin)")
	runCmd.Flags().StringVar(&source, "source", "", "legacy alias for --base-branch")
	runCmd.Flags().StringVar(&target, "target", "", "legacy target branch alias")
	runCmd.Flags().StringVar(&baseBranch, "base-branch", "", "base branch to start work from (optional)")
	runCmd.Flags().StringVar(&outputMode, "output-mode", "", "output mode: 'pr' or 'branch' (optional, default: pr)")
	runCmd.Flags().StringVar(&outputBranch, "output-branch", "", "branch to push generated commits to (optional)")
	runCmd.Flags().StringVar(&prTargetBranch, "pr-target-branch", "", "branch the pull request targets (optional)")
	runCmd.Flags().StringVar(&outputBranchPolicy, "output-branch-policy", "", "output branch policy: 'create' or 'reuse' (optional)")
	runCmd.Flags().StringVar(&title, "title", "", "title for the run (optional)")
	runCmd.Flags().StringVar(&runType, "run-type", "", "type of run: 'run' (optional, default: run); 'plan' is development-only during the OpenCode migration")
	runCmd.Flags().BoolVar(&basicRun, "basic", false, "use the Basic cloud agent preset")
	runCmd.Flags().BoolVar(&proRun, "pro", false, "use the Pro cloud agent preset")
	runCmd.Flags().BoolVar(&branchOnly, "branch-only", false, "push commits to a branch without creating a pull request")
	runCmd.Flags().BoolVar(&branchOnly, "no-pr", false, "alias for --branch-only")
	runCmd.Flags().BoolVar(&acknowledgePromptRisk, "acknowledge-prompt-risk", false, "acknowledge prompt-risk warning and create the run")
	runCmd.Flags().StringVar(&contextFlag, "context", "", "additional context (use @file to read from file, - for stdin)")
}

func runCommand(cmd *cobra.Command, args []string) error {
	return runCommandWithPreset(cmd, args, "")
}

func runCommandWithPreset(cmd *cobra.Command, args []string, presetName string) error {
	// For execution errors (not arg/flag errors), suppress usage
	cmd.SilenceUsage = true

	if cfg.APIKey == "" {
		return errors.NoAPIKeyError()
	}

	selectedPreset, err := resolveRunPreset(presetName)
	if err != nil {
		return err
	}

	if selectedPreset != nil && runType != "" {
		return fmt.Errorf("--run-type cannot be used with --%s", selectedPreset.RunType)
	}

	if prompt == "" && len(args) == 1 && (repo != "" || presetName != "") {
		prompt = args[0]
		args = nil
	}

	// Check if run is being created with flags
	if prompt != "" && (repo != "" || selectedPreset != nil) {
		// Process prompt input (handles @file, -, or literal string)
		processedPrompt, err := utils.ReadPromptInput(prompt)
		if err != nil {
			return fmt.Errorf("failed to process prompt: %w", err)
		}

		// Process context if provided (also supports @file syntax)
		processedContext := contextFlag
		if contextFlag != "" {
			processedContext, err = utils.ReadPromptInput(contextFlag)
			if err != nil {
				return fmt.Errorf("failed to process context: %w", err)
			}
		}

		// Create run from flags
		runConfig := &models.RunConfig{
			Repository:            repo,
			Prompt:                processedPrompt,
			Source:                source,
			Target:                target,
			BaseBranch:            baseBranch,
			OutputMode:            outputMode,
			OutputBranch:          outputBranch,
			PRTargetBranch:        prTargetBranch,
			OutputBranchPolicy:    outputBranchPolicy,
			Title:                 title,
			RunType:               selectedRunType(selectedPreset),
			Context:               processedContext,
			BranchOnly:            branchOnly,
			AcknowledgePromptRisk: acknowledgePromptRisk,
		}

		// Set default run type if not specified
		if runConfig.RunType == "" {
			runConfig.RunType = "run"
		}

		return processSingleRun(runConfig, "")
	}

	// If flags are partially set, show more specific error
	if repo != "" && prompt == "" {
		return fmt.Errorf("missing required flag: --prompt (-p) is required when --repo is specified")
	}
	if prompt != "" && repo == "" && selectedPreset == nil {
		return fmt.Errorf("missing required flag: --repo (-r) is required when --prompt is specified")
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
				fmt.Fprintf(os.Stderr, "%s Ignoring unknown fields in configuration: %s\n", stderrStyle().Info("Note:"), strings.Join(unknownFields, ", "))

				// Show suggestions if available
				suggestions := promptHandler.GetFieldSuggestions()
				for field, suggestion := range suggestions {
					if suggestion != "" {
						fmt.Fprintf(os.Stderr, "      %s Did you mean '%s' instead of '%s'?\n", stderrStyle().Info("Hint:"), suggestion, field)
					}
				}
			}
		}

		applyRunPreset(runConfig, selectedPreset)
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
		if !config.IsBulkRunsEnabled() {
			return bulkRunsUnavailableError()
		}
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
			fmt.Fprintf(os.Stderr, "%s Ignoring unknown fields in configuration: %s\n", stderrStyle().Info("Note:"), strings.Join(unknownFields, ", "))

			// Show suggestions if available
			suggestions := promptHandler.GetFieldSuggestions()
			for field, suggestion := range suggestions {
				if suggestion != "" {
					fmt.Fprintf(os.Stderr, "      %s Did you mean '%s' instead of '%s'?\n", stderrStyle().Info("Hint:"), suggestion, field)
				}
			}
		}
	}

	applyRunPreset(runConfig, selectedPreset)
	return processSingleRun(runConfig, additionalContext)
}

func processSingleRun(runConfig *models.RunConfig, additionalContext string) error {
	if runConfig.Repository == "" {
		container := getContainer()
		gitService := container.GitService()
		if gitService.IsGitRepository() {
			repoName, err := gitService.GetRepositoryName()
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s Could not auto-detect repository: %v\n", stderrStyle().Warning("Warning:"), err)
			} else {
				runConfig.Repository = repoName
				fmt.Printf("%s %s\n", stdoutStyle().Info("Auto-detected repository:"), repoName)
			}
		}
	}

	runConfig.NormalizeBranchOutput()

	// Validate the configuration
	if runConfig.RunType == string(models.RunTypePlan) && !config.IsPlanRunsEnabled() {
		return netstderrors.New(config.PlanRunsUnavailableMessage())
	}

	if err := utils.ValidateRunConfig(runConfig); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Convert to domain request
	createReq := domain.CreateRunRequest{
		Prompt:                runConfig.Prompt,
		RepositoryName:        runConfig.Repository,
		SourceBranch:          runConfig.Source,
		TargetBranch:          runConfig.Target,
		BaseBranch:            runConfig.BaseBranch,
		OutputMode:            runConfig.OutputMode,
		OutputBranch:          runConfig.OutputBranch,
		PRTargetBranch:        runConfig.PRTargetBranch,
		OutputBranchPolicy:    runConfig.OutputBranchPolicy,
		RunType:               runConfig.RunType,
		Agent:                 "opencode",
		OpenCodeModel:         modelForRunType(runConfig.RunType),
		OpenCodeProvider:      providerForRunType(runConfig.RunType),
		Title:                 runConfig.Title,
		Context:               runConfig.Context,
		Files:                 runConfig.Files,
		BranchOnly:            runConfig.BranchOnly,
		AcknowledgePromptRisk: runConfig.AcknowledgePromptRisk,
	}

	// Append additional markdown context if present
	if additionalContext != "" {
		if createReq.Context != "" {
			createReq.Context = createReq.Context + "\n\n" + additionalContext
		} else {
			createReq.Context = additionalContext
		}
	}

	if dryRun {
		fmt.Println(stdoutStyle().Success("Validation successful. Run would be created with:"))
		printRunSelection(createReq)
		b, _ := json.MarshalIndent(createReq, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	// Use service layer to create run
	container := getContainer()
	runService := container.RunService()
	ctx := context.Background()

	printRunSelection(createReq)
	fmt.Println(stdoutStyle().Info("Creating run..."))
	run, err := runService.CreateRun(ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to create run: %s", errors.FormatUserError(err))
	}

	styler := stdoutStyle()
	fmt.Println(styler.Success("Run created successfully!"))
	fmt.Printf("%s %s\n", styler.Label("ID:"), run.ID)
	fmt.Printf("%s %s\n", styler.Label("Status:"), styler.Status(formatStatusForDisplay(run.Status)))
	fmt.Printf("%s %s\n", styler.Label("Repository:"), run.RepositoryName)
	fmt.Printf("%s %s → %s\n", styler.Label("Source:"), run.SourceBranch, run.TargetBranch)

	// Print the RepoBird URL for the run
	runURL := utils.GenerateRepoBirdURL(run.ID)
	if runURL != "" {
		fmt.Printf("%s %s\n", styler.Label("URL:"), styler.URL(runURL))
	}

	if follow {
		fmt.Printf("\n%s\n", styler.Info("Following run status..."))
		return followRunStatus(runService, run.ID)
	}

	return nil
}

func newRunPresetCommand(presetName string) *cobra.Command {
	preset := runPresets[presetName]
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s [prompt]", presetName),
		Short: fmt.Sprintf("Create a %s cloud agent run", preset.Label),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommandWithPreset(cmd, args, presetName)
		},
		SilenceErrors: true,
		SilenceUsage:  false,
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "validate input without creating a run")
	cmd.Flags().BoolVar(&follow, "follow", false, "follow the run status after creation")
	cmd.Flags().StringVarP(&repo, "repo", "r", "", "repository name (owner/repo or numeric ID)")
	cmd.Flags().StringVar(&source, "source", "", "legacy alias for --base-branch")
	cmd.Flags().StringVar(&target, "target", "", "legacy target branch alias")
	cmd.Flags().StringVar(&baseBranch, "base-branch", "", "base branch to start work from (optional)")
	cmd.Flags().StringVar(&outputMode, "output-mode", "", "output mode: 'pr' or 'branch' (optional, default: pr)")
	cmd.Flags().StringVar(&outputBranch, "output-branch", "", "branch to push generated commits to (optional)")
	cmd.Flags().StringVar(&prTargetBranch, "pr-target-branch", "", "branch the pull request targets (optional)")
	cmd.Flags().StringVar(&outputBranchPolicy, "output-branch-policy", "", "output branch policy: 'create' or 'reuse' (optional)")
	cmd.Flags().StringVar(&title, "title", "", "title for the run (optional)")
	cmd.Flags().BoolVar(&branchOnly, "branch-only", false, "push commits to a branch without creating a pull request")
	cmd.Flags().BoolVar(&branchOnly, "no-pr", false, "alias for --branch-only")
	cmd.Flags().BoolVar(&acknowledgePromptRisk, "acknowledge-prompt-risk", false, "acknowledge prompt-risk warning and create the run")
	cmd.Flags().StringVar(&contextFlag, "context", "", "additional context (use @file to read from file, - for stdin)")
	return cmd
}

func resolveRunPreset(presetName string) (*runPreset, error) {
	if basicRun && proRun {
		return nil, fmt.Errorf("--basic and --pro cannot be used together")
	}
	if presetName != "" {
		preset, ok := runPresets[presetName]
		if !ok {
			return nil, fmt.Errorf("unknown run preset: %s", presetName)
		}
		return &preset, nil
	}
	if basicRun {
		preset := runPresets["basic"]
		return &preset, nil
	}
	if proRun {
		preset := runPresets["pro"]
		return &preset, nil
	}
	return nil, nil
}

func selectedRunType(preset *runPreset) string {
	if preset != nil {
		return preset.RunType
	}
	return runType
}

func applyRunPreset(runConfig *models.RunConfig, preset *runPreset) {
	if preset != nil {
		runConfig.RunType = preset.RunType
	}
}

func modelForRunType(runType string) string {
	preset, ok := runPresets[runType]
	if !ok {
		return ""
	}
	return preset.Model
}

func providerForRunType(runType string) string {
	preset, ok := runPresets[runType]
	if !ok {
		return ""
	}
	return preset.Provider
}

func printRunSelection(req domain.CreateRunRequest) {
	preset, ok := runPresets[req.RunType]
	if !ok {
		return
	}
	styler := stdoutStyle()
	fmt.Printf("%s %s\n", styler.Label("Run type:"), preset.Label)
	fmt.Printf("%s %s (%s)\n", styler.Label("Model:"), modelDisplayName(preset.Model), preset.Model)
}

func modelDisplayName(model string) string {
	switch model {
	case "openrouter/deepseek/deepseek-v4-flash":
		return "DeepSeek V4 Flash"
	case "openrouter/moonshotai/kimi-k2.6":
		return "Kimi K2.6"
	default:
		return model
	}
}

func processBulkRuns(filename string) error {
	if !config.IsBulkRunsEnabled() {
		return bulkRunsUnavailableError()
	}

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
		styler := stdoutStyle()
		fmt.Println(styler.Success("✓ Configuration valid"))
		fmt.Printf("%s %s\n", styler.Label("Repository:"), bulkConfig.Repository)
		fmt.Printf("%s %s\n", styler.Label("Source:"), bulkConfig.Source)
		fmt.Printf("%s %s\n", styler.Label("RunType:"), bulkConfig.RunType)
		fmt.Printf("%s %d\n", styler.Label("Total runs:"), len(bulkConfig.Runs))
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

// formatStatusForDisplay converts domain status to uppercase display format
func formatStatusForDisplay(status string) string {
	switch status {
	case domain.StatusCompleted:
		return "DONE"
	case domain.StatusQueued:
		return "QUEUED"
	case domain.StatusRunning:
		return "PROCESSING"
	case domain.StatusFailed:
		return "FAILED"
	case domain.StatusCancelled:
		return "CANCELLED"
	case domain.StatusCreated:
		return "CREATED"
	default:
		// Return uppercase version of unknown statuses
		return strings.ToUpper(status)
	}
}

func followRunStatus(runService domain.RunService, runID string) error {
	ctx := context.Background()
	startTime := time.Now()
	lastStatus := ""
	isTTY := stdoutIsTerminal()

	// Check if debug is enabled via environment variable or flag
	isDebug := debug || os.Getenv("REPOBIRD_DEBUG_LOG") == "1"

	callback := func(status string, message string) {
		displayStatus := formatStatusForDisplay(status)
		if displayStatus != lastStatus {
			clearLiveOutput(os.Stdout, isTTY)
			fmt.Printf("[%s] %s %s\n", time.Now().Format("15:04:05"), stdoutStyle().Label("Status:"), stdoutStyle().Status(displayStatus))
			lastStatus = displayStatus
			return
		}
		if !isTTY {
			return
		}

		elapsed := time.Since(startTime)
		if message != "" {
			printLiveUpdate(os.Stdout, isTTY, "[%s] %s - %s", formatDuration(elapsed), displayStatus, message)
		} else {
			printLiveUpdate(os.Stdout, isTTY, "[%s] %s", formatDuration(elapsed), displayStatus)
		}
	}

	finalRun, err := runService.WaitForCompletion(ctx, runID, callback)
	if err != nil {
		return fmt.Errorf("failed to follow run status: %s", errors.FormatUserError(err))
	}

	if isDebug {
		fmt.Printf("DEBUG: WaitForCompletion returned - Status: %s, PullRequestURL: '%s'\n", finalRun.Status, finalRun.PullRequestURL)
	}

	// If run completed successfully, fetch full details to get PR URL
	// Use a new context with its own timeout to avoid interference
	if finalRun.Status == domain.StatusCompleted {
		if isDebug {
			fmt.Printf("DEBUG: Run completed, fetching full details for PR URL...\n")
		}

		// Create a new context with a 10-second timeout for fetching full details
		detailsCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Wait a brief moment for API to update PR URL
		time.Sleep(1 * time.Second)

		// Fetch full run details
		if fullRun, err := runService.GetRun(detailsCtx, runID); err == nil {
			if isDebug {
				fmt.Printf("DEBUG: Fetched full details - Status: %s, PullRequestURL: '%s'\n", fullRun.Status, fullRun.PullRequestURL)
			}
			finalRun = fullRun
		} else {
			if isDebug {
				fmt.Printf("DEBUG: Failed to fetch full details: %v\n", err)
			}
		}
		// If fetch fails, we still have the basic run info from polling
	}

	clearLiveOutput(os.Stdout, isTTY)
	if finalRun.Status == domain.StatusFailed && finalRun.Error != "" {
		fmt.Printf("%s %s\n", stdoutStyle().Error("Run failed:"), finalRun.Error)
	} else {
		styler := stdoutStyle()
		fmt.Printf("%s %s\n", styler.Success("Run completed with status:"), styler.Status(formatStatusForDisplay(finalRun.Status)))

		// Display PR URL if run completed successfully and URL is available
		if finalRun.Status == domain.StatusCompleted && finalRun.PullRequestURL != "" {
			fmt.Printf("%s %s\n", styler.Label("Pull Request:"), styler.URL(finalRun.PullRequestURL))
		} else if isDebug && finalRun.Status == domain.StatusCompleted {
			fmt.Printf("DEBUG: Run completed but no PR URL available (PullRequestURL='%s')\n", finalRun.PullRequestURL)
		}
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
	if !config.IsBulkRunsEnabled() {
		return bulkRunsUnavailableError()
	}

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
	styler := stdoutStyle()
	fmt.Println(styler.Heading("Submitting bulk runs..."))
	fmt.Printf("%s %s\n", styler.Label("Repository:"), bulkConfig.Repository)
	fmt.Printf("%s %d\n", styler.Label("Total runs:"), len(bulkConfig.Runs))
	fmt.Println("\nThis may take up to 5 minutes. Please wait...")

	done := make(chan bool, 1) // Buffered to prevent goroutine leak

	if stdoutIsTerminal() {
		// Show a progress indicator with elapsed time
		startTime := time.Now()
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinnerIdx := 0

		// Start spinner in background
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					clearLiveOutput(os.Stdout, true)
					return
				case <-ticker.C:
					elapsed := time.Since(startTime)
					printLiveUpdate(os.Stdout, true, "%s Processing... (%.0fs)", spinner[spinnerIdx], elapsed.Seconds())
					_ = os.Stdout.Sync() // Force flush to ensure animation
					spinnerIdx = (spinnerIdx + 1) % len(spinner)
				}
			}
		}()
	}

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
		fmt.Println("\n" + styler.Warning("⚠ Bulk submission in progress:"))
		fmt.Printf("The server is still processing your runs. This is normal for large batches.\n")
		fmt.Printf("Created: %d/%d runs so far\n", bulkResp.Data.Metadata.TotalSuccessful, bulkResp.Data.Metadata.TotalRequested)

		if len(bulkResp.Data.Failed) > 0 {
			fmt.Println("\n" + styler.Error("Failed runs:"))
			for _, runErr := range bulkResp.Data.Failed {
				fmt.Printf("  %s Run %d: %s\n", styler.Error("✗"), runErr.RequestIndex+1, runErr.Message)
			}
		}

		fmt.Println("\n" + styler.Info("ℹ  The remaining runs are being processed in the background."))
		fmt.Println("Use --follow or check status to monitor progress.")
	} else if len(bulkResp.Data.Failed) > 0 {
		// Some runs failed
		fmt.Println("\n" + styler.Warning("⚠ Partial success:"))
		fmt.Printf("Created: %d/%d runs\n", bulkResp.Data.Metadata.TotalSuccessful, bulkResp.Data.Metadata.TotalRequested)

		// Check if failures are due to duplicates
		for _, runErr := range bulkResp.Data.Failed {
			fmt.Printf("  %s Run %d: %s\n", styler.Error("✗"), runErr.RequestIndex+1, runErr.Message)
			// Note: Duplicates are no longer blocked as --force is deprecated
		}
	} else {
		// All runs created successfully
		fmt.Println("\n" + styler.Success("✓ All runs created successfully"))
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
		fmt.Println("\n" + styler.Info("Following batch progress..."))
		// Create context with 1h 30m timeout
		followCtx, cancel := context.WithTimeout(context.Background(), 90*time.Minute)
		defer cancel()
		return followBulkProgress(followCtx, client, bulkResp.Data.BatchID)
	}

	fmt.Printf("\n%s %s\n", styler.Label("Batch ID:"), bulkResp.Data.BatchID)
	fmt.Println("Use 'repobird bulk status " + bulkResp.Data.BatchID + "' to check progress")

	return nil
}
