// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"

	"github.com/repobird/repobird-cli/internal/config"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation",
	Long: fmt.Sprintf(`Generate documentation for the RepoBird CLI and TUI - trigger AI coding agents, submit batch runs, and monitor your AI agent runs through an interactive dashboard.

Base URL: %s
Get API Key: %s`, config.GetURLs().BaseURL, config.GetAPIKeysURL()),
}

var manCmd = &cobra.Command{
	Use:   "man [output-dir]",
	Short: "Generate man pages",
	Long: fmt.Sprintf(`Generate man pages for RepoBird CLI commands.

The 'run' command supports both single and bulk configurations in JSON, YAML, and Markdown formats.
Use 'repobird examples' to see configuration formats and generate example files.

Get API Key: %s`, config.GetAPIKeysURL()),
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "man"
		if len(args) > 0 {
			outputDir = args[0]
		}

		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		header := &doc.GenManHeader{
			Title:   "REPOBIRD",
			Section: "1",
			Manual:  "RepoBird CLI Manual",
			Source:  "RepoBird.ai - " + config.GetAPIKeysURL(),
		}

		if err := doc.GenManTree(rootCmd, header, outputDir); err != nil {
			return fmt.Errorf("failed to generate man pages: %w", err)
		}

		// Add aliases to the generated man pages
		if err := addAliasesToMan(outputDir); err != nil {
			return fmt.Errorf("failed to add aliases to man pages: %w", err)
		}

		fmt.Printf("✓ Man pages generated in %s directory\n", outputDir)
		fmt.Println("Tip: The 'run' command supports both single and bulk configurations. Use 'repobird examples' for format details.")
		return nil
	},
}

var markdownCmd = &cobra.Command{
	Use:   "markdown [output-dir]",
	Short: "Generate markdown documentation",
	Long: fmt.Sprintf(`Generate markdown documentation for RepoBird CLI commands.

The 'run' command supports both single and bulk configurations in JSON, YAML, and Markdown formats.
Use 'repobird examples' to see configuration formats and generate example files.

Get API Key: %s`, config.GetAPIKeysURL()),
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "docs/generated"
		if len(args) > 0 {
			outputDir = args[0]
		}

		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := doc.GenMarkdownTree(rootCmd, outputDir); err != nil {
			return fmt.Errorf("failed to generate markdown docs: %w", err)
		}

		// Add aliases to the generated markdown files
		if err := addAliasesToMarkdown(outputDir); err != nil {
			return fmt.Errorf("failed to add aliases to markdown docs: %w", err)
		}

		fmt.Printf("✓ Markdown documentation generated in %s directory\n", outputDir)
		fmt.Println("Tip: The 'run' command supports both single and bulk configurations. Use 'repobird examples' for format details.")
		return nil
	},
}

var yamlCmd = &cobra.Command{
	Use:   "yaml [output-dir]",
	Short: "Generate YAML documentation",
	Long: fmt.Sprintf(`Generate YAML documentation for RepoBird CLI commands.

The 'run' command supports both single and bulk configurations in JSON, YAML, and Markdown formats.
Use 'repobird examples' to see configuration formats and generate example files.

Get API Key: %s`, config.GetAPIKeysURL()),
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "yaml"
		if len(args) > 0 {
			outputDir = args[0]
		}

		if err := os.MkdirAll(outputDir, 0750); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := doc.GenYamlTree(rootCmd, outputDir); err != nil {
			return fmt.Errorf("failed to generate YAML docs: %w", err)
		}

		// Add aliases to the generated YAML files
		if err := addAliasesToYAML(outputDir); err != nil {
			return fmt.Errorf("failed to add aliases to YAML docs: %w", err)
		}

		fmt.Printf("✓ YAML documentation generated in %s directory\n", outputDir)
		fmt.Println("Tip: The 'run' command supports both single and bulk configurations. Use 'repobird examples' for format details.")
		return nil
	},
}

func init() {
	docsCmd.AddCommand(manCmd)
	docsCmd.AddCommand(markdownCmd)
	docsCmd.AddCommand(yamlCmd)
	rootCmd.AddCommand(docsCmd)
}

// addAliasesToMarkdown post-processes markdown files to add alias information
func addAliasesToMarkdown(outputDir string) error {
	// Walk through all generated markdown files
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .md files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Convert to string for processing
		contentStr := string(content)

		// Add aliases for specific commands
		if strings.Contains(path, "repobird_status.md") {
			contentStr = strings.Replace(contentStr,
				"```\nrepobird status [run-id] [flags]\n```",
				"```\nrepobird status [run-id] [flags]\n```\n\n### Aliases\n\n```\nrepobird st [run-id] [flags]\n```",
				1)
		}

		if strings.Contains(path, "repobird_version.md") {
			contentStr = strings.Replace(contentStr,
				"```\nrepobird version [flags]\n```",
				"```\nrepobird version [flags]\n```\n\n### Aliases\n\n```\nrepobird v [flags]\n```",
				1)
		}

		// Write the updated content back
		return os.WriteFile(path, []byte(contentStr), 0644)
	})
}

// addAliasesToYAML post-processes YAML files to add alias information
func addAliasesToYAML(outputDir string) error {
	// Walk through all generated YAML files
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .yaml files
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Convert to string for processing
		contentStr := string(content)

		// Add aliases for specific commands
		if strings.Contains(path, "repobird_status.yaml") {
			contentStr = strings.Replace(contentStr,
				"usage: repobird status [run-id] [flags]",
				"usage: repobird status [run-id] [flags]\naliases: [st]",
				1)
		}

		if strings.Contains(path, "repobird_version.yaml") {
			contentStr = strings.Replace(contentStr,
				"usage: repobird version [flags]",
				"usage: repobird version [flags]\naliases: [v]",
				1)
		}

		// Write the updated content back
		return os.WriteFile(path, []byte(contentStr), 0644)
	})
}

// addAliasesToMan post-processes man pages to add alias information
func addAliasesToMan(outputDir string) error {
	// Walk through all generated man files
	return filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .1 files (man pages)
		if !strings.HasSuffix(path, ".1") {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Convert to string for processing
		contentStr := string(content)

		// Add aliases for specific commands
		if strings.Contains(path, "repobird-status.1") {
			contentStr = strings.Replace(contentStr,
				".SH SYNOPSIS",
				".SH SYNOPSIS\n.PP\n\\fBrepobird st\\fP [run-id] [flags] (alias)\n",
				1)
		}

		if strings.Contains(path, "repobird-version.1") {
			contentStr = strings.Replace(contentStr,
				".SH SYNOPSIS",
				".SH SYNOPSIS\n.PP\n\\fBrepobird v\\fP [flags] (alias)\n",
				1)
		}

		// Write the updated content back
		return os.WriteFile(path, []byte(contentStr), 0644)
	})
}
