package tui

import (
	"context"

	"github.com/repobird/repobird-cli/internal/models"
)

// APIClient defines the interface for API operations needed by the TUI
type APIClient interface {
	ListRuns(ctx context.Context, page, limit int) (*models.ListRunsResponse, error)
	ListRunsLegacy(limit, offset int) ([]*models.RunResponse, error)
	GetRun(id string) (*models.RunResponse, error)
	GetUserInfo() (*models.UserInfo, error)
	GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error)
	ListRepositories(ctx context.Context) ([]models.APIRepository, error)
	GetAPIEndpoint() string
	VerifyAuth() (*models.UserInfo, error)
	CreateRunAPI(request *models.APIRunRequest) (*models.RunResponse, error)
	GetFileHashes(ctx context.Context) ([]models.FileHashEntry, error)
}
