package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.design/x/clipboard"
)

var clipboardInitialized bool
var cgoAvailable bool

// InitClipboard tries to initialize the clipboard
// It will detect if CGO is available and set the appropriate mode
func InitClipboard() error {
	// Try to initialize the CGO clipboard
	// This will panic if CGO is not enabled, so we recover
	defer func() {
		if r := recover(); r != nil {
			// CGO is not available, we'll use fallback
			cgoAvailable = false
			clipboardInitialized = true
		}
	}()

	err := clipboard.Init()
	if err == nil {
		cgoAvailable = true
		clipboardInitialized = true
	}
	return nil
}

// WriteToClipboard writes text to the system clipboard
// It uses CGO clipboard if available, otherwise falls back to OS commands
func WriteToClipboard(text string) error {
	if !clipboardInitialized {
		InitClipboard()
	}

	if cgoAvailable {
		// Use CGO clipboard
		done := clipboard.Write(clipboard.FmtText, []byte(text))
		select {
		case <-done:
			return nil
		case <-time.After(2 * time.Second):
			return fmt.Errorf("clipboard write timeout")
		}
	}

	// Fallback to OS-specific commands
	return writeToClipboardFallback(text)
}

// writeToClipboardFallback uses OS-specific commands to write to clipboard
func writeToClipboardFallback(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Check if we're on Wayland or X11
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			// Wayland - use wl-copy
			cmd = exec.Command("wl-copy")
		} else if os.Getenv("DISPLAY") != "" {
			// X11 - use xclip
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			// Try xclip as fallback
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	case "windows":
		// Use PowerShell on Windows
		cmd = exec.Command("powershell", "-command", "Set-Clipboard", "-Value", text)
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if cmd != nil {
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}

	return fmt.Errorf("failed to find clipboard command")
}
