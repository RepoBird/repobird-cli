// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"os"
	"strings"
)

func IsDevelopmentEnvironment() bool {
	env := strings.ToLower(os.Getenv(EnvEnvironment))
	return env == "dev" || env == "development"
}

func IsPlanRunsEnabled() bool {
	return IsDevelopmentEnvironment()
}

func PlanRunsUnavailableMessage() string {
	return "plan runs are temporarily unavailable during the OpenCode migration; set REPOBIRD_ENV=development to test plan mode against a development server"
}

func IsBulkRunsEnabled() bool {
	if !IsDevelopmentEnvironment() {
		return false
	}

	switch strings.ToLower(os.Getenv(EnvEnableBulkRuns)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
