package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	outputFile   string
	formatType   string
	exampleType  string
	interactive  bool
)

var examplesCmd = &cobra.Command{
	Use:   "examples",
	Short: "Show configuration schemas and generate example files",
	Long: `Show configuration schemas and generate example files for RepoBird runs.

This command helps you understand the configuration format for single runs and bulk runs,
showing all required and optional fields with their types and descriptions.`,
	RunE: showExamples,
}

var schemaCmd = &cobra.Command{
	Use:   "schema [run|bulk]",
	Short: "Display configuration schema for run or bulk configurations",
	Long: `Display the complete configuration schema showing all fields, types, and descriptions.

Examples:
  repobird examples schema        # Show single run schema (default)
  repobird examples schema run    # Show single run schema  
  repobird examples schema bulk   # Show bulk run schema`,
	Args: cobra.MaximumNArgs(1),
	RunE: showSchema,
}

var generateCmd = &cobra.Command{
	Use:   "generate [run|bulk]",
	Short: "Generate example configuration files",
	Long: `Generate example configuration files for single or bulk runs.

Examples:
  repobird examples generate                     # Generate single run example (JSON)
  repobird examples generate run --format yaml   # Generate YAML example
  repobird examples generate run --format md     # Generate Markdown example
  repobird examples generate bulk                # Generate bulk run example
  repobird examples generate run -o task.json    # Save to file`,
	Args: cobra.MaximumNArgs(1),
	RunE: generateExample,
}

func init() {
	generateCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file path")
	generateCmd.Flags().StringVarP(&formatType, "format", "f", "json", "output format: json, yaml, or md (markdown)")
	
	examplesCmd.AddCommand(schemaCmd)
	examplesCmd.AddCommand(generateCmd)
}

func showExamples(cmd *cobra.Command, args []string) error {
	fmt.Println("RepoBird Configuration Examples")
	fmt.Println("================================")
	fmt.Println()
	fmt.Println("QUICK START:")
	fmt.Println("  repobird examples schema        # Show configuration fields")
	fmt.Println("  repobird examples generate      # Generate example file")
	fmt.Println()
	fmt.Println("SINGLE RUN FORMATS:")
	fmt.Println("  • JSON (.json)         - Standard JSON configuration")
	fmt.Println("  • YAML (.yaml, .yml)   - Human-friendly YAML format")
	fmt.Println("  • Markdown (.md)       - Documentation with YAML frontmatter")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  schema [type]     Show configuration schema")
	fmt.Println("  generate [type]   Generate example configuration files")
	fmt.Println()
	fmt.Println("For detailed documentation, see: docs/run-config-formats.md")
	return nil
}

func showSchema(cmd *cobra.Command, args []string) error {
	schemaType := "run"
	if len(args) > 0 {
		schemaType = args[0]
	}

	switch schemaType {
	case "run":
		showRunSchema()
	case "bulk":
		showBulkSchema()
	default:
		return fmt.Errorf("unknown schema type: %s (use 'run' or 'bulk')", schemaType)
	}
	return nil
}

func showRunSchema() {
	fmt.Println("SINGLE RUN CONFIGURATION SCHEMA")
	fmt.Println("================================")
	fmt.Println()
	fmt.Println("REQUIRED FIELDS:")
	fmt.Println("  • prompt      (string)  - Task description/instructions for the AI")
	fmt.Println("  • repository  (string)  - Repository in format 'owner/repo' (auto-detected in git repos)")
	fmt.Println("  • target      (string)  - Target branch name for changes")
	fmt.Println("  • title       (string)  - Human-readable title for the run")
	fmt.Println()
	fmt.Println("OPTIONAL FIELDS:")
	fmt.Println("  • source      (string)  - Source branch (default: 'main', auto-detected in git repos)")
	fmt.Println("  • runType     (string)  - Type: 'run', 'plan', or 'approval' (default: 'run')")
	fmt.Println("  • context     (string)  - Additional context or instructions")
	fmt.Println("  • files       (array)   - List of specific files to include")
	fmt.Println()
	fmt.Println("RUN TYPES:")
	fmt.Println("  • run      - AI makes changes and creates PR automatically")
	fmt.Println("  • plan     - AI creates detailed plan without code changes")
	fmt.Println("  • approval - AI makes changes but waits for approval")
	fmt.Println()
	fmt.Println("AUTO-DETECTION:")
	fmt.Println("  When running from a git repository:")
	fmt.Println("  • repository - Detected from git remote origin")
	fmt.Println("  • source     - Detected from current branch")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  repobird examples generate          # Generate JSON example")
	fmt.Println("  repobird examples generate -f yaml  # Generate YAML example")
	fmt.Println("  repobird examples generate -f md    # Generate Markdown example")
}

func showBulkSchema() {
	fmt.Println("BULK RUN CONFIGURATION SCHEMA")
	fmt.Println("==============================")
	fmt.Println()
	fmt.Println("TOP-LEVEL STRUCTURE:")
	fmt.Println("  {")
	fmt.Println("    \"runs\": [              // Array of run configurations")
	fmt.Println("      { /* run config */ },")
	fmt.Println("      { /* run config */ }")
	fmt.Println("    ]")
	fmt.Println("  }")
	fmt.Println()
	fmt.Println("EACH RUN CONFIGURATION:")
	fmt.Println("  Same as single run schema (see 'repobird examples schema run')")
	fmt.Println()
	fmt.Println("BULK-SPECIFIC BEHAVIOR:")
	fmt.Println("  • Processes runs sequentially by default")
	fmt.Println("  • Continues on individual run failures")
	fmt.Println("  • Provides summary of all runs at completion")
	fmt.Println("  • Supports --dry-run for validation")
	fmt.Println()
	fmt.Println("EXAMPLE:")
	fmt.Println("  repobird examples generate bulk     # Generate bulk example")
	fmt.Println("  repobird bulk config.json --dry-run # Validate bulk config")
}

func generateExample(cmd *cobra.Command, args []string) error {
	exampleType := "run"
	if len(args) > 0 {
		exampleType = args[0]
	}

	var content string
	var err error

	switch exampleType {
	case "run":
		content, err = generateRunExample(formatType)
	case "bulk":
		content, err = generateBulkExample()
	default:
		return fmt.Errorf("unknown example type: %s (use 'run' or 'bulk')", exampleType)
	}

	if err != nil {
		return err
	}

	if outputFile != "" {
		// Ensure directory exists
		dir := filepath.Dir(outputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if err := os.WriteFile(outputFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
		fmt.Printf("✓ Example configuration written to: %s\n", outputFile)
		fmt.Printf("\nRun with: repobird run %s\n", outputFile)
	} else {
		fmt.Println(content)
		fmt.Println()
		fmt.Println("Tip: Save to file with -o flag: repobird examples generate -o task.json")
	}

	return nil
}

func generateRunExample(format string) (string, error) {
	example := map[string]interface{}{
		"prompt":     "Fix the authentication bug where users cannot log in after 5 failed attempts",
		"repository": "myorg/webapp",
		"source":     "main",
		"target":     "fix/auth-rate-limit",
		"title":      "Fix authentication rate limiting",
		"runType":    "run",
		"context":    "Users report being locked out permanently. The rate limit should reset after 15 minutes.",
		"files": []string{
			"src/auth/login.js",
			"src/auth/rateLimit.js",
		},
	}

	switch strings.ToLower(format) {
	case "json":
		b, err := json.MarshalIndent(example, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil

	case "yaml", "yml":
		b, err := yaml.Marshal(example)
		if err != nil {
			return "", err
		}
		return string(b), nil

	case "md", "markdown":
		// Create frontmatter without context (will be in markdown body)
		frontmatter := map[string]interface{}{
			"prompt":     example["prompt"],
			"repository": example["repository"],
			"source":     example["source"],
			"target":     example["target"],
			"title":      example["title"],
			"runType":    example["runType"],
			"files":      example["files"],
		}
		
		b, err := yaml.Marshal(frontmatter)
		if err != nil {
			return "", err
		}

		markdown := "---\n" + string(b) + "---\n\n"
		markdown += "# Task: Fix Authentication Rate Limiting\n\n"
		markdown += "## Problem Description\n\n"
		markdown += "Users are experiencing a critical issue with our authentication system. "
		markdown += "After 5 failed login attempts, they are permanently locked out instead of being temporarily rate-limited.\n\n"
		markdown += "## Expected Behavior\n\n"
		markdown += "- After 5 failed attempts, users should be temporarily locked for 15 minutes\n"
		markdown += "- The lockout should automatically reset after the timeout period\n"
		markdown += "- Users should see a clear message indicating when they can try again\n\n"
		markdown += "## Technical Details\n\n"
		markdown += "The issue appears to be in the `rateLimit.js` module where the reset logic is not properly implemented.\n\n"
		markdown += "## Testing Requirements\n\n"
		markdown += "- Test with multiple failed attempts\n"
		markdown += "- Verify the 15-minute reset works\n"
		markdown += "- Ensure proper error messages are shown\n"

		return markdown, nil

	default:
		return "", fmt.Errorf("unsupported format: %s (use json, yaml, or md)", format)
	}
}

func generateBulkExample() (string, error) {
	bulk := map[string]interface{}{
		"runs": []map[string]interface{}{
			{
				"prompt":     "Add comprehensive error handling to the authentication module",
				"repository": "myorg/webapp",
				"source":     "main",
				"target":     "feature/auth-error-handling",
				"title":      "Improve auth error handling",
				"runType":    "run",
				"context":    "Add try-catch blocks and user-friendly error messages",
				"files": []string{
					"src/auth/login.js",
					"src/auth/register.js",
				},
			},
			{
				"prompt":     "Create unit tests for the user profile component",
				"repository": "myorg/webapp",
				"source":     "main", 
				"target":     "test/user-profile",
				"title":      "Add user profile tests",
				"runType":    "run",
				"context":    "Achieve at least 80% code coverage",
				"files": []string{
					"src/components/UserProfile.js",
				},
			},
			{
				"prompt":     "Plan refactoring of the database layer to use connection pooling",
				"repository": "myorg/backend",
				"source":     "develop",
				"target":     "plan/db-pooling",
				"title":      "Database connection pooling plan",
				"runType":    "plan",
				"context":    "Current implementation creates new connections for each request",
			},
		},
	}

	b, err := json.MarshalIndent(bulk, "", "  ")
	if err != nil {
		return "", err
	}
	
	header := "// Bulk run configuration example\n"
	header += "// This will create 3 separate runs sequentially\n"
	header += "// Save this file and run: repobird bulk bulk-tasks.json\n\n"
	
	return header + string(b), nil
}