// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

// Environment variable constants
const (
	// EnvAPIKey is the environment variable for the RepoBird API key
	EnvAPIKey = "REPOBIRD_API_KEY"

	// EnvAPIURL is the environment variable for the RepoBird API URL
	EnvAPIURL = "REPOBIRD_API_URL"

	// EnvDebug is the environment variable for debug mode
	EnvDebug = "REPOBIRD_DEBUG"

	// EnvColor controls CLI color output: auto, always, or never
	EnvColor = "REPOBIRD_COLOR"

	// EnvEnvironment is the environment variable for setting the environment (prod/dev)
	EnvEnvironment = "REPOBIRD_ENV"

	// EnvEnableBulkRuns enables legacy bulk run workflows in development only
	EnvEnableBulkRuns = "REPOBIRD_DEV_ENABLE_BULK_RUNS"
)
