package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
)

// BulkResultsView displays the results of a bulk run submission
type BulkResultsView struct {
	// Core components
	client   *api.Client
	cache    *cache.SimpleCache
	layout   *components.WindowLayout
	viewport viewport.Model

	// Dimensions
	width  int
	height int

	// Results data
	batchID    string
	batchTitle string
	repository string
	successful []dto.RunCreatedItem
	failed     []dto.RunError
	statistics dto.BulkStatistics

	// Original run configurations (for failed runs)
	originalRuns map[int]BulkRunItem // Indexed by requestIndex

	// UI state
	showDetails bool
	selectedTab int // 0 = successful, 1 = failed

	// Key bindings
	keys BulkResultsKeyMap
}

// BulkResultsKeyMap defines the key bindings for the results view
type BulkResultsKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Tab      key.Binding
	Details  key.Binding
	Back     key.Binding
	Dashboard key.Binding
	Quit     key.Binding
}

// DefaultBulkResultsKeyMap returns the default key bindings
func DefaultBulkResultsKeyMap() BulkResultsKeyMap {
	return BulkResultsKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch tab"),
		),
		Details: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "toggle details"),
		),
		Back: key.NewBinding(
			key.WithKeys("b", "esc"),
			key.WithHelp("b/esc", "back"),
		),
		Dashboard: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "dashboard"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

// NewBulkResultsView creates a new bulk results view
func NewBulkResultsView(client *api.Client, cache *cache.SimpleCache) *BulkResultsView {
	debug.LogToFilef("📊 Creating new BulkResultsView\n")
	
	v := &BulkResultsView{
		client:       client,
		cache:        cache,
		keys:         DefaultBulkResultsKeyMap(),
		originalRuns: make(map[int]BulkRunItem),
		selectedTab:  0,
		showDetails:  true,
	}

	// Load results from navigation context
	if batchID := cache.GetNavigationContext("batchID"); batchID != nil {
		if id, ok := batchID.(string); ok {
			v.batchID = id
		}
	}
	if batchTitle := cache.GetNavigationContext("batchTitle"); batchTitle != nil {
		if title, ok := batchTitle.(string); ok {
			v.batchTitle = title
		}
	}
	if repository := cache.GetNavigationContext("repository"); repository != nil {
		if repo, ok := repository.(string); ok {
			v.repository = repo
		}
	}
	if successful := cache.GetNavigationContext("successful"); successful != nil {
		if runs, ok := successful.([]dto.RunCreatedItem); ok {
			v.successful = runs
		}
	}
	if failed := cache.GetNavigationContext("failed"); failed != nil {
		if runs, ok := failed.([]dto.RunError); ok {
			v.failed = runs
		}
	}
	if stats := cache.GetNavigationContext("statistics"); stats != nil {
		if s, ok := stats.(dto.BulkStatistics); ok {
			v.statistics = s
		}
	}
	if originalRuns := cache.GetNavigationContext("originalRuns"); originalRuns != nil {
		if runs, ok := originalRuns.(map[int]BulkRunItem); ok {
			v.originalRuns = runs
		}
	}

	return v
}

// Init initializes the view
func (v *BulkResultsView) Init() tea.Cmd {
	debug.LogToFilef("📊 BulkResultsView.Init()\n")
	return nil
}

// Update handles messages and updates the view
func (v *BulkResultsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.handleWindowSizeMsg(msg)
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyMsg(msg)

	default:
		// Update viewport
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
}

// handleWindowSizeMsg handles terminal resize events
func (v *BulkResultsView) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	v.width = msg.Width
	v.height = msg.Height

	// Initialize or update layout
	if v.layout == nil {
		v.layout = components.NewWindowLayout(msg.Width, msg.Height)
		debug.LogToFilef("📐 RESULTS: Created layout with %dx%d\n", msg.Width, msg.Height)
	} else {
		v.layout.Update(msg.Width, msg.Height)
	}

	// Update viewport dimensions
	viewportWidth, viewportHeight := v.layout.GetViewportDimensions()
	v.viewport.Width = viewportWidth
	v.viewport.Height = viewportHeight - 4 // Account for tabs and status line

	// Update viewport content
	v.updateViewportContent()
}

// handleKeyMsg handles keyboard input
func (v *BulkResultsView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		// Go back to bulk view
		return v, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}

	case key.Matches(msg, v.keys.Dashboard), key.Matches(msg, v.keys.Quit):
		// Go to dashboard
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}

	case key.Matches(msg, v.keys.Tab):
		// Switch between successful and failed tabs
		if len(v.failed) > 0 {
			v.selectedTab = (v.selectedTab + 1) % 2
			v.updateViewportContent()
		}
		return v, nil

	case key.Matches(msg, v.keys.Details):
		// Toggle details view
		v.showDetails = !v.showDetails
		v.updateViewportContent()
		return v, nil

	default:
		// Pass to viewport for scrolling
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
}

// updateViewportContent updates the content displayed in the viewport
func (v *BulkResultsView) updateViewportContent() {
	var content strings.Builder

	if v.selectedTab == 0 {
		// Show successful runs
		content.WriteString(v.renderSuccessfulRuns())
	} else {
		// Show failed runs
		content.WriteString(v.renderFailedRuns())
	}

	v.viewport.SetContent(content.String())
}

// renderSuccessfulRuns renders the list of successful runs
func (v *BulkResultsView) renderSuccessfulRuns() string {
	if len(v.successful) == 0 {
		return "\n  No successful runs"
	}

	var content strings.Builder
	
	for i, run := range v.successful {
		icon := "✅"
		status := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(run.Status)
		
		if v.showDetails {
			content.WriteString(fmt.Sprintf("\n  %s Run #%d\n", icon, run.ID))
			content.WriteString(fmt.Sprintf("     Title: %s\n", run.Title))
			content.WriteString(fmt.Sprintf("     Status: %s\n", status))
			content.WriteString(fmt.Sprintf("     Repository: %s\n", run.RepositoryName))
			if i < len(v.successful)-1 {
				content.WriteString("\n")
			}
		} else {
			content.WriteString(fmt.Sprintf("  %s #%d: %s [%s]\n", icon, run.ID, run.Title, status))
		}
	}

	return content.String()
}

// renderFailedRuns renders the list of failed runs
func (v *BulkResultsView) renderFailedRuns() string {
	if len(v.failed) == 0 {
		return "\n  No failed runs"
	}

	var content strings.Builder
	
	for i, runErr := range v.failed {
		icon := "❌"
		
		if v.showDetails {
			content.WriteString(fmt.Sprintf("\n  %s Failed Run (Index: %d)\n", icon, runErr.RequestIndex))
			
			// Show original configuration if available
			if original, ok := v.originalRuns[runErr.RequestIndex]; ok {
				if original.Title != "" {
					content.WriteString(fmt.Sprintf("     Title: %s\n", original.Title))
				}
				if original.Target != "" {
					content.WriteString(fmt.Sprintf("     Target: %s\n", original.Target))
				}
			}
			
			// Show prompt (truncated if too long)
			prompt := runErr.Prompt
			if len(prompt) > 100 {
				prompt = prompt[:97] + "..."
			}
			content.WriteString(fmt.Sprintf("     Prompt: %s\n", prompt))
			
			// Show error message
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			content.WriteString(fmt.Sprintf("     Error: %s\n", errorStyle.Render(runErr.Message)))
			
			// Show existing run ID if duplicate
			if runErr.ExistingRunId > 0 {
				content.WriteString(fmt.Sprintf("     Existing Run ID: #%d\n", runErr.ExistingRunId))
			}
			
			if i < len(v.failed)-1 {
				content.WriteString("\n")
			}
		} else {
			// Compact view
			title := "Untitled"
			if original, ok := v.originalRuns[runErr.RequestIndex]; ok && original.Title != "" {
				title = original.Title
			}
			errorMsg := runErr.Message
			if len(errorMsg) > 50 {
				errorMsg = errorMsg[:47] + "..."
			}
			content.WriteString(fmt.Sprintf("  %s %s: %s\n", icon, title, errorMsg))
		}
	}

	return content.String()
}

// View renders the view
func (v *BulkResultsView) View() string {
	if v.layout == nil || v.width == 0 || v.height == 0 {
		return ""
	}

	if !v.layout.IsValidDimensions() {
		return v.layout.GetMinimalView("Bulk Results - Terminal too small")
	}

	// Create the main box
	boxStyle := v.layout.CreateStandardBox()
	titleStyle := v.layout.CreateTitleStyle()

	// Create title with statistics
	title := "📊 Bulk Run Results"
	if v.batchTitle != "" {
		title = fmt.Sprintf("📊 %s - Results", v.batchTitle)
	}

	// Create tabs
	tabs := v.renderTabs()

	// Create status line
	statusLine := v.renderStatusLine()

	// Combine all elements
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		tabs,
		v.viewport.View(),
		statusLine,
	)

	// Apply box styling
	boxWidth, boxHeight := v.layout.GetBoxDimensions()
	return boxStyle.
		Width(boxWidth).
		Height(boxHeight).
		Render(content)
}

// renderTabs renders the tab bar
func (v *BulkResultsView) renderTabs() string {
	activeTabStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 2)

	inactiveTabStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)

	var tabs []string

	// Successful tab
	successCount := len(v.successful)
	successTab := fmt.Sprintf("✅ Successful (%d)", successCount)
	if v.selectedTab == 0 {
		tabs = append(tabs, activeTabStyle.Render(successTab))
	} else {
		tabs = append(tabs, inactiveTabStyle.Render(successTab))
	}

	// Failed tab (only show if there are failures)
	if len(v.failed) > 0 {
		failedTab := fmt.Sprintf("❌ Failed (%d)", len(v.failed))
		if v.selectedTab == 1 {
			tabs = append(tabs, activeTabStyle.Render(failedTab))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(failedTab))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// renderStatusLine renders the status line with help text
func (v *BulkResultsView) renderStatusLine() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Build help text
	var helpItems []string
	
	if len(v.failed) > 0 {
		helpItems = append(helpItems, "tab: switch")
	}
	
	helpItems = append(helpItems,
		fmt.Sprintf("d: %s details", map[bool]string{true: "hide", false: "show"}[v.showDetails]),
		"b: back",
		"q: dashboard",
	)

	help := helpStyle.Render(strings.Join(helpItems, " • "))

	// Statistics summary
	statsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
	stats := fmt.Sprintf("Total: %d | Success: %d | Failed: %d",
		v.statistics.Total,
		len(v.successful),
		len(v.failed),
	)

	// Combine with proper spacing
	width, _ := v.layout.GetViewportDimensions()
	statusLeft := statsStyle.Render(stats)
	statusRight := help

	gap := width - lipgloss.Width(statusLeft) - lipgloss.Width(statusRight)
	if gap < 0 {
		gap = 0
	}

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		statusLeft,
		strings.Repeat(" ", gap),
		statusRight,
	)
}