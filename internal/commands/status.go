package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
)

var (
	statusAll    bool
	statusLimit  int
	statusFollow bool
	statusJSON   bool
)

var statusCmd = &cobra.Command{
	Use:   "status [run-id]",
	Short: "Check the status of runs",
	Long: `Check the status of a specific run or list all runs.
If no run ID is provided, lists recent runs.`,
	Args: cobra.MaximumNArgs(1),
	RunE: statusCommand,
}

func init() {
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "list all runs")
	statusCmd.Flags().IntVar(&statusLimit, "limit", 10, "number of runs to display")
	statusCmd.Flags().BoolVar(&statusFollow, "follow", false, "follow run status with polling")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output in JSON format")
}

func statusCommand(_ *cobra.Command, args []string) error {
	if cfg.APIKey == "" {
		return fmt.Errorf("API key not configured. Set REPOBIRD_API_KEY or run 'repobird config set api-key'")
	}

	client := api.NewClient(cfg.APIKey, cfg.APIURL, cfg.Debug)

	if len(args) > 0 {
		return getRunStatus(client, args[0])
	}

	return listRuns(client)
}

func getRunStatus(client *api.Client, runID string) error {
	if statusFollow {
		return followSingleRun(client, runID)
	}

	run, err := client.GetRun(runID)
	if err != nil {
		return fmt.Errorf("failed to get run status: %w", err)
	}

	if statusJSON {
		b, _ := json.MarshalIndent(run, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	printRunDetails(run)
	return nil
}

func listRuns(client *api.Client) error {
	userInfo, err := client.VerifyAuth()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch user info: %v\n", err)
	} else {
		fmt.Printf("Remaining runs: %d/%d (%s tier)\n\n", userInfo.RemainingRuns, userInfo.TotalRuns, userInfo.Tier)
	}

	runs, err := client.ListRuns(statusLimit, 0)
	if err != nil {
		return fmt.Errorf("failed to list runs: %w", err)
	}

	if len(runs) == 0 {
		fmt.Println("No runs found")
		return nil
	}

	if statusJSON {
		b, _ := json.MarshalIndent(runs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSTATUS\tREPOSITORY\tCREATED\tTITLE")
	fmt.Fprintln(w, "──\t──────\t──────────\t───────\t─────")

	for _, run := range runs {
		created := run.CreatedAt.Format("2006-01-02 15:04")
		title := run.Title
		if title == "" {
			title = truncate(run.Prompt, 30)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			run.ID[:8],
			run.Status,
			run.Repository,
			created,
			title,
		)
	}

	w.Flush()
	return nil
}

func followSingleRun(client *api.Client, runID string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	run, err := client.GetRun(runID)
	if err != nil {
		return fmt.Errorf("failed to get run status: %w", err)
	}

	printRunDetails(run)

	if run.Status == models.StatusDone || run.Status == models.StatusFailed {
		return nil
	}

	fmt.Println("\nFollowing run status...")

	for range ticker.C {
		run, err := client.GetRun(runID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking status: %v\n", err)
			continue
		}

		fmt.Printf("[%s] Status: %s\n", time.Now().Format("15:04:05"), run.Status)

		if run.Status == models.StatusDone || run.Status == models.StatusFailed {
			fmt.Println("\nFinal status:")
			printRunDetails(run)
			return nil
		}
	}
	return nil
}

func printRunDetails(run *models.RunResponse) {
	fmt.Printf("Run ID: %s\n", run.ID)
	fmt.Printf("Status: %s\n", run.Status)
	fmt.Printf("Repository: %s\n", run.Repository)
	fmt.Printf("Branch: %s → %s\n", run.Source, run.Target)
	if run.Title != "" {
		fmt.Printf("Title: %s\n", run.Title)
	}
	fmt.Printf("Created: %s\n", run.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated: %s\n", run.UpdatedAt.Format("2006-01-02 15:04:05"))
	if run.Error != "" {
		fmt.Printf("Error: %s\n", run.Error)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
