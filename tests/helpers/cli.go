package helpers

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/repobird/repobird-cli/internal/config"
	"github.com/stretchr/testify/require"
)

// CLIResult represents the result of running a CLI command
type CLIResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// RunCLI executes the CLI binary with the given arguments and returns the result
func RunCLI(t *testing.T, args ...string) *CLIResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	binaryPath := filepath.Join("..", "..", "build", "repobird")

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return &CLIResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

// RunCLIWithInput runs CLI command with stdin input
func RunCLIWithInput(t *testing.T, input string, args ...string) *CLIResult {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	binaryPath := filepath.Join("..", "..", "build", "repobird")

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Stdin = bytes.NewBufferString(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	return &CLIResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Err:      err,
	}
}

// SetupTestEnvironment prepares a clean test environment
func SetupTestEnvironment(t *testing.T) (cleanup func()) {
	t.Helper()

	// Create temporary directory for config
	tempDir, err := os.MkdirTemp("", "repobird-test-*")
	require.NoError(t, err)

	// Set environment variables
	origHome := os.Getenv("HOME")
	origAPIKey := os.Getenv(config.EnvAPIKey)
	origAPIURL := os.Getenv(config.EnvAPIURL)

	_ = os.Setenv("HOME", tempDir)
	_ = os.Unsetenv(config.EnvAPIKey)
	_ = os.Unsetenv(config.EnvAPIURL)

	return func() {
		_ = os.RemoveAll(tempDir)
		if origHome != "" {
			_ = os.Setenv("HOME", origHome)
		}
		if origAPIKey != "" {
			_ = os.Setenv(config.EnvAPIKey, origAPIKey)
		}
		if origAPIURL != "" {
			_ = os.Setenv(config.EnvAPIURL, origAPIURL)
		}
	}
}

// EnsureBinaryExists ensures the CLI binary is built before running tests
func EnsureBinaryExists(t *testing.T) {
	t.Helper()

	binaryPath := filepath.Join("..", "..", "build", "repobird")
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("CLI binary not found. Run 'make build' first.")
	}
}
