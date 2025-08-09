package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/utils"
	gitutils "github.com/repobird/repobird-cli/pkg/utils"
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

	if runReq.Repository == "" {
		repo, err := gitutils.DetectRepository()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not auto-detect repository: %v\n", err)
		} else {
			runReq.Repository = repo
			fmt.Printf("Auto-detected repository: %s\n", repo)
		}
	}

	if runReq.Source == "" {
		branch, err := gitutils.GetCurrentBranch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not auto-detect branch: %v\n", err)
		} else {
			runReq.Source = branch
			fmt.Printf("Auto-detected source branch: %s\n", branch)
		}
	}

	if dryRun {
		fmt.Println("Validation successful. Run would be created with:")
		b, _ := json.MarshalIndent(runReq, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	client := api.NewClient(cfg.APIKey, cfg.APIURL, cfg.Debug)

	fmt.Println("Creating run...")
	ctx := context.Background()
	runResp, err := client.CreateRunWithRetry(ctx, &runReq)
	if err != nil {
		return fmt.Errorf("failed to create run: %s", errors.FormatUserError(err))
	}

	fmt.Printf("Run created successfully!\n")
	fmt.Printf("ID: %s\n", runResp.GetIDString())
	fmt.Printf("Status: %s\n", runResp.Status)
	fmt.Printf("Repository: %s\n", runResp.Repository)
	fmt.Printf("Source: %s → Target: %s\n", runResp.Source, runResp.Target)

	if follow {
		fmt.Println("\nFollowing run status...")
		return followRunStatus(client, runResp.GetIDString())
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

func followRunStatus(client *api.Client, runID string) error {
	ctx := context.Background()
	config := utils.DefaultPollConfig()
	config.Debug = cfg.Debug
	poller := utils.NewPoller(config)

	startTime := time.Now()
	lastStatus := ""

	pollFunc := func(ctx context.Context) (*models.RunResponse, error) {
		return client.GetRunWithRetry(ctx, runID)
	}

	onUpdate := func(run *models.RunResponse) {
		if string(run.Status) != lastStatus {
			utils.ClearLine()
			fmt.Printf("[%s] Status: %s\n", time.Now().Format("15:04:05"), run.Status)
			lastStatus = string(run.Status)
		} else {
			utils.ShowPollingProgress(startTime, string(run.Status), run.Error)
		}
	}

	finalRun, err := poller.Poll(ctx, pollFunc, onUpdate)
	if err != nil {
		return fmt.Errorf("failed to follow run status: %s", errors.FormatUserError(err))
	}

	utils.ClearLine()
	if finalRun.Status == models.StatusFailed && finalRun.Error != "" {
		fmt.Printf("Run failed: %s\n", finalRun.Error)
	} else {
		fmt.Printf("Run completed with status: %s\n", finalRun.Status)
	}
	return nil
}
