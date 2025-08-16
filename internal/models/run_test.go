package models

import (
	"testing"
)

func TestRunType(t *testing.T) {
	tests := []struct {
		name     string
		runType  RunType
		expected string
	}{
		{"Run type", RunTypeRun, "run"},
		{"Plan type", RunTypePlan, "plan"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.runType) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.runType))
			}
		})
	}
}

func TestRunStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   RunStatus
		expected string
	}{
		{"Queued status", StatusQueued, "QUEUED"},
		{"Initializing status", StatusInitializing, "INITIALIZING"},
		{"Processing status", StatusProcessing, "PROCESSING"},
		{"PostProcess status", StatusPostProcess, "POST_PROCESS"},
		{"Done status", StatusDone, "DONE"},
		{"Failed status", StatusFailed, "FAILED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, string(tt.status))
			}
		})
	}
}
