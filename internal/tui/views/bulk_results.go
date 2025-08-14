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
	client     *api.Client
	cache      *cache.SimpleCache
	layout     *components.WindowLayout
	viewport   viewport.Model
	statusLine *components.StatusLine

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

	// Navigation state
	selectedTab    int    // 0 = successful, 1 = failed
	selectedRow    int    // Currently selected row in the active tab
	selectedButton int    // Currently selected button (0 = none, 1 = DASH)
	focusMode      string // "runs" or "buttons"

	// Key bindings
	keys BulkResultsKeyMap
}

// BulkResultsKeyMap defines the key bindings for the results view
type BulkResultsKeyMap struct {
	Up        key.Binding
	Down      key.Binding
	PageUp    key.Binding
	PageDown  key.Binding
	Tab       key.Binding
	Enter     key.Binding
	Back      key.Binding
	Dashboard key.Binding
	Quit      key.Binding
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
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch tab"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "select"),
		),
		Back: key.NewBinding(
			key.WithKeys("h", "esc"),
			key.WithHelp("h/esc", "back"),
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

	vp := viewport.New(80, 20) // Default size, will be updated
	vp.YPosition = 0

	v := &BulkResultsView{
		client:       client,
		cache:        cache,
		keys:         DefaultBulkResultsKeyMap(),
		originalRuns: make(map[int]BulkRunItem),
		selectedTab:  0,
		selectedRow:  0,
		focusMode:    "runs",
		viewport:     vp,
		statusLine:   components.NewStatusLine(),
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

	// Update viewport dimensions - account for box border and status line
	v.viewport.Width = v.width - 4   // Account for border and padding
	v.viewport.Height = v.height - 5 // Account for border, status line, and tabs

	// Update viewport content
	v.updateViewportContent()
}

// handleKeyMsg handles keyboard input
func (v *BulkResultsView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle navigation between runs and buttons
	if v.focusMode == "runs" {
		return v.handleRunKeys(msg)
	} else {
		return v.handleButtonKeys(msg)
	}
}

// handleRunKeys handles keys when focused on runs
func (v *BulkResultsView) handleRunKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case key.Matches(msg, v.keys.Up):
		v.navigateUp()
		v.ensureSelectedVisible()
		v.updateViewportContent()
		return v, nil

	case key.Matches(msg, v.keys.Down):
		v.navigateDown()
		v.ensureSelectedVisible()
		v.updateViewportContent()
		return v, nil

	case key.Matches(msg, v.keys.PageUp):
		v.viewport.HalfViewUp()
		return v, nil

	case key.Matches(msg, v.keys.PageDown):
		v.viewport.HalfViewDown()
		return v, nil

	case key.Matches(msg, v.keys.Tab):
		// Switch between successful and failed tabs
		if len(v.failed) > 0 {
			v.selectedTab = (v.selectedTab + 1) % 2
			v.selectedRow = 0
			v.updateViewportContent()
		}
		return v, nil

	case key.Matches(msg, v.keys.Enter):
		// Switch to button mode or activate current item
		if v.getItemCount() == 0 {
			// No items, switch to button mode directly
			v.focusMode = "buttons"
			v.selectedButton = 1
		} else {
			// Has items, switch to button mode
			v.focusMode = "buttons"
			v.selectedButton = 1
		}
		v.updateViewportContent()
		return v, nil

	default:
		// Pass to viewport for scrolling
		var cmd tea.Cmd
		v.viewport, cmd = v.viewport.Update(msg)
		return v, cmd
	}
}

// handleButtonKeys handles keys when focused on buttons
func (v *BulkResultsView) handleButtonKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Up):
		// Go back to runs list
		v.focusMode = "runs"
		v.updateViewportContent()
		return v, nil

	case key.Matches(msg, v.keys.Enter):
		// Activate selected button
		if v.selectedButton == 1 {
			// DASH button
			return v, func() tea.Msg {
				return messages.NavigateToDashboardMsg{}
			}
		}
		return v, nil

	case key.Matches(msg, v.keys.Back), msg.Type == tea.KeyEsc:
		// Go back to runs list
		v.focusMode = "runs"
		v.updateViewportContent()
		return v, nil

	case key.Matches(msg, v.keys.Dashboard), key.Matches(msg, v.keys.Quit):
		// Go to dashboard
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}

	default:
		return v, nil
	}
}

// navigateUp moves selection up
func (v *BulkResultsView) navigateUp() {
	if v.selectedRow > 0 {
		v.selectedRow--
	}
}

// navigateDown moves selection down
func (v *BulkResultsView) navigateDown() {
	maxRow := v.getItemCount() - 1
	if v.selectedRow < maxRow {
		v.selectedRow++
	}
}

// getItemCount returns the number of items in the current tab
func (v *BulkResultsView) getItemCount() int {
	if v.selectedTab == 0 {
		return len(v.successful)
	}
	return len(v.failed)
}

// ensureSelectedVisible ensures the selected item is visible in the viewport
func (v *BulkResultsView) ensureSelectedVisible() {
	// Calculate line position of selected item
	linePos := v.selectedRow * 3 // Each item takes ~3 lines

	// Ensure the selected item is visible
	if linePos < v.viewport.YOffset {
		v.viewport.SetYOffset(linePos)
	} else if linePos >= v.viewport.YOffset+v.viewport.Height-3 {
		v.viewport.SetYOffset(linePos - v.viewport.Height + 4)
	}
}

// updateViewportContent updates the content displayed in the viewport
func (v *BulkResultsView) updateViewportContent() {
	var content strings.Builder

	// Add tabs
	content.WriteString(v.renderTabs() + "\n\n")

	if v.selectedTab == 0 {
		// Show successful runs
		content.WriteString(v.renderSuccessfulRuns())
	} else {
		// Show failed runs
		content.WriteString(v.renderFailedRuns())
	}

	// Add navigation button if in button mode
	if v.focusMode == "buttons" {
		content.WriteString("\n\n")
		content.WriteString(v.renderButtons())
	}

	v.viewport.SetContent(content.String())
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

// renderSuccessfulRuns renders the list of successful runs
func (v *BulkResultsView) renderSuccessfulRuns() string {
	if len(v.successful) == 0 {
		return "  No successful runs"
	}

	var content strings.Builder

	// Style for selected row
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	for i, run := range v.successful {
		isSelected := v.focusMode == "runs" && i == v.selectedRow

		var line strings.Builder
		if isSelected {
			line.WriteString("▸ ")
		} else {
			line.WriteString("  ")
		}

		icon := "✅"
		status := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Render(run.Status)

		line.WriteString(fmt.Sprintf("%s Run #%d - %s [%s]", icon, run.ID, run.Title, status))

		if isSelected {
			content.WriteString(selectedStyle.Render(line.String()))
		} else {
			content.WriteString(normalStyle.Render(line.String()))
		}
		content.WriteString("\n")

		// Add details under selected item
		if isSelected {
			detailStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				PaddingLeft(5)
			content.WriteString(detailStyle.Render(fmt.Sprintf("Repository: %s", run.RepositoryName)))
			content.WriteString("\n")
		}
	}

	return content.String()
}

// renderFailedRuns renders the list of failed runs
func (v *BulkResultsView) renderFailedRuns() string {
	if len(v.failed) == 0 {
		return "  No failed runs"
	}

	var content strings.Builder

	// Style for selected row
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	for i, runErr := range v.failed {
		isSelected := v.focusMode == "runs" && i == v.selectedRow

		var line strings.Builder
		if isSelected {
			line.WriteString("▸ ")
		} else {
			line.WriteString("  ")
		}

		icon := "❌"

		// Get title from original run if available
		title := "Untitled"
		if original, ok := v.originalRuns[runErr.RequestIndex]; ok && original.Title != "" {
			title = original.Title
		}

		line.WriteString(fmt.Sprintf("%s %s - Failed", icon, title))

		if isSelected {
			content.WriteString(selectedStyle.Render(line.String()))
		} else {
			content.WriteString(normalStyle.Render(line.String()))
		}
		content.WriteString("\n")

		// Add details under selected item
		if isSelected {
			detailStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("244")).
				PaddingLeft(5)

			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				PaddingLeft(5)

			content.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", runErr.Message)))
			content.WriteString("\n")

			if runErr.ExistingRunId > 0 {
				content.WriteString(detailStyle.Render(fmt.Sprintf("Existing Run: #%d", runErr.ExistingRunId)))
				content.WriteString("\n")
			}
		}
	}

	return content.String()
}

// renderButtons renders navigation buttons
func (v *BulkResultsView) renderButtons() string {
	var content strings.Builder

	selectedBtnStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	normalBtnStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// DASH button
	if v.selectedButton == 1 {
		content.WriteString(selectedBtnStyle.Render("▸ ← [DASH]"))
	} else {
		content.WriteString(normalBtnStyle.Render("  ← [DASH]"))
	}

	return content.String()
}

// View renders the view
func (v *BulkResultsView) View() string {
	if v.width == 0 || v.height == 0 {
		return ""
	}

	// Create box for the viewport
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(0, 1).
		Width(v.width).
		Height(v.height - 3) // Account for status line

	// Check if content overflows and needs scrolling
	var scrollIndicator string
	content := v.viewport.View()
	totalLines := strings.Count(content, "\n")
	viewportCanShow := v.viewport.Height

	if totalLines > viewportCanShow {
		// Content overflows, show scroll indicator
		if v.viewport.AtTop() {
			scrollIndicator = "[TOP ↓]"
		} else if v.viewport.AtBottom() {
			scrollIndicator = "[↑ BOTTOM]"
		} else {
			percentScrolled := v.viewport.ScrollPercent()
			scrollIndicator = fmt.Sprintf("[↑ %d%% ↓]", int(percentScrolled*100))
		}
	}

	// Render viewport in box
	boxedContent := boxStyle.Render(v.viewport.View())

	// Status line with proper [RESULTS] format
	statusLine := v.renderStatusLine(scrollIndicator)

	return lipgloss.JoinVertical(lipgloss.Left, boxedContent, statusLine)
}

// renderStatusLine renders the status line with [RESULTS] format
func (v *BulkResultsView) renderStatusLine(scrollIndicator string) string {
	// Create formatter for consistent formatting
	formatter := components.NewStatusFormatter("RESULTS", v.width)

	// Help text based on current mode
	var helpText string
	if v.focusMode == "runs" {
		if len(v.failed) > 0 {
			helpText = "↑↓ nav • tab: switch • enter: buttons • h: back • q: dash"
		} else {
			helpText = "↑↓ nav • enter: buttons • h: back • q: dash"
		}
	} else {
		helpText = "enter: select [DASH] • ↑: back to list • esc: cancel • q: dash"
	}

	// Statistics summary
	stats := fmt.Sprintf("Total: %d | Success: %d | Failed: %d",
		v.statistics.Total,
		len(v.successful),
		len(v.failed))

	// Combine left content with view name
	leftContent := formatter.FormatViewName() + " " + stats

	// Right content is scroll indicator
	rightContent := scrollIndicator

	// Use status line directly
	return v.statusLine.
		SetWidth(v.width).
		SetLeft(leftContent).
		SetRight(rightContent).
		SetHelp(helpText).
		Render()
}
