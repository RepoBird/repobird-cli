package views

import (
	"github.com/repobird/repobird-cli/internal/models"
)

// dashboardDataLoadedMsg is sent when dashboard data has been loaded
type dashboardDataLoadedMsg struct {
	repositories []models.Repository
	allRuns      []*models.RunResponse
	detailsCache map[string]*models.RunResponse
	error        error
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

// yankBlinkMsg is sent to trigger yank blink animation
type yankBlinkMsg struct{}

// messageClearMsg is sent to clear status messages
type messageClearMsg struct{}

// gKeyTimeoutMsg is sent when the 'g' key timeout expires
type gKeyTimeoutMsg struct{}

// syncFileHashesMsg is sent when file hash sync is completed
type syncFileHashesMsg struct{}