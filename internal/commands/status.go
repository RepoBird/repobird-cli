// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/services"
	"github.com/repobird/repobird-cli/internal/utils"
	"github.com/repobird/repobird-cli/pkg/version"
)

var (
	statusAll    bool
	statusLimit  int
	statusFollow bool
	statusJSON   bool
)

var statusCmd = &cobra.Command{
	Use:     "status [run-id]",
	Aliases: []string{"st"},
	Short:   "Check the status of runs",
	Long: `Check the status of a specific run or list all runs.
If no run ID is provided, lists recent runs.`,
	Args: cobra.MaximumNArgs(1),
	RunE: statusCommand,
}

//nolint:gochecknoinits // Required for CLI command registration
func init() {
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "list all runs")
	statusCmd.Flags().IntVar(&statusLimit, "limit", 10, "number of runs to display")
	statusCmd.Flags().BoolVar(&statusFollow, "follow", false, "follow run status with polling")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output in JSON format")
}

func statusCommand(_ *cobra.Command, args []string) error {
	if cfg.APIKey == "" {
		return errors.NoAPIKeyError()
	}

	apiURL := utils.GetAPIURL(cfg.APIURL)
	client := api.NewClient(cfg.APIKey, apiURL, cfg.Debug)

	if len(args) > 0 {
		return getRunStatus(client, args[0])
	}

	return listRuns(client)
}

func getRunStatus(client *api.Client, runID string) error {
	if statusFollow {
		return followSingleRun(client, runID)
	}

	ctx := context.Background()
	run, err := client.GetRunWithRetry(ctx, runID)
	if err != nil {
		return fmt.Errorf("failed to get run status: %s", errors.FormatUserError(err))
	}

	if statusJSON || jsonOutput {
		b, _ := json.MarshalIndent(run, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	printRunDetails(run)
	return nil
}

func listRuns(client *api.Client) error {
	styler := stdoutStyle()
	// Always show version info in dev/debug mode or when there's an error
	env := os.Getenv("REPOBIRD_ENV")
	showDebugInfo := strings.ToLower(env) == "dev" || strings.ToLower(env) == "development" || cfg.Debug

	// Try to verify auth first to check for API/auth errors
	userInfo, authErr := client.VerifyAuth()

	// If API/auth error, show version info and error, then exit
	if authErr != nil && (errors.IsAuthError(authErr) || errors.IsNetworkError(authErr)) {
		// Always show version/debug info when there's an API error
		fmt.Printf("%s %s", styler.Label("Build:"), version.GetVersion())
		if version.GetVersion() == "dev" {
			fmt.Printf(" (development)")
		}
		fmt.Printf(" | %s %s", styler.Label("Commit:"), version.GitCommit)
		if cfg.Debug {
			fmt.Printf(" | %s %s", styler.Label("Debug:"), styler.Warning("ON"))
		}
		fmt.Println()
		fmt.Println()

		// Show the error below version info
		fmt.Fprintf(os.Stderr, "%s %s\n", stderrStyle().Error("Error:"), errors.FormatUserError(authErr))
		return nil // Return nil to prevent cobra from showing usage and error again
	}

	// Show version info in dev/debug mode for successful requests
	if showDebugInfo {
		fmt.Printf("%s %s", styler.Label("Build:"), version.GetVersion())
		if version.GetVersion() == "dev" {
			fmt.Printf(" (development)")
		}
		fmt.Printf(" | %s %s", styler.Label("Commit:"), version.GitCommit)
		if cfg.Debug {
			fmt.Printf(" | %s %s", styler.Label("Debug:"), styler.Warning("ON"))
		}
		fmt.Println()
		fmt.Println()
	}

	// If auth succeeded but had a warning-level error, show warning
	if authErr != nil {
		fmt.Fprintf(os.Stderr, "%s Could not fetch user info: %s\n", stderrStyle().Warning("Warning:"), errors.FormatUserError(authErr))
	} else if userInfo != nil {
		// Set the current user for cache initialization and show user info
		services.SetCurrentUser(userInfo)

		printStatusAccountUsage(userInfo)
		fmt.Println()
	}

	runs, err := client.ListRunsLegacy(statusLimit, 0)
	if err != nil {
		// If this is also an API/auth error and we haven't shown version info yet, show it
		if !showDebugInfo && (errors.IsAuthError(err) || errors.IsNetworkError(err)) {
			fmt.Printf("%s %s", styler.Label("Build:"), version.GetVersion())
			if version.GetVersion() == "dev" {
				fmt.Printf(" (development)")
			}
			fmt.Printf(" | %s %s", styler.Label("Commit:"), version.GitCommit)
			if cfg.Debug {
				fmt.Printf(" | %s %s", styler.Label("Debug:"), styler.Warning("ON"))
			}
			fmt.Println()
			fmt.Println()

			fmt.Fprintf(os.Stderr, "%s failed to list runs: %s\n", stderrStyle().Error("Error:"), errors.FormatUserError(err))
			return nil // Return nil to prevent cobra from showing usage and error again
		}
		return fmt.Errorf("failed to list runs: %s", errors.FormatUserError(err))
	}

	if len(runs) == 0 {
		fmt.Println(styler.Muted("No runs found"))
		return nil
	}

	if statusJSON || jsonOutput {
		b, _ := json.MarshalIndent(runs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tSTATUS\tREPOSITORY\tCREATED\tTITLE")
	_, _ = fmt.Fprintln(w, "──\t──────\t──────────\t───────\t─────")

	for _, run := range runs {
		created := run.CreatedAt.Format("2006-01-02 15:04")
		title := run.Title
		if title == "" {
			title = truncate(run.Prompt, 30)
		}
		idStr := run.GetIDString()
		if len(idStr) > 8 {
			idStr = idStr[:8]
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			idStr,
			styler.Status(string(run.Status)),
			run.GetRepositoryName(),
			created,
			title,
		)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush output: %w", err)
	}
	return nil
}

func followSingleRun(client *api.Client, runID string) error {
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
			printRunDetails(run)
			lastStatus = string(run.Status)
			fmt.Printf("\n%s\n", stdoutStyle().Info("Following run status..."))
		} else {
			utils.ShowPollingProgress(startTime, string(run.Status), run.Error)
		}
	}

	finalRun, err := poller.Poll(ctx, pollFunc, onUpdate)
	if err != nil {
		return fmt.Errorf("failed to follow run status: %s", errors.FormatUserError(err))
	}

	utils.ClearLine()
	fmt.Printf("\n%s\n", stdoutStyle().Heading("Final status:"))
	printRunDetails(finalRun)
	return nil
}

func printRunDetails(run *models.RunResponse) {
	styler := stdoutStyle()
	fmt.Printf("%s %s\n", styler.Label("Run ID:"), run.GetIDString())
	fmt.Printf("%s %s\n", styler.Label("Status:"), styler.Status(string(run.Status)))
	fmt.Printf("%s %s\n", styler.Label("Repository:"), run.Repository)
	fmt.Printf("%s %s → %s\n", styler.Label("Branch:"), run.Source, run.Target)
	if run.Title != "" {
		fmt.Printf("%s %s\n", styler.Label("Title:"), run.Title)
	}
	fmt.Printf("%s %s\n", styler.Label("Created:"), run.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("%s %s\n", styler.Label("Updated:"), run.UpdatedAt.Format("2006-01-02 15:04:05"))
	if run.Error != "" {
		fmt.Printf("%s %s\n", styler.Error("Error:"), run.Error)
	}
}

// truncate is now replaced by utils.TruncateSimple
// Keeping this as an alias for backward compatibility
func truncate(s string, maxLen int) string {
	return utils.TruncateSimple(s, maxLen)
}
