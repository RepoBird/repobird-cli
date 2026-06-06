// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import "testing"

func TestIsBulkRunsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		flag     string
		expected bool
	}{
		{name: "disabled by default", expected: false},
		{name: "flag alone is not enough", flag: "1", expected: false},
		{name: "development with true flag", env: "development", flag: "true", expected: true},
		{name: "dev with one flag", env: "dev", flag: "1", expected: true},
		{name: "production blocks flag", env: "production", flag: "1", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvEnvironment, tt.env)
			t.Setenv(EnvEnableBulkRuns, tt.flag)

			if got := IsBulkRunsEnabled(); got != tt.expected {
				t.Fatalf("IsBulkRunsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsPlanRunsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{name: "disabled by default", expected: false},
		{name: "enabled in development", env: "development", expected: true},
		{name: "enabled in dev", env: "dev", expected: true},
		{name: "disabled in production", env: "production", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvEnvironment, tt.env)

			if got := IsPlanRunsEnabled(); got != tt.expected {
				t.Fatalf("IsPlanRunsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}
