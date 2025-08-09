package commands

import "github.com/spf13/cobra"

// Test helper functions for commands package

// NewRootCommand creates a new root command for testing
func NewRootCommand() *cobra.Command {
	return rootCmd
}

// NewRunCommand creates a new run command for testing
func NewRunCommand() *cobra.Command {
	return runCmd
}

// NewStatusCommand creates a new status command for testing
func NewStatusCommand() *cobra.Command {
	return statusCmd
}

// NewConfigCommand creates a new config command for testing
func NewConfigCommand() *cobra.Command {
	return configCmd
}

// NewAuthCommand creates a new auth command for testing
func NewAuthCommand() *cobra.Command {
	return authCmd
}

// NewTUICommand creates a new TUI command for testing
func NewTUICommand() *cobra.Command {
	// Return a dummy command since TUI might not be defined yet
	return &cobra.Command{
		Use:   "tui",
		Short: "Terminal User Interface",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
}
