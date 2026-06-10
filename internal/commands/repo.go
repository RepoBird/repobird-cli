// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/utils"
)

type repositoryDefaultsOptions struct {
	baseBranch          *string
	prTargetBranch      *string
	outputBranch        *string
	clearBaseBranch     bool
	clearPRTargetBranch bool
	clearOutputBranch   bool
	json                bool
}

var repoCmd = newRepoCommand()

func newRepoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repo",
		Aliases: []string{"repos", "repository", "repositories"},
		Short:   "Manage connected repositories",
		Long:    "Manage connected repositories and persisted repository branch defaults.",
	}

	cmd.AddCommand(newRepoListCommand())
	cmd.AddCommand(newRepoShowCommand())
	cmd.AddCommand(newRepoDefaultsCommand())
	return cmd
}

func newRepoListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List connected repositories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := newRepoAPIClient()
			if err != nil {
				return err
			}
			repos, err := client.ListRepositories(context.Background())
			if err != nil {
				return fmt.Errorf("failed to list repositories: %s", errors.FormatUserError(err))
			}
			if jsonOutput {
				return printJSON(cmd.OutOrStdout(), repos)
			}
			printRepositoryList(cmd.OutOrStdout(), repos)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	return cmd
}

func newRepoShowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <repo-id>",
		Short: "Show repository details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := newRepoAPIClient()
			if err != nil {
				return err
			}
			repo, err := client.GetRepository(args[0])
			if err != nil {
				return fmt.Errorf("failed to get repository: %s", errors.FormatUserError(err))
			}
			if jsonOutput {
				return printJSON(cmd.OutOrStdout(), repo)
			}
			printRepositoryDetails(cmd.OutOrStdout(), repo)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	return cmd
}

func newRepoDefaultsCommand() *cobra.Command {
	var baseBranch string
	var prTargetBranch string
	var outputBranch string
	var opts repositoryDefaultsOptions

	cmd := &cobra.Command{
		Use:   "defaults <repo-id>",
		Short: "Set or clear repository branch defaults",
		Long: `Set or clear persisted repository branch defaults.

Per-run branch flags on "repobird run" still override these repository defaults.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("base") {
				opts.baseBranch = &baseBranch
			}
			if cmd.Flags().Changed("pr-target") {
				opts.prTargetBranch = &prTargetBranch
			}
			if cmd.Flags().Changed("output") {
				opts.outputBranch = &outputBranch
			}

			client, err := newRepoAPIClient()
			if err != nil {
				return err
			}
			update := buildRepositoryDefaultsUpdate(opts)
			repo, err := client.UpdateRepositoryDefaults(args[0], update)
			if err != nil {
				return fmt.Errorf("failed to update repository defaults: %s", errors.FormatUserError(err))
			}
			if opts.json {
				return printJSON(cmd.OutOrStdout(), repo)
			}
			printRepositoryDetails(cmd.OutOrStdout(), repo)
			return nil
		},
	}
	cmd.Flags().StringVar(&baseBranch, "base", "", "default base branch for new runs")
	cmd.Flags().StringVar(&prTargetBranch, "pr-target", "", "default pull request target branch")
	cmd.Flags().StringVar(&outputBranch, "output", "", "default branch-only output branch; blank clears to generated")
	cmd.Flags().BoolVar(&opts.clearBaseBranch, "clear-base", false, "clear the default base branch")
	cmd.Flags().BoolVar(&opts.clearPRTargetBranch, "clear-pr-target", false, "clear the default pull request target branch")
	cmd.Flags().BoolVar(&opts.clearOutputBranch, "clear-output", false, "clear the default output branch so branch-only runs generate one")
	cmd.Flags().BoolVar(&opts.json, "json", false, "output in JSON format")
	return cmd
}

func newRepoAPIClient() (*api.Client, error) {
	secureConfig := cfg
	if secureConfig == nil {
		loaded, err := config.LoadSecureConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		secureConfig = loaded
	}
	if secureConfig.APIKey == "" {
		return nil, errors.NoAPIKeyError()
	}
	apiURL := utils.GetAPIURL(secureConfig.APIURL)
	return api.NewClient(secureConfig.APIKey, apiURL, secureConfig.Debug), nil
}

func buildRepositoryDefaultsUpdate(opts repositoryDefaultsOptions) models.RepositoryDefaultsUpdate {
	update := models.RepositoryDefaultsUpdate{
		DefaultBaseBranch:        opts.baseBranch,
		DefaultPRTargetBranch:    opts.prTargetBranch,
		ClearDefaultBaseBranch:   opts.clearBaseBranch,
		ClearDefaultPRTarget:     opts.clearPRTargetBranch,
		ClearDefaultOutputBranch: opts.clearOutputBranch,
	}
	if opts.outputBranch != nil {
		if *opts.outputBranch == "" {
			update.ClearDefaultOutputBranch = true
		} else {
			update.DefaultOutputBranch = opts.outputBranch
		}
	}
	return update
}

func printRepositoryList(out io.Writer, repos []models.APIRepository) {
	styler := styleFor(out)
	if len(repos) == 0 {
		_, _ = fmt.Fprintln(out, styler.Muted("No repositories found"))
		return
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tREPOSITORY\tDEFAULT\tBASE\tPR TARGET\tOUTPUT")
	for _, repo := range repos {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			repo.ID,
			repo.FullName(),
			displayDefault(repo.DefaultBranch, "(unknown)"),
			displayStringPtr(repo.DefaultBaseBranch, "(repo default)"),
			displayStringPtr(repo.DefaultPRTargetBranch, "(base branch)"),
			displayStringPtr(repo.DefaultOutputBranch, "(generated)"),
		)
	}
	_ = w.Flush()
}

func printRepositoryDetails(out io.Writer, repo *models.APIRepository) {
	styler := styleFor(out)
	_, _ = fmt.Fprintf(out, "%s %s\n", styler.Label("Repository:"), repo.FullName())
	_, _ = fmt.Fprintf(out, "%s %d\n", styler.Label("ID:"), repo.ID)
	_, _ = fmt.Fprintf(out, "%s %s\n", styler.Label("Default Branch:"), displayDefault(repo.DefaultBranch, "(unknown)"))
	_, _ = fmt.Fprintf(out, "%s %s\n", styler.Label("Default Base Branch:"), displayStringPtr(repo.DefaultBaseBranch, "(repo default)"))
	_, _ = fmt.Fprintf(out, "%s %s\n", styler.Label("Default PR Target Branch:"), displayStringPtr(repo.DefaultPRTargetBranch, "(base branch)"))
	_, _ = fmt.Fprintf(out, "%s %s\n", styler.Label("Default Output Branch:"), displayStringPtr(repo.DefaultOutputBranch, "(generated)"))
}

func displayStringPtr(value *string, fallback string) string {
	if value == nil || *value == "" {
		return fallback
	}
	return *value
}

func displayDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func printJSON(out io.Writer, value interface{}) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, string(b))
	return err
}
