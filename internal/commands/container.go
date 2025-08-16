package commands

import (
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
		config := &config.Config{
			APIKey: cfg.APIKey,
			APIURL: apiURL,
			Debug:  cfg.Debug,
		}
		appContainer = container.NewContainer(config)
	}
	return appContainer
}

// resetContainer resets the container (useful for testing)
func resetContainer() {
	appContainer = nil
}
