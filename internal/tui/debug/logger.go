package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// getDebugLogPath returns the debug log path, configurable via environment variable
func getDebugLogPath() string {
	if path := os.Getenv("REPOBIRD_DEBUG_LOG"); path != "" {
		return path
	}
	// Get the working directory
	wd, err := os.Getwd()
	if err != nil {
		return filepath.Join(os.TempDir(), "repobird_debug.log")
	}
	// Use logs directory in the project
	logsDir := filepath.Join(wd, "logs")
	// Create logs directory if it doesn't exist
	_ = os.MkdirAll(logsDir, 0755)
	return filepath.Join(logsDir, "repobird_debug.log")
}

// LogToFile writes a debug message to the debug log file
func LogToFile(message string) {
	if f, err := os.OpenFile(getDebugLogPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
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
