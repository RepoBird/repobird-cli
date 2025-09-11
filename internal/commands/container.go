// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"os"

	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/container"
	"github.com/repobird/repobird-cli/internal/utils"
)

var appContainer *container.Container

// getContainer returns the application container, creating it if necessary
func getContainer() *container.Container {
	if appContainer == nil {
		// Convert SecureConfig to Config
		apiURL := utils.GetAPIURL(cfg.APIURL)
		// Enable debug if either flag or environment variable is set
		isDebug := cfg.Debug || os.Getenv("REPOBIRD_DEBUG_LOG") == "1"
		config := &config.Config{
			APIKey: cfg.APIKey,
			APIURL: apiURL,
			Debug:  isDebug,
		}
		appContainer = container.NewContainer(config)
	}
	return appContainer
}

// resetContainer resets the container (useful for testing)
func resetContainer() {
	appContainer = nil
}
