package views

import (
	"time"

	"github.com/repobird/repobird-cli/internal/models"
)

// dashboardDataLoadedMsg is sent when dashboard data has been loaded
type dashboardDataLoadedMsg struct {
	repositories   []models.Repository
	allRuns        []*models.RunResponse
	detailsCache   map[string]*models.RunResponse
	error          error
	retryExhausted bool // Indicates if all retry attempts have been exhausted
}

// dashboardRepositorySelectedMsg is sent when a repository is selected
type dashboardRepositorySelectedMsg struct {
	repository *models.Repository
	runs       []*models.RunResponse
}

// dashboardUserInfoLoadedMsg is sent when user info has been loaded
type dashboardUserInfoLoadedMsg struct {
	userInfo *models.UserInfo
	error    error
}

// messageClearMsg is sent to clear status messages
type messageClearMsg struct{}

// syncFileHashesMsg is sent when file hash sync is completed
type syncFileHashesMsg struct{}

// yankBlinkMsg triggers yank blink animation
type yankBlinkMsg struct{}

// clearStatusMsg clears the status message
type clearStatusMsg time.Time

// gKeyTimeoutMsg handles g key timeout
type gKeyTimeoutMsg struct{}
