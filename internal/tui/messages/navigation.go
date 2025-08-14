package messages

import "github.com/repobird/repobird-cli/internal/models"

// NavigationMsg is the base interface for all navigation messages
type NavigationMsg interface {
	IsNavigation() bool
}

// NavigateToCreateMsg requests navigation to the create run view
type NavigateToCreateMsg struct {
	SelectedRepository string // Optional context from dashboard
}

// NavigateToDetailsMsg requests navigation to the run details view
type NavigateToDetailsMsg struct {
	RunID      string
	FromCreate bool // Optional context indicating source
	RunData    *models.RunResponse // Optional: cached run data to avoid API call
}

// NavigateToDashboardMsg requests navigation to the dashboard view
type NavigateToDashboardMsg struct{}

// NavigateToListMsg requests navigation to the run list view
type NavigateToListMsg struct {
	SelectedIndex int // Optional: restore selection
}

// NavigateToBulkMsg requests navigation to the bulk operations view
type NavigateToBulkMsg struct{}

// NavigateToStatusMsg requests navigation to the status/user info view
type NavigateToStatusMsg struct{}

// NavigateToFileViewerMsg requests navigation to the file viewer
type NavigateToFileViewerMsg struct{}

// NavigateToHelpMsg requests navigation to the help view
type NavigateToHelpMsg struct{}

// NavigateToExamplesMsg requests navigation to the examples view
type NavigateToExamplesMsg struct{}

// NavigateBackMsg requests navigation to the previous view in the stack
type NavigateBackMsg struct{}

// NavigateToErrorMsg requests navigation to an error view
type NavigateToErrorMsg struct {
	Error       error
	Message     string
	Recoverable bool   // Can user go back?
	ReturnTo    string // View to return to after acknowledgment
}

// Implement NavigationMsg interface for all messages
func (NavigateToCreateMsg) IsNavigation() bool     { return true }
func (NavigateToDetailsMsg) IsNavigation() bool    { return true }
func (NavigateToDashboardMsg) IsNavigation() bool  { return true }
func (NavigateToListMsg) IsNavigation() bool       { return true }
func (NavigateToBulkMsg) IsNavigation() bool       { return true }
func (NavigateToStatusMsg) IsNavigation() bool     { return true }
func (NavigateToFileViewerMsg) IsNavigation() bool { return true }
func (NavigateToHelpMsg) IsNavigation() bool       { return true }
func (NavigateToExamplesMsg) IsNavigation() bool   { return true }
func (NavigateBackMsg) IsNavigation() bool         { return true }
func (NavigateToErrorMsg) IsNavigation() bool      { return true }
