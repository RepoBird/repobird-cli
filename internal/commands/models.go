// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/utils"
)

func newModelsCommand() *cobra.Command {
	list := &cobra.Command{
		Use:   "list",
		Short: "List models available for RepoBird runs",
		Args:  cobra.NoArgs,
		RunE:  listModelsCommand,
	}
	list.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")

	cmd := &cobra.Command{
		Use:   "models",
		Short: "Discover server-authoritative models",
		Args:  cobra.NoArgs,
		RunE:  listModelsCommand,
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	cmd.AddCommand(list)
	return cmd
}

func listModelsCommand(cmd *cobra.Command, _ []string) error {
	client, err := newModelsAPIClient()
	if err != nil {
		return err
	}

	result, err := client.GetModels("")
	if err != nil {
		return fmt.Errorf("failed to list models: %s", errors.FormatUserError(err))
	}
	if result.NotModified || result.Catalog == nil {
		return fmt.Errorf("failed to list models: server returned no catalog")
	}

	if jsonOutput {
		return printJSON(cmd.OutOrStdout(), result.Catalog)
	}
	printModelsHuman(cmd.OutOrStdout(), result.Catalog)
	return nil
}

func newModelsAPIClient() (*api.Client, error) {
	secureConfig := cfg
	if secureConfig == nil {
		loaded, err := config.LoadSecureConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		secureConfig = loaded
	}

	var apiKey, apiURL string
	var debugEnabled bool
	if secureConfig != nil && secureConfig.Config != nil {
		apiKey = secureConfig.APIKey
		apiURL = secureConfig.APIURL
		debugEnabled = secureConfig.Debug
	}
	return api.NewClient(apiKey, utils.GetAPIURL(apiURL), debugEnabled), nil
}

func printModelsHuman(out io.Writer, response *dto.PublicModelsResponse) {
	if response == nil || response.Data == nil || len(response.Data.Models) == 0 {
		_, _ = fmt.Fprintln(out, "No models found")
		return
	}

	_, _ = fmt.Fprintf(out, "Catalog revision: %s\n", response.Data.CatalogRevision)
	if details := formatAdditionalFields(response.Data.AdditionalFields); details != "" {
		_, _ = fmt.Fprintf(out, "Catalog fields: %s\n", details)
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tLABEL\tPROVIDER\tCAPABILITIES\tDEFAULT\tDETAILS")
	for _, model := range response.Data.Models {
		defaultValue := "no"
		if model.IsDefault {
			defaultValue = "yes"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			model.ID,
			model.Label,
			model.Provider,
			strings.Join(model.Capabilities, ","),
			defaultValue,
			formatAdditionalFields(model.AdditionalFields),
		)
	}
	_ = w.Flush()
}

func formatAdditionalFields(fields map[string]json.RawMessage) string {
	if len(fields) == 0 {
		return ""
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return ""
	}
	return string(encoded)
}
