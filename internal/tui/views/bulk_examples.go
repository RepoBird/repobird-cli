package views

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
)

// ExampleConfig represents a configuration example with content and description
type ExampleConfig struct {
	Name        string
	Description string
	Format      string // "json", "yaml", "markdown"
	Content     string
}

// getExampleConfigs returns all available example configurations
func getExampleConfigs() []ExampleConfig {
	return []ExampleConfig{
		{
			Name:        "single_run.json",
			Description: "Single run - JSON format",
			Format:      "json",
			Content: `{
  "prompt": "Fix the authentication bug in the login flow",
  "repository": "org/repo",
  "source": "main",
  "target": "fix/auth-bug",
  "runType": "run",
  "title": "Fix authentication bug",
  "context": "Users are unable to login with valid credentials",
  "files": ["src/auth.js", "src/login.js"]
}`,
		},
		{
			Name:        "bulk_runs.json",
			Description: "Bulk runs - JSON format",
			Format:      "json",
			Content: `{
  "repositoryName": "owner/repo-name",
  "batchTitle": "Q1 2024 Bug Fixes",
  "runType": "run",
  "sourceBranch": "main",
  "force": false,
  "runs": [
    {
      "prompt": "Fix authentication timeout bug in login.js",
      "title": "Auth timeout fix",
      "context": "Users report getting logged out after 5 minutes",
      "target": "fix/auth-timeout"
    },
    {
      "prompt": "Update user profile validation to handle special characters",
      "title": "Profile validation update",
      "target": "fix/profile-validation"
    }
  ]
}`,
		},
		{
			Name:        "minimal_bulk.json",
			Description: "Minimal bulk - JSON (only required fields)",
			Format:      "json",
			Content: `{
  "repositoryName": "owner/repo-name",
  "runs": [
    {
      "prompt": "Fix authentication timeout bug in login.js"
    },
    {
      "prompt": "Update user profile validation"
    },
    {
      "prompt": "Add error handling to payment processing"
    }
  ]
}`,
		},
		{
			Name:        "single_run.yaml",
			Description: "Single run - YAML format",
			Format:      "yaml",
			Content: `# Full example with all fields
prompt: |
  Refactor the payment processing module with the following goals:
  - Improve error handling and retry logic
  - Add support for multiple payment providers
  - Implement idempotency for all transactions
  - Add comprehensive logging and monitoring
  
  Ensure all changes are backward compatible.
  
repository: fintech/payment-service
source: develop
target: refactor/payment-module
runType: approval
title: Refactor payment processing module
context: |
  Current issues:
  - Single point of failure with one payment provider
  - Inconsistent error handling
  - Missing transaction logs
  
  Tech stack: Node.js, TypeScript, PostgreSQL
  
files:
  - src/payments/processor.ts
  - src/payments/providers/
  - src/utils/retry.ts
  - tests/payments/`,
		},
		{
			Name:        "bulk_runs.yaml",
			Description: "Bulk runs - YAML format",
			Format:      "yaml",
			Content: `repositoryName: myapp/backend
batchTitle: API Improvements
runType: run
sourceBranch: main
runs:
  - prompt: Add input validation to all API endpoints
    title: API input validation
    target: feature/api-validation
    context: Prevent invalid data from reaching the database
    
  - prompt: Implement rate limiting for public endpoints
    title: Rate limiting
    target: feature/rate-limiting
    context: Protect against abuse and DDoS attacks
    
  - prompt: Add comprehensive API documentation
    title: API documentation
    target: docs/api-docs
    context: Generate OpenAPI/Swagger documentation`,
		},
		{
			Name:        "minimal.yaml",
			Description: "Minimal - YAML (only required fields)",
			Format:      "yaml",
			Content: `# Minimal YAML with defaults
repository: myapp/backend
prompt: Add input validation to the API endpoints
target: feature/api-validation
title: Add API input validation
# source defaults to "main"
# runType defaults to "run"`,
		},
		{
			Name:        "single_run.md",
			Description: "Single run - Markdown with frontmatter",
			Format:      "markdown",
			Content: `---
prompt: "Implement user authentication with JWT tokens"
repository: "acme/webapp"
source: "main"
target: "feature/jwt-auth"
runType: "run"
title: "Add JWT authentication system"
context: "Need secure authentication with JWT tokens"
files:
  - "src/auth/jwt.go"
  - "src/middleware/auth.go"
---

# JWT Authentication Implementation

## Overview
Implement a secure JWT-based authentication system for the web application.

## Requirements
- User login with email/password credentials
- Generate access tokens (15 min expiry) and refresh tokens (7 days)
- Implement token rotation on refresh
- Add middleware for protecting routes
- Store refresh tokens securely in database

## Technical Details
- Use RS256 algorithm for token signing
- Implement proper token validation
- Add rate limiting for login attempts
- Include user roles and permissions in token claims

## Security Considerations
- Never expose private keys
- Implement CSRF protection
- Use secure HTTP-only cookies for tokens
- Add token revocation mechanism`,
		},
		{
			Name:        "bulk_runs.md",
			Description: "Bulk runs - Markdown with frontmatter",
			Format:      "markdown",
			Content: `---
repositoryName: "acme/ecommerce"
batchTitle: "Security Enhancements"
runType: "approval"
sourceBranch: "develop"
runs:
  - prompt: "Implement OAuth 2.0 authentication"
    title: "OAuth 2.0 integration"
    target: "feature/oauth2"
    context: "Support login via Google, GitHub, and Microsoft"
    
  - prompt: "Add two-factor authentication"
    title: "2FA implementation"
    target: "feature/2fa"
    context: "TOTP-based 2FA using authenticator apps"
    
  - prompt: "Implement API key management"
    title: "API key system"
    target: "feature/api-keys"
    context: "Allow users to generate and manage API keys"
---

# Security Enhancement Project

## Objective
Implement comprehensive security features to protect user accounts and API access.

## Phase 1: OAuth 2.0 Integration
Support social login providers to simplify authentication while maintaining security.

## Phase 2: Two-Factor Authentication
Add optional 2FA for users who want extra account protection.

## Phase 3: API Key Management
Enable programmatic access with proper key rotation and scoping.`,
		},
	}
}

// renderExamples renders the examples view with file list and preview
func (v *BulkView) renderExamples() string {
	// Initialize layout if not done yet
	if v.layout == nil {
		v.layout = components.NewWindowLayout(v.width, v.height)
	}

	// Initialize double column layout for list + preview
	if v.doubleColumnLayout == nil {
		v.doubleColumnLayout = components.NewDoubleColumnLayout(v.width, v.height, &components.DoubleColumnConfig{
			LeftRatio:  0.4, // 40% for file list
			RightRatio: 0.6, // 60% for preview
			Gap:        1,
		})
	} else {
		v.doubleColumnLayout.Update(v.width, v.height)
	}

	// Get examples
	examples := getExampleConfigs()

	// Build left column (file list)
	var leftContent strings.Builder
	normalStyle := lipgloss.NewStyle()
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	
	leftContent.WriteString("📚 Example Configurations\n")
	leftContent.WriteString(strings.Repeat("─", 30) + "\n\n")
	
	for i, example := range examples {
		icon := "📄"
		if example.Format == "yaml" {
			icon = "📋"
		} else if example.Format == "markdown" {
			icon = "📝"
		}
		
		line := fmt.Sprintf("%s %s", icon, example.Description)
		
		if i == v.selectedExample {
			leftContent.WriteString(selectedStyle.Render("▸ " + line))
		} else {
			leftContent.WriteString(normalStyle.Render("  " + line))
		}
		leftContent.WriteString("\n")
	}
	
	leftContent.WriteString("\n")
	leftContent.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true).
		Render("↑↓ nav • y yank • h back"))

	// Build right column (preview with syntax highlighting)
	var rightContent strings.Builder
	if v.selectedExample < len(examples) {
		example := examples[v.selectedExample]
		
		// Add filename header
		headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63"))
		rightContent.WriteString(headerStyle.Render(example.Name))
		rightContent.WriteString("\n" + strings.Repeat("─", 40) + "\n\n")
		
		// Add content with basic syntax coloring
		rightContent.WriteString(v.syntaxHighlight(example.Content, example.Format))
	}

	// Get content dimensions for each column
	_, leftHeight, _, rightHeight := v.doubleColumnLayout.GetContentDimensions()
	
	// Ensure content fits in columns
	leftLines := strings.Split(leftContent.String(), "\n")
	if len(leftLines) > leftHeight {
		leftLines = leftLines[:leftHeight]
	}
	
	rightLines := strings.Split(rightContent.String(), "\n")
	if len(rightLines) > rightHeight {
		rightLines = rightLines[:rightHeight]
	}

	// Status line
	statusLine := v.renderStatusLine("EXAMPLES")

	// Use double column layout to render everything
	return v.doubleColumnLayout.RenderWithTitle(
		"Configuration Examples",
		strings.Join(leftLines, "\n"),
		strings.Join(rightLines, "\n"),
		statusLine,
	)
}

// syntaxHighlight applies basic syntax highlighting to config content
func (v *BulkView) syntaxHighlight(content, format string) string {
	// Simple syntax highlighting based on format
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33"))     // Blue for keys
	commentStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // Gray for comments
	
	lines := strings.Split(content, "\n")
	var highlighted []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Handle comments
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") {
			highlighted = append(highlighted, commentStyle.Render(line))
			continue
		}
		
		// Handle YAML/JSON keys (simplified)
		if format == "yaml" {
			if strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "-") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					highlighted = append(highlighted, 
						keyStyle.Render(parts[0]) + ":" + parts[1])
					continue
				}
			}
		} else if format == "json" {
			// Highlight JSON keys (between quotes before colon)
			if strings.Contains(line, `":`) {
				// Simple replacement for demonstration
				line = strings.ReplaceAll(line, `"prompt"`, keyStyle.Render(`"prompt"`))
				line = strings.ReplaceAll(line, `"repository"`, keyStyle.Render(`"repository"`))
				line = strings.ReplaceAll(line, `"title"`, keyStyle.Render(`"title"`))
				line = strings.ReplaceAll(line, `"target"`, keyStyle.Render(`"target"`))
				line = strings.ReplaceAll(line, `"runType"`, keyStyle.Render(`"runType"`))
				line = strings.ReplaceAll(line, `"source"`, keyStyle.Render(`"source"`))
				line = strings.ReplaceAll(line, `"context"`, keyStyle.Render(`"context"`))
				line = strings.ReplaceAll(line, `"files"`, keyStyle.Render(`"files"`))
				line = strings.ReplaceAll(line, `"runs"`, keyStyle.Render(`"runs"`))
				line = strings.ReplaceAll(line, `"repositoryName"`, keyStyle.Render(`"repositoryName"`))
				line = strings.ReplaceAll(line, `"batchTitle"`, keyStyle.Render(`"batchTitle"`))
				line = strings.ReplaceAll(line, `"sourceBranch"`, keyStyle.Render(`"sourceBranch"`))
				line = strings.ReplaceAll(line, `"force"`, keyStyle.Render(`"force"`))
			}
		}
		
		highlighted = append(highlighted, line)
	}
	
	return strings.Join(highlighted, "\n")
}

// handleExamplesKeys handles key events in examples mode
func (v *BulkView) handleExamplesKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	examples := getExampleConfigs()
	
	switch {
	case key.Matches(msg, v.keys.Quit) || msg.String() == "h":
		// Go back to instructions
		v.mode = ModeInstructions
		return v, nil
		
	case key.Matches(msg, v.keys.Up) || msg.String() == "k":
		if v.selectedExample > 0 {
			v.selectedExample--
		}
		return v, nil
		
	case key.Matches(msg, v.keys.Down) || msg.String() == "j":
		if v.selectedExample < len(examples)-1 {
			v.selectedExample++
		}
		return v, nil
		
	case msg.String() == "y":
		// Yank (copy) the entire example file content
		if v.selectedExample < len(examples) {
			example := examples[v.selectedExample]
			err := clipboard.WriteAll(example.Content)
			if err != nil {
				debug.LogToFilef("Failed to copy to clipboard: %v\n", err)
			} else {
				debug.LogToFilef("Copied example '%s' to clipboard\n", example.Name)
			}
		}
		return v, nil
		
	case msg.String() == "d":
		// Quick dashboard navigation
		return v, func() tea.Msg {
			return messages.NavigateToDashboardMsg{}
		}
	}
	
	return v, nil
}