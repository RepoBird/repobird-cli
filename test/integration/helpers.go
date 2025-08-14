//go:build integration
// +build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	binaryPath string
	buildOnce  sync.Once
	buildErr   error
)

// BuildBinary builds the CLI binary once for all tests
func BuildBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		// Build to a fixed location that persists across test runs
		tmpDir := os.TempDir()
		binaryPath = filepath.Join(tmpDir, "repobird-test")
		if os.Getenv("GOOS") == "windows" {
			binaryPath += ".exe"
		}

		cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/repobird")
		output, err := cmd.CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("failed to build binary: %v\nOutput: %s", err, output)
		}
	})

	if buildErr != nil {
		t.Fatalf("Failed to build binary: %v", buildErr)
	}

	return binaryPath
}

// CommandResult represents the result of running a command
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// RunCommand executes the CLI with given arguments
func RunCommand(t *testing.T, args ...string) *CommandResult {
	t.Helper()
	return RunCommandWithEnv(t, nil, args...)
}

// RunCommandWithEnv executes the CLI with environment variables and arguments
func RunCommandWithEnv(t *testing.T, env map[string]string, args ...string) *CommandResult {
	t.Helper()

	binary := BuildBinary(t)
	cmd := exec.Command(binary, args...)

	// Set up environment
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run with timeout
	start := time.Now()
	err := runWithTimeout(cmd, 30*time.Second)
	duration := time.Since(start)

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Timeout or other error
			exitCode = -1
		}
	}

	return &CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: duration,
		Error:    err,
	}
}

// RunCommandWithInput executes the repobird CLI with stdin input
func RunCommandWithInput(t *testing.T, input string, args ...string) *CommandResult {
	t.Helper()

	binary := BuildBinary(t)
	cmd := exec.Command(binary, args...)

	// Set up stdin
	cmd.Stdin = strings.NewReader(input)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Run command
	start := time.Now()
	err := runWithTimeout(cmd, 30*time.Second)
	duration := time.Since(start)

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &CommandResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Duration: duration,
		Error:    err,
	}
}

// runWithTimeout runs a command with a timeout
func runWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		cmd.Process.Kill()
		return fmt.Errorf("command timed out after %v", timeout)
	case err := <-done:
		return err
	}
}

// SetupTestConfig creates a temporary config directory
func SetupTestConfig(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".repobird")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Also create cache directory
	cacheDir := filepath.Join(tmpDir, ".config", "repobird", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	return tmpDir
}

// SetupTestEnv creates a complete test environment with config and mock server
func SetupTestEnv(t *testing.T) (map[string]string, *MockServer) {
	t.Helper()

	// Create temporary home directory
	homeDir := SetupTestConfig(t)

	// Start mock server
	mockServer := NewMockServer(t)

	// Create environment
	env := map[string]string{
		"HOME":               homeDir,
		"REPOBIRD_API_URL":   mockServer.URL,
		"REPOBIRD_API_KEY":   "TEST_KEY",
		"REPOBIRD_TEST_MODE": "true", // Safety flag
		"NO_COLOR":           "true", // Disable color output for easier testing
	}

	// Set XDG_CONFIG_HOME to isolate cache
	env["XDG_CONFIG_HOME"] = filepath.Join(homeDir, ".config")

	return env, mockServer
}

// CompareGolden compares actual output with a golden file
func CompareGolden(t *testing.T, actual, goldenPath string, update bool) {
	t.Helper()

	// Ensure golden directory exists
	goldenDir := filepath.Dir(goldenPath)
	if err := os.MkdirAll(goldenDir, 0755); err != nil {
		t.Fatalf("Failed to create golden directory: %v", err)
	}

	if update {
		// Update golden file
		if err := os.WriteFile(goldenPath, []byte(actual), 0644); err != nil {
			t.Fatalf("Failed to update golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	// Read expected output
	expected, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("Golden file does not exist: %s\nRun with -update flag to create it", goldenPath)
		}
		t.Fatalf("Failed to read golden file: %v", err)
	}

	// Compare
	if string(expected) != actual {
		t.Errorf("Output does not match golden file %s\nExpected:\n%s\nActual:\n%s",
			goldenPath, expected, actual)
	}
}

// AssertContains checks if the output contains expected string
func AssertContains(t *testing.T, output, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output to contain %q\nActual output:\n%s", expected, truncate(output, 500))
	}
}

// AssertNotContains checks if the output does not contain string
func AssertNotContains(t *testing.T, output, unexpected string) {
	t.Helper()
	if strings.Contains(output, unexpected) {
		t.Errorf("Expected output to NOT contain %q\nActual output:\n%s", unexpected, truncate(output, 500))
	}
}

// AssertExitCode checks if the command exited with expected code
func AssertExitCode(t *testing.T, result *CommandResult, expected int) {
	t.Helper()
	if result.ExitCode != expected {
		t.Errorf("Expected exit code %d, got %d\nStdout:\n%s\nStderr:\n%s",
			expected, result.ExitCode, result.Stdout, result.Stderr)
	}
}

// AssertSuccess checks if the command succeeded (exit code 0)
func AssertSuccess(t *testing.T, result *CommandResult) {
	t.Helper()
	AssertExitCode(t, result, 0)
}

// AssertFailure checks if the command failed (exit code != 0)
func AssertFailure(t *testing.T, result *CommandResult) {
	t.Helper()
	if result.ExitCode == 0 {
		t.Errorf("Expected command to fail, but it succeeded\nStdout:\n%s", result.Stdout)
	}
}

// AssertEquals checks that two strings are equal
func AssertEquals(t *testing.T, actual, expected string) {
	t.Helper()
	if actual != expected {
		t.Errorf("Values are not equal.\nExpected: %q\nActual: %q", expected, actual)
	}
}

// AssertJSONEquals checks that two JSON strings are equivalent
func AssertJSONEquals(t *testing.T, actual, expected string) {
	t.Helper()

	var actualJSON, expectedJSON interface{}

	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Errorf("Failed to parse actual JSON: %v\nJSON: %s", err, actual)
		return
	}

	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Errorf("Failed to parse expected JSON: %v\nJSON: %s", err, expected)
		return
	}

	actualBytes, _ := json.Marshal(actualJSON)
	expectedBytes, _ := json.Marshal(expectedJSON)

	if string(actualBytes) != string(expectedBytes) {
		t.Errorf("JSON values are not equal.\nExpected: %s\nActual: %s",
			string(expectedBytes), string(actualBytes))
	}
}

// CreateTestFile creates a temporary file with content
func CreateTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	return path
}

// CreateTestDirectory creates a test directory structure
func CreateTestDirectory(t *testing.T, base string, structure map[string]string) {
	t.Helper()

	for path, content := range structure {
		fullPath := filepath.Join(base, path)
		dir := filepath.Dir(fullPath)

		// Create parent directories
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Create file with content
		if content != "" {
			if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
				t.Fatalf("Failed to create file %s: %v", fullPath, err)
			}
		}
	}
}

// CopyFile copies a file from src to dst
func CopyFile(t *testing.T, src, dst string) {
	t.Helper()

	srcFile, err := os.Open(src)
	if err != nil {
		t.Fatalf("Failed to open source file: %v", err)
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}
}

// WaitForCondition waits for a condition to be true with timeout
func WaitForCondition(timeout time.Duration, check func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(attempts int, initial time.Duration, f func() error) error {
	var err error
	delay := initial

	for i := 0; i < attempts; i++ {
		if err = f(); err == nil {
			return nil
		}

		if i < attempts-1 {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", attempts, err)
}

// GetUpdateFlag returns whether to update golden files
func GetUpdateFlag() bool {
	// Check environment variable
	if os.Getenv("UPDATE_GOLDEN") == "1" || os.Getenv("UPDATE_GOLDEN") == "true" {
		return true
	}
	// Also check command line args for backward compatibility
	for _, arg := range os.Args {
		if arg == "-update" {
			return true
		}
	}
	return false
}

// truncate truncates a string to a maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... (truncated)"
}

// CaptureOutput captures stdout and stderr during function execution
func CaptureOutput(f func()) (string, string) {
	// Save original stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	// Create pipes
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	// Redirect stdout and stderr
	os.Stdout = wOut
	os.Stderr = wErr

	// Run function
	f()

	// Restore original stdout and stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Close write ends
	wOut.Close()
	wErr.Close()

	// Read captured output
	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)

	return string(outBytes), string(errBytes)
}

// CleanupTestData removes test artifacts
func CleanupTestData(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			t.Logf("Warning: failed to cleanup %s: %v", path, err)
		}
	}
}

// SkipIfShort skips the test if -short flag is set
func SkipIfShort(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
}

// SkipIfNoNetwork skips the test if network is not available
func SkipIfNoNetwork(t *testing.T) {
	t.Helper()
	// Simple check - try to resolve a well-known domain
	cmd := exec.Command("ping", "-c", "1", "-W", "1", "8.8.8.8")
	if err := cmd.Run(); err != nil {
		t.Skip("Skipping test: network not available")
	}
}

// RequireBinary ensures a binary is available or skips the test
func RequireBinary(t *testing.T, binary string) {
	t.Helper()
	if _, err := exec.LookPath(binary); err != nil {
		t.Skipf("Skipping test: required binary '%s' not found", binary)
	}
}
