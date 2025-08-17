// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

// ActiveStatuses are the status values that indicate a run is still active
var ActiveStatuses = []string{"QUEUED", "INITIALIZING", "PROCESSING", "POST_PROCESS"}

// TerminalStatuses are the status values that indicate a run has completed
var TerminalStatuses = []string{"COMPLETED", "FAILED", "CANCELLED", "ERROR"}

// IsActiveStatus checks if a status string indicates the run is still active
func IsActiveStatus(status string) bool {
	for _, s := range ActiveStatuses {
		if status == s {
			return true
		}
	}
	return false
}

// IsTerminalStatus checks if a status string indicates the run has completed
func IsTerminalStatus(status string) bool {
	for _, s := range TerminalStatuses {
		if status == s {
			return true
		}
	}
	return false
}

// IsSuccessStatus checks if a status string indicates successful completion
func IsSuccessStatus(status string) bool {
	return status == "COMPLETED"
}

// IsFailureStatus checks if a status string indicates failure
func IsFailureStatus(status string) bool {
	return status == "FAILED" || status == "ERROR" || status == "CANCELLED"
}
