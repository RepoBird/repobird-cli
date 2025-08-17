// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
)

// StatusView displays user account information and system status
type StatusView struct {
	client APIClient
	layout *components.WindowLayout
	keys   components.KeyMap

	// State
	width      int
	height     int
	userInfo   *models.UserInfo
	systemInfo StatusSystemInfo
	loading    bool
	error      error

	// Navigation state for scrollable list
	selectedRow int
	keyOffset   int // Horizontal scroll for keys
	valueOffset int // Horizontal scroll for values
	focusColumn int // 0: keys, 1: values

	// Data for display
	statusFields []string // Display values
	statusKeys   []string // Display keys
	fieldLines   []int    // Line numbers for each field

	// Copy feedback
	copiedMessage     string
	copiedMessageTime time.Time
	clipboardManager  components.ClipboardManager
}

// StatusSystemInfo contains system-level information for display
type StatusSystemInfo struct {
	RepositoryCount int
	TotalRuns       int
	RunningRuns     int
	CompletedRuns   int
	FailedRuns      int
	LastRefresh     time.Time
	APIEndpoint     string
	Connected       bool
}

// NewStatusView creates a new status view instance
func NewStatusView(client APIClient) *StatusView {
	return &StatusView{
		client:           client,
		layout:           components.NewWindowLayout(80, 24), // Default dimensions
		keys:             components.DefaultKeyMap,
		selectedRow:      0,
		focusColumn:      0,
		clipboardManager: components.NewClipboardManager(),
	}
}

// Init initializes the status view by loading user info
func (s *StatusView) Init() tea.Cmd {
	return tea.Batch(
		s.loadUserInfo(),
		s.loadSystemInfo(),
	)
}

// Update handles all messages for the status view
func (s *StatusView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return s.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		return s.handleKeyMsg(msg)

	case statusUserInfoLoadedMsg:
		return s.handleUserInfoLoaded(msg)

	case systemInfoLoadedMsg:
		return s.handleSystemInfoLoaded(msg)

	case statusErrorMsg:
		s.loading = false
		s.error = msg.error
		return s, nil

	case copySuccessMsg:
		return s.handleCopySuccess(msg)

	case components.ClipboardBlinkMsg:
		// Handle clipboard blink animation
		var clipCmd tea.Cmd
		s.clipboardManager, clipCmd = s.clipboardManager.Update(msg)
		return s, clipCmd

	case clearMessageMsg:
		s.copiedMessage = ""
		return s, nil
	}

	return s, nil
}

// View renders the status view
func (s *StatusView) View() string {
	if !s.layout.IsValidDimensions() {
		return s.layout.GetMinimalView("Status - Terminal too small")
	}

	if s.loading {
		return s.renderLoading()
	}

	if s.error != nil {
		return s.renderError()
	}

	return s.renderStatus()
}

// handleWindowSizeMsg handles terminal resize events
func (s *StatusView) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	s.width = msg.Width
	s.height = msg.Height
	s.layout.Update(msg.Width, msg.Height)

	debug.LogToFilef("🔄 STATUS: Window resized to %dx%d\n", msg.Width, msg.Height)
	return s, nil
}

// handleKeyMsg handles keyboard input
func (s *StatusView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "b":
		// Navigate back
		debug.LogToFilef("🔙 STATUS: Navigating back from status view\n")
		return s, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}

	case "r":
		// Refresh data
		debug.LogToFilef("🔄 STATUS: Refreshing status data\n")
		s.loading = true
		s.error = nil
		return s, tea.Batch(
			s.loadUserInfo(),
			s.loadSystemInfo(),
		)

	case "j", "down":
		if s.selectedRow < len(s.statusFields)-1 {
			s.selectedRow++
			s.resetHorizontalScroll()
		} else if len(s.statusFields) > 0 {
			// Wrap around to the first item
			s.selectedRow = 0
			s.resetHorizontalScroll()
		}
		return s, nil

	case "k", "up":
		if s.selectedRow > 0 {
			s.selectedRow--
			s.resetHorizontalScroll()
		} else if len(s.statusFields) > 0 {
			// Wrap around to the last item
			s.selectedRow = len(s.statusFields) - 1
			s.resetHorizontalScroll()
		}
		return s, nil

	case "g":
		s.selectedRow = 0
		s.resetHorizontalScroll()
		return s, nil

	case "G":
		if len(s.statusFields) > 0 {
			s.selectedRow = len(s.statusFields) - 1
			s.resetHorizontalScroll()
		}
		return s, nil

	case "h", "left":
		if s.focusColumn == 1 {
			s.focusColumn = 0
		} else {
			if s.keyOffset > 0 {
				s.keyOffset--
			}
		}
		return s, nil

	case "l", "right":
		if s.focusColumn == 0 {
			s.focusColumn = 1
		} else {
			s.scrollValueRight()
		}
		return s, nil

	case "y":
		return s.copyCurrentField()

	case "Y":
		return s.copyAllFields()
	}

	return s, nil
}

// resetHorizontalScroll resets horizontal scroll offsets when moving rows
func (s *StatusView) resetHorizontalScroll() {
	s.keyOffset = 0
	s.valueOffset = 0
}

// scrollValueRight scrolls the value column to the right if there's more content
func (s *StatusView) scrollValueRight() {
	if s.selectedRow >= 0 && s.selectedRow < len(s.statusFields) {
		value := s.statusFields[s.selectedRow]
		valueMaxWidth := 40 // Available width for value column

		if len(value) > s.valueOffset+valueMaxWidth {
			s.valueOffset++
			debug.LogToFilef("🔄 STATUS: Scrolling value to offset %d\n", s.valueOffset)
		}
	}
}

// renderLoading renders the loading state
func (s *StatusView) renderLoading() string {
	boxStyle := s.layout.CreateStandardBox()
	titleStyle := s.layout.CreateTitleStyle()
	contentStyle := s.layout.CreateContentStyle()

	title := titleStyle.Render("Status Information")
	content := contentStyle.Render("Loading status information...")

	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

// renderError renders the error state
func (s *StatusView) renderError() string {
	boxStyle := s.layout.CreateStandardBox()
	titleStyle := s.layout.CreateTitleStyle()
	contentStyle := s.layout.CreateContentStyle()

	title := titleStyle.Render("Status Information - Error")

	errorText := fmt.Sprintf("Error loading status: %v\n\nPress 'r' to retry or 'q' to go back", s.error)
	content := contentStyle.Render(errorText)

	return boxStyle.Render(lipgloss.JoinVertical(lipgloss.Left, title, content))
}

// renderStatus renders the main status display
func (s *StatusView) renderStatus() string {
	if len(s.statusFields) == 0 {
		s.initializeStatusFields()
	}

	boxStyle := s.layout.CreateStandardBox()
	titleStyle := s.layout.CreateTitleStyle()

	title := titleStyle.Render("System Status & Account Information")

	content := s.renderStatusContent()

	// Create status line at bottom
	statusLine := s.renderStatusLine()

	mainContent := lipgloss.JoinVertical(lipgloss.Left, title, content)

	return lipgloss.JoinVertical(lipgloss.Left,
		boxStyle.Render(mainContent),
		statusLine)
}

// renderStatusContent renders the scrollable status information
func (s *StatusView) renderStatusContent() string {
	if len(s.statusFields) == 0 {
		return "No status information available"
	}

	contentWidth, contentHeight := s.layout.GetContentDimensions()

	var lines []string

	// Add section headers and fields
	currentSection := ""
	for i, key := range s.statusKeys {
		if i >= len(s.statusFields) {
			break
		}

		value := s.statusFields[i]

		// Add section breaks
		// Check for Account Information section (triggered by Name, Email, or Account Tier)
		if (key == "Name:" || key == "Email:" || key == "Account Tier:") && currentSection != "account" {
			// Add empty line before Account Information section for spacing
			lines = append(lines, "") // Empty line before section
			lines = append(lines, s.renderSectionHeader("Account Information"))
			currentSection = "account"
		} else if key == "Repositories:" && currentSection != "system" {
			lines = append(lines, "") // Empty line
			lines = append(lines, s.renderSectionHeader("System Information"))
			currentSection = "system"
		} else if key == "API Endpoint:" && currentSection != "connection" {
			lines = append(lines, "") // Empty line
			lines = append(lines, s.renderSectionHeader("Connection Information"))
			currentSection = "connection"
		}

		// Render the field line
		fieldLine := s.renderFieldLine(i, key, value, contentWidth)
		lines = append(lines, fieldLine)
	}

	// Join lines and ensure it fits within content height
	allContent := strings.Join(lines, "\n")
	contentLines := strings.Split(allContent, "\n")

	// Truncate if too many lines
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}

	return strings.Join(contentLines, "\n")
}

// renderSectionHeader renders a section header
func (s *StatusView) renderSectionHeader(title string) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Underline(true)

	return headerStyle.Render(title)
}

// renderFieldLine renders a single field line with key-value pair
func (s *StatusView) renderFieldLine(index int, key, value string, maxWidth int) string {
	isSelected := index == s.selectedRow

	// Calculate column widths
	keyWidth := 20
	valueWidth := maxWidth - keyWidth - 3 // 3 for spacing

	if valueWidth < 10 {
		valueWidth = 10
	}

	// Apply horizontal scrolling
	displayKey := key
	if len(displayKey) > s.keyOffset {
		displayKey = displayKey[s.keyOffset:]
	}
	if len(displayKey) > keyWidth {
		displayKey = displayKey[:keyWidth-3] + "..."
	}

	displayValue := value
	if len(displayValue) > s.valueOffset {
		displayValue = displayValue[s.valueOffset:]
	}
	if len(displayValue) > valueWidth {
		displayValue = displayValue[:valueWidth-3] + "..."
	}

	// Style based on selection and focus
	var keyStyle, valueStyle lipgloss.Style

	if isSelected {
		if s.focusColumn == 0 {
			// Key column focused
			keyStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255")).
				Width(keyWidth)
			valueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Width(valueWidth)
		} else {
			// Value column focused
			keyStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Width(keyWidth)
			valueStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("255")).
				Width(valueWidth)
		}

		// Add yank blink effect
		if s.clipboardManager.ShouldHighlight() {
			if s.focusColumn == 0 {
				keyStyle = keyStyle.Background(lipgloss.Color("10")) // Green flash
			} else {
				valueStyle = valueStyle.Background(lipgloss.Color("10")) // Green flash
			}
		}
	} else {
		keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(keyWidth)
		valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Width(valueWidth)
	}

	formattedKey := keyStyle.Render(displayKey)
	formattedValue := valueStyle.Render(displayValue)

	return lipgloss.JoinHorizontal(lipgloss.Left, formattedKey, " ", formattedValue)
}

// renderStatusLine renders the status line at the bottom
func (s *StatusView) renderStatusLine() string {
	helpText := "[j/k]navigate [y]copy [Y]copy all [r]refresh [h]back [q]dashboard"

	// Create formatter for consistent formatting
	formatter := components.NewStatusFormatter("STATUS", s.width)

	// Format left and right content
	leftContent := formatter.FormatViewName()
	rightContent := fmt.Sprintf("Line %d/%d", s.selectedRow+1, len(s.statusFields))

	// Create status line using consistent component
	statusLine := formatter.StandardStatusLine(leftContent, rightContent, helpText)

	// Handle copy message if active
	if s.copiedMessage != "" && time.Since(s.copiedMessageTime) < 2*time.Second {
		statusLine.SetTemporaryMessage(s.copiedMessage, components.GetMessageColor(components.MessageSuccess), 2*time.Second)
	}

	return statusLine.Render()
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// initializeStatusFields populates the status fields from user info and system info
func (s *StatusView) initializeStatusFields() {
	s.statusFields = []string{}
	s.statusKeys = []string{}
	s.fieldLines = []int{}

	lineNum := 0

	// User Info fields
	if s.userInfo != nil {
		if s.userInfo.Name != "" {
			s.statusKeys = append(s.statusKeys, "Name:")
			s.statusFields = append(s.statusFields, s.userInfo.Name)
			s.fieldLines = append(s.fieldLines, lineNum)
			lineNum++
		}
		if s.userInfo.Email != "" {
			s.statusKeys = append(s.statusKeys, "Email:")
			s.statusFields = append(s.statusFields, s.userInfo.Email)
			s.fieldLines = append(s.fieldLines, lineNum)
			lineNum++
		}
		if s.userInfo.GithubUsername != "" {
			s.statusKeys = append(s.statusKeys, "GitHub:")
			s.statusFields = append(s.statusFields, s.userInfo.GithubUsername)
			s.fieldLines = append(s.fieldLines, lineNum)
			lineNum++
		}

		// Account tier
		tierDisplay := strings.Title(strings.ToLower(s.userInfo.Tier))
		if tierDisplay == "" {
			tierDisplay = "Basic"
		}
		s.statusKeys = append(s.statusKeys, "Account Tier:")
		s.statusFields = append(s.statusFields, tierDisplay)
		s.fieldLines = append(s.fieldLines, lineNum)
		lineNum++

		// Usage information - show pro runs as "Runs" and plan runs as "Plan Runs"
		// Pro runs (displayed as "Runs")
		if s.userInfo.ProTotalRuns > 0 {
			s.statusKeys = append(s.statusKeys, "Runs:")
			s.statusFields = append(s.statusFields, fmt.Sprintf("%d/%d", s.userInfo.RemainingProRuns, s.userInfo.ProTotalRuns))
			s.fieldLines = append(s.fieldLines, lineNum)
			lineNum++
		}

		// Plan runs (displayed as "Plan Runs")
		if s.userInfo.PlanTotalRuns > 0 {
			s.statusKeys = append(s.statusKeys, "Plan Runs:")
			s.statusFields = append(s.statusFields, fmt.Sprintf("%d/%d", s.userInfo.RemainingPlanRuns, s.userInfo.PlanTotalRuns))
			s.fieldLines = append(s.fieldLines, lineNum)
			lineNum++
		}

		// Calculate usage percentage (combined)
		totalRuns := s.userInfo.ProTotalRuns + s.userInfo.PlanTotalRuns
		remainingRuns := s.userInfo.RemainingProRuns + s.userInfo.RemainingPlanRuns
		if totalRuns > 0 {
			usedRuns := totalRuns - remainingRuns
			percentage := float64(usedRuns) / float64(totalRuns) * 100

			var usageValue string
			if percentage >= 90 {
				usageValue = fmt.Sprintf("%.1f%% ⚠️", percentage)
			} else if percentage >= 75 {
				usageValue = fmt.Sprintf("%.1f%% ⚡", percentage)
			} else {
				usageValue = fmt.Sprintf("%.1f%% ✅", percentage)
			}

			s.statusKeys = append(s.statusKeys, "Usage:")
			s.statusFields = append(s.statusFields, usageValue)
			s.fieldLines = append(s.fieldLines, lineNum)
			lineNum++
		}

		// Show unlimited for enterprise tier
		tierUpper := strings.ToUpper(s.userInfo.Tier)
		if tierUpper == "ENTERPRISE" {
			s.statusKeys = append(s.statusKeys, "Runs Available:")
			s.statusFields = append(s.statusFields, "Unlimited ♾️")
			s.fieldLines = append(s.fieldLines, lineNum)
			lineNum++
		}
	}

	// System info
	s.statusKeys = append(s.statusKeys, "Repositories:")
	s.statusFields = append(s.statusFields, fmt.Sprintf("%d", s.systemInfo.RepositoryCount))
	s.fieldLines = append(s.fieldLines, lineNum)
	lineNum++

	s.statusKeys = append(s.statusKeys, "Total Runs:")
	s.statusFields = append(s.statusFields, fmt.Sprintf("%d", s.systemInfo.TotalRuns))
	s.fieldLines = append(s.fieldLines, lineNum)
	lineNum++

	s.statusKeys = append(s.statusKeys, "Run Status:")
	statusBreakdown := fmt.Sprintf("🔄 %d  ✅ %d  ❌ %d",
		s.systemInfo.RunningRuns, s.systemInfo.CompletedRuns, s.systemInfo.FailedRuns)
	s.statusFields = append(s.statusFields, statusBreakdown)
	s.fieldLines = append(s.fieldLines, lineNum)
	lineNum++

	// Last refresh time
	if !s.systemInfo.LastRefresh.IsZero() {
		refreshText := fmt.Sprintf("%s ago", time.Since(s.systemInfo.LastRefresh).Truncate(time.Second))
		s.statusKeys = append(s.statusKeys, "Last Refresh:")
		s.statusFields = append(s.statusFields, refreshText)
		s.fieldLines = append(s.fieldLines, lineNum)
		lineNum++
	}

	// API connection info
	s.statusKeys = append(s.statusKeys, "API Endpoint:")
	s.statusFields = append(s.statusFields, s.systemInfo.APIEndpoint)
	s.fieldLines = append(s.fieldLines, lineNum)
	lineNum++

	s.statusKeys = append(s.statusKeys, "Status:")
	connectionStatus := "Connected ✅"
	if !s.systemInfo.Connected {
		connectionStatus = "Disconnected ❌"
	}
	s.statusFields = append(s.statusFields, connectionStatus)
	s.fieldLines = append(s.fieldLines, lineNum)

	// Ensure we have at least one field selected
	if len(s.statusFields) > 0 && s.selectedRow >= len(s.statusFields) {
		s.selectedRow = 0
	}
}

// Message types for async operations
type statusUserInfoLoadedMsg struct {
	userInfo *models.UserInfo
}

type systemInfoLoadedMsg struct {
	systemInfo StatusSystemInfo
}

type statusErrorMsg struct {
	error error
}

type copySuccessMsg struct {
	text string
}

type clearMessageMsg struct{}

// loadUserInfo loads user information from the API
func (s *StatusView) loadUserInfo() tea.Cmd {
	return func() tea.Msg {
		userInfo, err := s.client.GetUserInfo()
		if err != nil {
			return statusErrorMsg{error: err}
		}
		return statusUserInfoLoadedMsg{userInfo: userInfo}
	}
}

// loadSystemInfo loads system information from API
func (s *StatusView) loadSystemInfo() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Load repositories
		repositories, repoErr := s.client.ListRepositories(ctx)
		repositoryCount := 0
		if repoErr == nil {
			repositoryCount = len(repositories)
		}

		// Load runs to calculate statistics
		runsResp, runsErr := s.client.ListRuns(ctx, 1, 1000) // Get up to 1000 runs for stats
		var allRuns []*models.RunResponse
		if runsErr == nil && runsResp != nil {
			allRuns = runsResp.Data
		}

		// Calculate run statistics
		totalRuns := len(allRuns)
		var runningRuns, completedRuns, failedRuns int

		for _, run := range allRuns {
			switch run.Status {
			case "RUNNING", "PENDING":
				runningRuns++
			case "DONE":
				completedRuns++
			case "FAILED", "CANCELLED":
				failedRuns++
			}
		}

		// Determine connection status
		connected := repoErr == nil || runsErr == nil

		systemInfo := StatusSystemInfo{
			RepositoryCount: repositoryCount,
			TotalRuns:       totalRuns,
			RunningRuns:     runningRuns,
			CompletedRuns:   completedRuns,
			FailedRuns:      failedRuns,
			LastRefresh:     time.Now(),
			APIEndpoint:     s.client.GetAPIEndpoint(),
			Connected:       connected,
		}

		// If there were errors, report them
		if repoErr != nil && runsErr != nil {
			return statusErrorMsg{error: fmt.Errorf("failed to load system info: %v", repoErr)}
		}

		return systemInfoLoadedMsg{systemInfo: systemInfo}
	}
}

// handleUserInfoLoaded handles successful user info loading
func (s *StatusView) handleUserInfoLoaded(msg statusUserInfoLoadedMsg) (tea.Model, tea.Cmd) {
	s.userInfo = msg.userInfo
	s.loading = false
	s.error = nil
	s.initializeStatusFields()

	debug.LogToFilef("✅ STATUS: User info loaded for %s\n", s.userInfo.Email)
	return s, nil
}

// handleSystemInfoLoaded handles successful system info loading
func (s *StatusView) handleSystemInfoLoaded(msg systemInfoLoadedMsg) (tea.Model, tea.Cmd) {
	s.systemInfo = msg.systemInfo
	s.loading = false
	s.error = nil
	s.initializeStatusFields()

	debug.LogToFilef("✅ STATUS: System info loaded\n")
	return s, nil
}

// copyCurrentField copies the currently selected field to clipboard
func (s *StatusView) copyCurrentField() (tea.Model, tea.Cmd) {
	if s.selectedRow >= 0 && s.selectedRow < len(s.statusFields) {
		var textToCopy string
		if s.focusColumn == 0 && s.selectedRow < len(s.statusKeys) {
			// Copy the key (without the colon)
			textToCopy = strings.TrimSuffix(s.statusKeys[s.selectedRow], ":")
		} else {
			// Copy the value
			textToCopy = s.statusFields[s.selectedRow]
		}

		return s, s.copyToClipboard(textToCopy)
	}
	return s, nil
}

// copyAllFields copies all status information to clipboard
func (s *StatusView) copyAllFields() (tea.Model, tea.Cmd) {
	var allText strings.Builder
	allText.WriteString("Status Information\n")
	allText.WriteString("==================\n\n")

	for i, key := range s.statusKeys {
		if i < len(s.statusFields) {
			allText.WriteString(fmt.Sprintf("%s %s\n", key, s.statusFields[i]))
		}
	}

	return s, s.copyToClipboard(allText.String())
}

// copyToClipboard copies text to clipboard and shows feedback
func (s *StatusView) copyToClipboard(text string) tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual clipboard copy
		// For now, just return success message
		return copySuccessMsg{text: text}
	}
}

// handleCopySuccess handles successful clipboard copy
func (s *StatusView) handleCopySuccess(msg copySuccessMsg) (tea.Model, tea.Cmd) {
	s.copiedMessage = fmt.Sprintf("Copied: %s", msg.text)
	if len(s.copiedMessage) > 50 {
		s.copiedMessage = s.copiedMessage[:47] + "..."
	}
	s.copiedMessageTime = time.Now()

	// Start blink animation using clipboard manager
	cmd, err := s.clipboardManager.CopyWithBlink(msg.text, "")
	if err != nil {
		// Error already handled by clipboard operation, just show message
		return s, s.startMessageClearTimer(2 * time.Second)
	}

	// Start blink animation and clear timer
	return s, tea.Batch(
		cmd,
		s.startMessageClearTimer(2*time.Second),
	)
}

// startMessageClearTimer starts a timer to clear the copied message
func (s *StatusView) startMessageClearTimer(duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(t time.Time) tea.Msg {
		return clearMessageMsg{}
	})
}
