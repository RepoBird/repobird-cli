// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.design/x/clipboard"
)

// ClipboardError represents a clipboard operation error
type ClipboardError struct {
	Message string
}

func (e ClipboardError) Error() string {
	return e.Message
}

// NewClipboardError creates a new clipboard error
func NewClipboardError(message string) error {
	return ClipboardError{Message: message}
}

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

// WriteToClipboard writes text to the system clipboard with context support
// It uses CGO clipboard if available, otherwise falls back to OS commands
func WriteToClipboard(ctx context.Context, text string) error {
	if !clipboardInitialized {
		_ = InitClipboard()
	}

	if cgoAvailable {
		// Use CGO clipboard with context awareness
		done := clipboard.Write(clipboard.FmtText, []byte(text))
		select {
		case <-done:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return fmt.Errorf("clipboard write timeout")
		}
	}

	// Fallback to OS-specific commands
	return writeToClipboardFallback(ctx, text)
}

// WriteToClipboardWithTimeout is a convenience function that creates a context with default timeout
func WriteToClipboardWithTimeout(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return WriteToClipboard(ctx, text)
}

// writeToClipboardFallback uses OS-specific commands to write to clipboard with context support
func writeToClipboardFallback(ctx context.Context, text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "pbcopy")
	case "linux":
		// Check if we're on Wayland or X11
		if os.Getenv("WAYLAND_DISPLAY") != "" {
			// Wayland - use wl-copy
			cmd = exec.CommandContext(ctx, "wl-copy")
		} else if os.Getenv("DISPLAY") != "" {
			// X11 - use xclip
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		} else {
			// Try xclip as fallback
			cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
		}
	case "windows":
		// Use PowerShell on Windows
		cmd = exec.CommandContext(ctx, "powershell", "-command", "Set-Clipboard", "-Value", text)
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
