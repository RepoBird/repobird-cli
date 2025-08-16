package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate documentation",
	Long:  "Generate documentation for the RepoBird CLI and TUI - trigger AI coding agents, submit batch runs, and interactively manage your automated pull request generation.",
}

var manCmd = &cobra.Command{
	Use:   "man [output-dir]",
	Short: "Generate man pages",
	Long:  "Generate man pages for RepoBird CLI commands.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "man"
		if len(args) > 0 {
			outputDir = args[0]
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		header := &doc.GenManHeader{
			Title:   "REPOBIRD",
			Section: "1",
			Manual:  "RepoBird CLI Manual",
			Source:  "CLI and TUI for RepoBird.ai - trigger AI coding agents and manage runs",
		}

		if err := doc.GenManTree(rootCmd, header, outputDir); err != nil {
			return fmt.Errorf("failed to generate man pages: %w", err)
		}

		fmt.Printf("✓ Man pages generated in %s directory\n", outputDir)
		return nil
	},
}

var markdownCmd = &cobra.Command{
	Use:   "markdown [output-dir]",
	Short: "Generate markdown documentation",
	Long:  "Generate markdown documentation for RepoBird CLI commands.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "docs/generated"
		if len(args) > 0 {
			outputDir = args[0]
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := doc.GenMarkdownTree(rootCmd, outputDir); err != nil {
			return fmt.Errorf("failed to generate markdown docs: %w", err)
		}

		fmt.Printf("✓ Markdown documentation generated in %s directory\n", outputDir)
		return nil
	},
}

var yamlCmd = &cobra.Command{
	Use:   "yaml [output-dir]",
	Short: "Generate YAML documentation",
	Long:  "Generate YAML documentation for RepoBird CLI commands.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "yaml"
		if len(args) > 0 {
			outputDir = args[0]
		}

		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		if err := doc.GenYamlTree(rootCmd, outputDir); err != nil {
			return fmt.Errorf("failed to generate YAML docs: %w", err)
		}

		fmt.Printf("✓ YAML documentation generated in %s directory\n", outputDir)
		return nil
	},
}

func init() {
	docsCmd.AddCommand(manCmd)
	docsCmd.AddCommand(markdownCmd)
	docsCmd.AddCommand(yamlCmd)
	rootCmd.AddCommand(docsCmd)
}
