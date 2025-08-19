//go:build !windows
// +build !windows

package commands

import "syscall"

// getStdinFD returns the file descriptor for stdin on Unix systems
func getStdinFD() int {
	return int(syscall.Stdin)
}