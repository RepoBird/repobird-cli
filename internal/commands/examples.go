// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
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
	Use:   "generate [run|bulk|minimal]",
	Short: "Generate example configuration files",
	Long: `Generate example configuration files for single or bulk runs.

Examples:
  repobird examples generate                     # Generate full single run example (JSON)
  repobird examples generate minimal             # Generate minimal config (only required fields)
  repobird examples generate run --format yaml   # Generate YAML example
  repobird examples generate run --format md     # Generate Markdown example
  repobird examples generate bulk                # Generate bulk run example
  repobird examples generate minimal -o task.json # Save minimal config to file`,
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
	fmt.Println()
	fmt.Println("OPTIONAL FIELDS:")
	fmt.Println("  • target      (string)  - Target branch name for changes (default: auto-generated)")
	fmt.Println("  • title       (string)  - Human-readable title for the run (default: auto-generated)")
	fmt.Println("  • source      (string)  - Source branch (default: 'main', auto-detected in git repos)")
	fmt.Println("  • runType     (string)  - Type: 'run' or 'plan' (default: 'run')")
	fmt.Println("  • context     (string)  - Additional context or instructions")
	fmt.Println("  • files       (array)   - List of specific files to include")
	fmt.Println()
	fmt.Println("RUN TYPES:")
	fmt.Println("  • run      - AI makes changes and creates PR automatically")
	fmt.Println("  • plan     - AI creates detailed plan without code changes")
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
	// If no arguments provided and no output file, show help
	if len(args) == 0 && outputFile == "" {
		return cmd.Help()
	}
	
	exampleType := "run"
	if len(args) > 0 {
		exampleType = args[0]
	}

	var content string
	var err error

	switch exampleType {
	case "run":
		content, err = generateRunExample(formatType, false)
	case "minimal":
		content, err = generateRunExample(formatType, true)
	case "bulk":
		content, err = generateBulkExample()
	default:
		return fmt.Errorf("unknown example type: %s (use 'run', 'minimal', or 'bulk')", exampleType)
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
		fmt.Printf("Run with: repobird run %s\n", outputFile)
	} else {
		// When no output file specified, just show what would be generated
		fmt.Printf("Example %s configuration (%s format):\n\n", exampleType, formatType)
		fmt.Println(content)
	}

	return nil
}

// jsonQuote properly escapes and quotes a string for JSON
func jsonQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func generateRunExample(format string, minimal bool) (string, error) {
	var example map[string]interface{}
	
	if minimal {
		// Minimal example with only required fields
		example = map[string]interface{}{
			"repository": "myorg/webapp",
			"prompt":     "Fix the authentication bug where users cannot log in after 5 failed attempts",
		}
	} else {
		// Full example with all commonly used fields (repository first, prompt second)
		example = map[string]interface{}{
			"repository": "myorg/webapp",
			"prompt":     "Fix the authentication bug where users cannot log in after 5 failed attempts",
			"source":     "main",
			"target":     "fix/auth-bug",
			"title":      "Fix authentication rate limiting",
			"runType":    "run",
			"context":    "Users report being locked out permanently. The rate limit should reset after 15 minutes.",
		}
	}

	switch strings.ToLower(format) {
	case "json":
		// Manually build JSON to maintain field order (repository first, prompt second)
		var jsonStr string
		if minimal {
			jsonStr = fmt.Sprintf(`{
  "repository": %s,
  "prompt": %s
}`, jsonQuote(example["repository"].(string)), jsonQuote(example["prompt"].(string)))
		} else {
			// Full example with all fields in desired order
			jsonStr = fmt.Sprintf(`{
  "repository": %s,
  "prompt": %s,
  "source": %s,
  "target": %s,
  "title": %s,
  "runType": %s,
  "context": %s
}`, jsonQuote(example["repository"].(string)),
				jsonQuote(example["prompt"].(string)),
				jsonQuote(example["source"].(string)),
				jsonQuote(example["target"].(string)),
				jsonQuote(example["title"].(string)),
				jsonQuote(example["runType"].(string)),
				jsonQuote(example["context"].(string)))
		}
		return jsonStr, nil

	case "yaml", "yml":
		// Manually build YAML to maintain field order (repository first, prompt second)
		var yamlStr string
		if minimal {
			// For minimal, use multiline YAML format for prompt
			yamlStr = fmt.Sprintf(`repository: %s
prompt: |
  Fix the authentication rate limiting bug.
  Users are permanently locked out after 5 failed login attempts.
  Should reset after 15 minutes but doesn't.`, 
				example["repository"])
		} else {
			// For full example, use concise multiline YAML for prompt and context
			yamlStr = fmt.Sprintf(`repository: %s
prompt: |
  Fix authentication rate limiting that permanently locks users after 5 failed attempts.
  
  Expected: Temporary 15-minute lockout with automatic reset
  Current: Permanent lockout requiring manual intervention
  Impact: Multiple daily support tickets from affected users
source: %s
target: %s
title: %s
runType: %s
context: |
  Bug introduced in v2.3.0 security update. Check auth middleware rate limiting logic.
  Consider timestamp tracking and timezone handling.
  Add tests for lockout/reset behavior.`, 
				example["repository"],
				example["source"],
				example["target"],
				example["title"],
				example["runType"])
		}
		return yamlStr, nil

	case "md", "markdown":
		// Create frontmatter with proper field ordering
		var frontmatterYAML string
		if minimal {
			frontmatterYAML = fmt.Sprintf(`repository: %s
prompt: %s`, example["repository"], example["prompt"])
		} else {
			frontmatterYAML = fmt.Sprintf(`repository: %s
prompt: %s
source: %s
target: %s
title: %s
runType: %s`, 
				example["repository"],
				example["prompt"],
				example["source"],
				example["target"],
				example["title"],
				example["runType"])
		}

		markdown := "---\n" + frontmatterYAML + "\n---\n\n"
		
		if !minimal {
			// Add detailed documentation for full example
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
		}

		return markdown, nil

	default:
		return "", fmt.Errorf("unsupported format: %s (use json, yaml, or md)", format)
	}
}

func generateBulkExample() (string, error) {
	// Manually build JSON to maintain field order
	bulkJSON := `{
  "runs": [
    {
      "repository": "myorg/webapp",
      "prompt": "Add comprehensive error handling to the authentication module",
      "context": "Add try-catch blocks and user-friendly error messages"
    },
    {
      "repository": "myorg/webapp",
      "prompt": "Create unit tests for the user profile component with at least 80% code coverage"
    },
    {
      "repository": "myorg/backend",
      "prompt": "Plan refactoring of the database layer to use connection pooling",
      "runType": "plan",
      "context": "Current implementation creates new connections for each request"
    }
  ]
}`
	
	header := "// Bulk run configuration example\n"
	header += "// This will create 3 separate runs sequentially\n"
	header += "// Save this file and run: repobird bulk bulk-tasks.json\n\n"
	
	return header + bulkJSON, nil
}