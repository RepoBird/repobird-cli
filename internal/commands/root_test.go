package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRootCommand_NoDuplicateCommands(t *testing.T) {
	// Create a new root command to test
	testRootCmd := &cobra.Command{
		Use:   "repobird",
		Short: "RepoBird CLI - AI-powered code generation and repository management",
	}

	// Manually add commands as done in init() functions
	// This simulates the actual command registration
	testRootCmd.AddCommand(versionCmd)
	testRootCmd.AddCommand(runCmd)
	testRootCmd.AddCommand(statusCmd)
	testRootCmd.AddCommand(configCmd)
	testRootCmd.AddCommand(authCmd)
	// Note: completion and docs commands are added by their own init() functions

	// Track command names that have been seen
	commandNames := make(map[string]bool)
	duplicates := []string{}

	// Check all commands registered to testRootCmd
	for _, cmd := range testRootCmd.Commands() {
		name := cmd.Name()
		if commandNames[name] {
			// Found a duplicate
			duplicates = append(duplicates, name)
		}
		commandNames[name] = true
	}

	// Assert no duplicates found
	assert.Empty(t, duplicates, "Duplicate commands found in root command: %v", duplicates)
}

func TestRootCommand_HasExpectedCommands(t *testing.T) {
	// List of expected commands
	expectedCommands := []string{
		"version",
		"run",
		"status",
		"config",
		"auth",
		"bulk",
		"tui",
		"completion",
		"docs",
	}

	// Get actual command names
	actualCommands := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		actualCommands[cmd.Name()] = true
	}

	// Check each expected command is present
	for _, expected := range expectedCommands {
		assert.True(t, actualCommands[expected], "Expected command '%s' not found in root command", expected)
	}
}

func TestRootCommand_CommandDescriptions(t *testing.T) {
	// Map of command names to their expected short descriptions
	expectedDescriptions := map[string]string{
		"version":    "Print version information",
		"run":        "Create a new run from a JSON, YAML, or Markdown file",
		"status":     "Check the status of runs",
		"config":     "Manage RepoBird configuration",
		"auth":       "Manage authentication and API keys",
		"bulk":       "Submit multiple runs in parallel from configuration files",
		"tui":        "Launch the interactive Terminal User Interface",
		"completion": "Generate shell completion scripts",
		"docs":       "Generate documentation",
	}

	// Check each command has the correct description
	for _, cmd := range rootCmd.Commands() {
		expectedDesc, exists := expectedDescriptions[cmd.Name()]
		if exists {
			assert.Equal(t, expectedDesc, cmd.Short, "Command '%s' has incorrect description", cmd.Name())
		}
	}
}
