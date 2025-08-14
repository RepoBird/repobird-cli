package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/utils"
)

// ExampleConfiguration represents a bulk run configuration example
type ExampleConfiguration struct {
	Name        string // Display name
	Description string // Short description
	Content     string // Full file content with ANSI highlighting
}

// ExamplesView displays bulk run configuration examples with preview
type ExamplesView struct {
	// Navigation
	client APIClient
	cache  *cache.SimpleCache

	// UI state
	width  int
	height int
	help   help.Model
	keys   examplesKeyMap

	// Layout
	layout *components.WindowLayout

	// Examples data
	examples        []ExampleConfiguration
	selectedExample int
	previewViewport viewport.Model

	// Clipboard state
	clipboardMsg    string
	clipboardExpiry time.Time
	yankAnimating   bool
	yankAnimExpiry  time.Time
}

// examplesKeyMap defines key bindings for examples view
type examplesKeyMap struct {
	Up       key.Binding
	Down     key.Binding
	PageUp   key.Binding
	PageDown key.Binding
	Yank     key.Binding
	Back     key.Binding
	Quit     key.Binding
	Help     key.Binding
}

// NewExamplesView creates a new examples view
func NewExamplesView(client APIClient, cache *cache.SimpleCache) *ExamplesView {
	vp := viewport.New(40, 20) // Will be resized
	vp.YPosition = 0

	return &ExamplesView{
		client:          client,
		cache:           cache,
		help:            help.New(),
		keys:            defaultExamplesKeyMap(),
		previewViewport: vp,
		examples:        getExampleConfigurations(),
		selectedExample: 0,
	}
}

func defaultExamplesKeyMap() examplesKeyMap {
	return examplesKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u", "K"),
			key.WithHelp("pgup/ctrl+u/K", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d", "J"),
			key.WithHelp("pgdn/ctrl+d/J", "page down"),
		),
		Yank: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yank/copy"),
		),
		Back: key.NewBinding(
			key.WithKeys("h", "esc", "backspace"),
			key.WithHelp("h/esc", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit to dashboard"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

func (v *ExamplesView) Init() tea.Cmd {
	// Initialize clipboard (will detect CGO availability)
	err := utils.InitClipboard()
	if err != nil {
		debug.LogToFilef("DEBUG: Failed to initialize clipboard: %v\n", err)
	}

	return nil
}

func (v *ExamplesView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		v.handleWindowSizeMsg(msg)
		return v, nil

	case tea.KeyMsg:
		return v.handleKeyInput(msg)

	case clearClipboardMsg:
		v.clipboardMsg = ""
		v.clipboardExpiry = time.Time{}
		return v, nil

	case yankAnimationMsg:
		v.yankAnimating = false
		return v, nil
	}

	// Update viewport
	var vpCmd tea.Cmd
	v.previewViewport, vpCmd = v.previewViewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return v, tea.Batch(cmds...)
}

func (v *ExamplesView) handleWindowSizeMsg(msg tea.WindowSizeMsg) {
	v.width = msg.Width
	v.height = msg.Height

	// Initialize layout if needed
	if v.layout == nil {
		v.layout = components.NewWindowLayout(msg.Width, msg.Height)
		debug.LogToFilef("📐 EXAMPLES INIT: Created layout with %dx%d 📐\n", msg.Width, msg.Height)
	} else {
		v.layout.Update(msg.Width, msg.Height)
	}

	// Top row: List (left) + Preview (right) 
	// Bottom row: Description (full width, compact)
	
	totalHeight := msg.Height - 1 // Account for status line (1 line)
	leftWidth := msg.Width / 3    // Left side takes 1/3 of width
	rightWidth := msg.Width - leftWidth // Right side takes remaining space (no gap)
	
	// Description gets very compact height, top row gets the rest
	descHeight := 2 // Minimal height for description (back to 2)
	topRowHeight := totalHeight - descHeight
	
	// Update preview viewport dimensions - account for borders properly
	v.previewViewport.Width = rightWidth - 2     // Account for border (2 not 4)
	v.previewViewport.Height = topRowHeight - 2  // Account for border (2 not 4)

	debug.LogToFilef("📐 EXAMPLES LAYOUT: Left=%d, Right=%d, TopRow=%d, Desc=%d, Preview=%dx%d 📐\n",
		leftWidth, rightWidth, topRowHeight, descHeight, v.previewViewport.Width, v.previewViewport.Height)

	// Update preview content for new dimensions
	v.updatePreviewContent()
}

func (v *ExamplesView) handleKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, v.keys.Back):
		return v, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}

	case key.Matches(msg, v.keys.Quit), msg.String() == "Q":
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}

	case key.Matches(msg, v.keys.Up):
		if v.selectedExample > 0 {
			v.selectedExample--
		} else {
			v.selectedExample = len(v.examples) - 1 // Wrap to bottom
		}
		v.updatePreviewContent()
		return v, nil

	case key.Matches(msg, v.keys.Down):
		if v.selectedExample < len(v.examples)-1 {
			v.selectedExample++
		} else {
			v.selectedExample = 0 // Wrap to top
		}
		v.updatePreviewContent()
		return v, nil

	case key.Matches(msg, v.keys.PageUp):
		v.previewViewport.LineUp(5)
		return v, nil

	case key.Matches(msg, v.keys.PageDown):
		v.previewViewport.LineDown(5)
		return v, nil

	case key.Matches(msg, v.keys.Yank):
		return v, v.handleYank()

	case key.Matches(msg, v.keys.Help):
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
	}

	return v, nil
}

func (v *ExamplesView) handleYank() tea.Cmd {
	if v.selectedExample >= 0 && v.selectedExample < len(v.examples) {
		selectedConfig := v.examples[v.selectedExample]
		content := selectedConfig.Content

		// Try to copy to clipboard
		err := utils.WriteToClipboard(content)
		if err != nil {
			debug.LogToFilef("DEBUG: Failed to copy to clipboard: %v\n", err)
			v.clipboardMsg = fmt.Sprintf("⚠ Copy failed: %v", err)
		} else {
			v.clipboardMsg = fmt.Sprintf("📋 Copied %s config", selectedConfig.Name)
		}

		// Set clipboard message expiry
		v.clipboardExpiry = time.Now().Add(3 * time.Second)

		// Start yank animation
		v.yankAnimating = true
		v.yankAnimExpiry = time.Now().Add(200 * time.Millisecond)

		var cmds []tea.Cmd

		// Clear clipboard message after delay
		cmds = append(cmds, func() tea.Msg {
			time.Sleep(3 * time.Second)
			return clearClipboardMsg{}
		})

		// Clear yank animation after short delay
		cmds = append(cmds, func() tea.Msg {
			time.Sleep(200 * time.Millisecond)
			return yankAnimationMsg{}
		})

		return tea.Batch(cmds...)
	}

	return nil
}

func (v *ExamplesView) updatePreviewContent() {
	if v.selectedExample >= 0 && v.selectedExample < len(v.examples) {
		content := v.examples[v.selectedExample].Content
		v.previewViewport.SetContent(content)
		v.previewViewport.GotoTop()
	}
}

func (v *ExamplesView) View() string {
	if v.layout == nil || v.width == 0 || v.height == 0 {
		return ""
	}

	if !v.layout.IsValidDimensions() {
		return v.layout.GetMinimalView("Examples - Loading...")
	}

	// Layout: Top row (list left + preview right), Bottom row (description full width)
	totalHeight := v.height - 1 // Account for status line (1 line)
	leftWidth := v.width / 3    // Left side takes 1/3
	rightWidth := v.width - leftWidth // Right side takes remaining space (no gap)
	
	// Heights
	descHeight := 2 // Minimal description height
	topRowHeight := totalHeight - descHeight

	// Top row: List (left) + Preview (right)
	listContent := v.renderExamplesList(leftWidth, topRowHeight)
	listBox := lipgloss.NewStyle().
		Width(leftWidth).
		Height(topRowHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Render(listContent)

	previewContent := v.renderPreview(rightWidth, topRowHeight)
	previewBox := lipgloss.NewStyle().
		Width(rightWidth).
		Height(topRowHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Render(previewContent)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, listBox, previewBox)

	// Bottom row: Description (full width, minimal spacing)
	descContent := v.renderDescription(v.width, descHeight)
	descBox := lipgloss.NewStyle().
		Width(v.width).
		Height(descHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("141")).
		Margin(0).         // Remove all margins
		Padding(0).        // Remove all padding
		Render(descContent)

	// Combine top and bottom rows with minimal spacing
	mainView := lipgloss.JoinVertical(lipgloss.Left, topRow, descBox)

	// Use global status line like other views
	statusLine := v.renderGlobalStatusLine()

	return lipgloss.JoinVertical(lipgloss.Left, mainView, statusLine)
}

func (v *ExamplesView) renderExamplesList(width, height int) string {
	var content strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
	content.WriteString(titleStyle.Render("📚 Configuration Examples"))
	content.WriteString("\n\n")

	// Example list - no descriptions here, they go in the bottom section
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	for i, example := range v.examples {
		var lineStyle lipgloss.Style
		prefix := "  "

		if i == v.selectedExample {
			lineStyle = selectedStyle
			prefix = "▸ "

			// Add yank animation with clipboard emoji
			if v.yankAnimating && time.Now().Before(v.yankAnimExpiry) {
				prefix = "📋 "
			}
		} else {
			lineStyle = normalStyle
		}

		line := fmt.Sprintf("%s%s", prefix, example.Name)
		content.WriteString(lineStyle.Render(line))
		content.WriteString("\n")
	}

	// Add padding for consistent height
	contentLines := strings.Count(content.String(), "\n")
	maxLines := height - 2 // Account for borders (reduced from 4 to 2)
	for i := contentLines; i < maxLines; i++ {
		content.WriteString("\n")
	}

	return content.String()
}

func (v *ExamplesView) renderPreview(width, height int) string {
	var content strings.Builder

	// Title with selected example name
	if v.selectedExample >= 0 && v.selectedExample < len(v.examples) {
		selectedConfig := v.examples[v.selectedExample]
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
		content.WriteString(titleStyle.Render(fmt.Sprintf("📄 %s", selectedConfig.Name)))
		content.WriteString("\n")

		// Scroll indicators
		scrollInfo := v.getScrollIndicators()
		infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		content.WriteString(infoStyle.Render(scrollInfo))
		content.WriteString("\n\n")
	}

	// Viewport content - should fit exactly within the allocated space
	content.WriteString(v.previewViewport.View())

	return content.String()
}

func (v *ExamplesView) getScrollIndicators() string {
	if v.previewViewport.TotalLineCount() <= v.previewViewport.Height {
		return "── Full Content ──"
	}

	// Calculate percentage
	if v.previewViewport.TotalLineCount() == 0 {
		return "── Empty ──"
	}

	topLine := v.previewViewport.YOffset
	bottomLine := topLine + v.previewViewport.Height
	totalLines := v.previewViewport.TotalLineCount()

	if topLine == 0 {
		return "── TOP ──"
	} else if bottomLine >= totalLines {
		return "── BOTTOM ──"
	} else {
		percentage := int(float64(topLine) / float64(totalLines-v.previewViewport.Height) * 100)
		return fmt.Sprintf("── %d%% ──", percentage)
	}
}

func (v *ExamplesView) renderDescription(width, height int) string {
	// No title, no padding - just compact content
	if v.selectedExample >= 0 && v.selectedExample < len(v.examples) {
		selectedConfig := v.examples[v.selectedExample]
		
		// Single line combining description + usage
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("250"))
		
		var usageText string
		switch {
		case strings.Contains(selectedConfig.Name, "Simple"):
			usageText = "• Quick setup"
		case strings.Contains(selectedConfig.Name, "Full"):
			usageText = "• Production ready"
		case strings.Contains(selectedConfig.Name, "YAML"):
			usageText = "• Multiline format"
		case strings.Contains(selectedConfig.Name, "Markdown"):
			usageText = "• Self-documenting"
		case strings.Contains(selectedConfig.Name, "Plan"):
			usageText = "• Safe planning"
		case strings.Contains(selectedConfig.Name, "Force"):
			usageText = "• Force execution"
		case strings.Contains(selectedConfig.Name, "Large"):
			usageText = "• High throughput"
		case strings.Contains(selectedConfig.Name, "Mixed"):
			usageText = "• Flexible config"
		default:
			usageText = "• Customizable"
		}
		
		// Single compact line
		return descStyle.Render(selectedConfig.Description + " " + usageText)
	}
	
	return "Select an example to view description"
}

func (v *ExamplesView) renderGlobalStatusLine() string {
	// Create formatter for consistent global status line format
	formatter := components.NewStatusFormatter("EXAMPLES", v.width)

	// Status message content
	var statusMsg string
	if v.clipboardMsg != "" && time.Now().Before(v.clipboardExpiry) {
		statusMsg = v.clipboardMsg
	} else {
		statusMsg = "↑↓:nav y:copy Ctrl+D/U,J/K:scroll h/ESC:back"
	}

	// Create consistent status line - let formatter handle the brackets
	statusLine := formatter.StandardStatusLine("", "", statusMsg)
	return statusLine.Render()
}

// Message types for examples view
type clearClipboardMsg struct{}
type yankAnimationMsg struct{}

// getExampleConfigurations returns all the bulk run configuration examples
func getExampleConfigurations() []ExampleConfiguration {
	return []ExampleConfiguration{
		{
			Name:        "Simple JSON",
			Description: "Basic bulk run with minimal fields",
			Content: `{
  "repositoryName": "org/my-project",
  "runs": [
    {
      "prompt": "Fix the authentication bug in login.js"
    },
    {
      "prompt": "Update validation for user profile forms"
    },
    {
      "prompt": "Add error handling to payment processing"
    }
  ]
}`,
		},
		{
			Name:        "Full JSON Configuration",
			Description: "Complete example with all optional fields",
			Content: `{
  "repositoryName": "org/my-project",
  "batchTitle": "Q1 2024 Bug Fixes",
  "runType": "run",
  "sourceBranch": "main",
  "force": false,
  "runs": [
    {
      "prompt": "Fix the authentication timeout bug in login.js",
      "title": "Auth timeout fix",
      "context": "Users report getting logged out after 5 minutes",
      "target": "fix/auth-timeout"
    },
    {
      "prompt": "Update user profile validation to handle special characters",
      "title": "Profile validation update",
      "context": "Support international usernames with accents and symbols",
      "target": "fix/profile-validation"
    }
  ]
}`,
		},
		{
			Name:        "YAML Configuration",
			Description: "YAML format with multiline strings",
			Content: `repositoryName: fintech/payment-service
batchTitle: Payment Module Refactoring
runType: approval
sourceBranch: develop
force: false
runs:
  - prompt: |
      Refactor the payment processing module:
      - Improve error handling and retry logic
      - Add support for multiple payment providers
      - Implement idempotency for all transactions
      
      Ensure all changes are backward compatible.
    title: Refactor payment processing module
    target: refactor/payment-module
    context: |
      Current issues:
      - Single point of failure with one payment provider
      - Inconsistent error handling
      - Missing transaction logs
      
      Tech stack: Node.js, TypeScript, PostgreSQL
      
  - prompt: Add comprehensive logging and monitoring to payment flows
    title: Payment monitoring
    target: feature/payment-monitoring
    context: Need visibility into transaction processing`,
		},
		{
			Name:        "Markdown with Frontmatter",
			Description: "Markdown format with YAML frontmatter",
			Content: `---
repositoryName: "acme/webapp"
batchTitle: "Authentication System Upgrade"
runType: "run"
sourceBranch: "main"
runs:
  - prompt: "Implement JWT authentication with refresh tokens"
    title: "JWT authentication system"
    target: "feature/jwt-auth"
  - prompt: "Add OAuth2 integration for Google and GitHub"
    title: "OAuth2 social login"
    target: "feature/oauth2"
---

# Authentication System Upgrade

## Overview
This batch implements a modern authentication system with JWT tokens and OAuth2 integration.

## Requirements
- Secure token-based authentication
- Social login integration
- Session management
- Password reset functionality

## Implementation Notes
All changes should maintain backward compatibility with existing sessions.`,
		},
		{
			Name:        "Plan Type Run",
			Description: "Planning mode for review before execution",
			Content: `{
  "repositoryName": "startup/backend-api",
  "batchTitle": "API Security Hardening",
  "runType": "plan",
  "sourceBranch": "main",
  "runs": [
    {
      "prompt": "Add rate limiting to all API endpoints",
      "title": "API rate limiting",
      "target": "security/rate-limiting",
      "context": "Prevent API abuse and DDoS attacks"
    },
    {
      "prompt": "Implement input validation and sanitization",
      "title": "Input validation",
      "target": "security/input-validation",
      "context": "Prevent SQL injection and XSS attacks"
    },
    {
      "prompt": "Add API key authentication for public endpoints",
      "title": "API key auth",
      "target": "security/api-keys",
      "context": "Control access to public API endpoints"
    }
  ]
}`,
		},
		{
			Name:        "Force Override",
			Description: "Override duplicate detection with force flag",
			Content: `{
  "repositoryName": "team/legacy-app",
  "batchTitle": "Critical Security Patches",
  "runType": "run",
  "sourceBranch": "main",
  "force": true,
  "runs": [
    {
      "prompt": "Patch SQL injection vulnerability in user search",
      "title": "SQL injection fix",
      "target": "hotfix/sql-injection",
      "context": "URGENT: Security vulnerability reported by security team"
    },
    {
      "prompt": "Update all dependencies to latest secure versions",
      "title": "Dependency updates",
      "target": "security/dep-updates",
      "context": "Multiple CVEs found in current dependencies"
    }
  ]
}`,
		},
		{
			Name:        "Large Batch",
			Description: "Maximum batch size (10 runs)",
			Content: `{
  "repositoryName": "enterprise/microservice",
  "batchTitle": "Code Quality Improvements",
  "runType": "run",
  "sourceBranch": "develop",
  "runs": [
    {
      "prompt": "Add unit tests for authentication service",
      "target": "test/auth-service"
    },
    {
      "prompt": "Add unit tests for user management service",
      "target": "test/user-service"
    },
    {
      "prompt": "Add unit tests for payment service",
      "target": "test/payment-service"
    },
    {
      "prompt": "Add unit tests for notification service",
      "target": "test/notification-service"
    },
    {
      "prompt": "Add integration tests for API endpoints",
      "target": "test/api-integration"
    },
    {
      "prompt": "Add error handling to database operations",
      "target": "improvement/db-errors"
    },
    {
      "prompt": "Implement graceful shutdown for all services",
      "target": "improvement/graceful-shutdown"
    },
    {
      "prompt": "Add health check endpoints for all services",
      "target": "feature/health-checks"
    },
    {
      "prompt": "Implement distributed tracing",
      "target": "feature/tracing"
    },
    {
      "prompt": "Add comprehensive logging to all services",
      "target": "improvement/logging"
    }
  ]
}`,
		},
		{
			Name:        "Mixed Formats",
			Description: "Combination of detailed and minimal run configurations",
			Content: `{
  "repositoryName": "product/mobile-app",
  "batchTitle": "Performance Optimization Sprint",
  "runType": "run",
  "runs": [
    {
      "prompt": "Optimize image loading and caching",
      "title": "Image optimization",
      "target": "perf/image-loading",
      "context": "Images take too long to load, especially on slow connections"
    },
    {
      "prompt": "Implement lazy loading for list views"
    },
    {
      "prompt": "Add compression to API responses",
      "title": "API compression",
      "target": "perf/api-compression"
    },
    {
      "prompt": "Optimize database queries"
    },
    {
      "prompt": "Implement caching layer for frequently accessed data",
      "context": "Cache user preferences, settings, and frequently viewed content"
    }
  ]
}`,
		},
	}
}