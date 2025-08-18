package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// getDebugLogPath returns the debug log path, configurable via environment variable
func getDebugLogPath() string {
	// Check if debug logging is enabled (REPOBIRD_DEBUG_LOG=1 or any value)
	debugEnv := os.Getenv("REPOBIRD_DEBUG_LOG")
	
	// If it's a path (contains / or \), use it as the log path
	if debugEnv != "" && (filepath.IsAbs(debugEnv) || filepath.Dir(debugEnv) != ".") {
		return debugEnv
	}

	// Find project root by looking for go.mod file
	projectRoot := findProjectRoot()
	if projectRoot == "" {
		// Fallback to temp directory if project root not found
		return filepath.Join(os.TempDir(), "repobird_debug.log")
	}

	// Use logs directory in the project root
	logsDir := filepath.Join(projectRoot, "logs")
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		// Fallback to temp directory if can't create logs dir
		return filepath.Join(os.TempDir(), "repobird_debug.log")
	}
	return filepath.Join(logsDir, "repobird_debug.log")
}

// findProjectRoot searches for the project root by looking for go.mod
func findProjectRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return ""
}

// LogToFile writes a debug message to the debug log file
func LogToFile(message string) {
	// Only log if debug logging is enabled (any non-empty value)
	debugEnv := os.Getenv("REPOBIRD_DEBUG_LOG")
	if debugEnv == "" || debugEnv == "0" || debugEnv == "false" {
		return
	}
	
	if f, err := os.OpenFile(getDebugLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600); err == nil {
		defer func() { _ = f.Close() }()
		_, _ = f.WriteString(message)
	}
}

// LogToFilef writes a formatted debug message to the debug log file
func LogToFilef(format string, args ...interface{}) {
	LogToFile(fmt.Sprintf(format, args...))
}

// LogToFileWithTimestamp writes a debug message with timestamp prefix
func LogToFileWithTimestamp(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	LogToFile(fmt.Sprintf("[%s] %s", timestamp, message))
}

// LogToFileWithTimestampf writes a formatted debug message with timestamp prefix
func LogToFileWithTimestampf(format string, args ...interface{}) {
	LogToFileWithTimestamp(fmt.Sprintf(format, args...))
}
