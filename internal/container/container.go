package container

import (
	"net/http"
	"time"

	"github.com/repobird/repobird-cli/internal/cache"
	"github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/internal/repository"
	"github.com/repobird/repobird-cli/internal/services"
)

// Container holds all application dependencies
type Container struct {
	config       *config.Config
	httpClient   *http.Client
	runRepo      domain.RunRepository
	runService   domain.RunService
	cacheService domain.CacheService
	gitService   domain.GitService
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *config.Config) *Container {
	// Create HTTP client
	httpClient := &http.Client{
		Timeout: 45 * time.Minute,
	}

	// Create services
	cacheService := cache.NewMemoryCache(30 * time.Second)
	gitService := services.NewGitService()

	// Create repository
	runRepo := repository.NewAPIRunRepository(
		httpClient,
		cfg.APIURL,
		cfg.APIKey,
		cfg.Debug,
	)

	// Create domain service
	runService := services.NewRunService(
		runRepo,
		cacheService,
		gitService,
	)

	return &Container{
		config:       cfg,
		httpClient:   httpClient,
		runRepo:      runRepo,
		runService:   runService,
		cacheService: cacheService,
		gitService:   gitService,
	}
}

// Config returns the application configuration
func (c *Container) Config() *config.Config {
	return c.config
}

// RunService returns the run service
func (c *Container) RunService() domain.RunService {
	return c.runService
}

// CacheService returns the cache service
func (c *Container) CacheService() domain.CacheService {
	return c.cacheService
}

// GitService returns the git service
func (c *Container) GitService() domain.GitService {
	return c.gitService
}

// RunRepository returns the run repository
func (c *Container) RunRepository() domain.RunRepository {
	return c.runRepo
}
