//go:build windows
// +build windows

package commands

import (
	"os"
	"syscall"
)

// getStdinFD returns the file descriptor for stdin on Windows systems
func getStdinFD() int {
	// On Windows, we need to get the file descriptor from os.Stdin
	// syscall.Stdin is a Handle, not an int
	return int(os.Stdin.Fd())
}