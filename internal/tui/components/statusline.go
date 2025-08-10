package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusLine represents a universal status line component
type StatusLine struct {
	width        int
	leftContent  string
	rightContent string
	helpContent  string
	style        lipgloss.Style
	helpStyle    lipgloss.Style
}

// NewStatusLine creates a new status line component
func NewStatusLine() *StatusLine {
	return &StatusLine{
		style: lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1),
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
	}
}

// SetWidth sets the width of the status line
func (s *StatusLine) SetWidth(width int) *StatusLine {
	s.width = width
	return s
}

// SetLeft sets the left content of the status line
func (s *StatusLine) SetLeft(content string) *StatusLine {
	s.leftContent = content
	return s
}

// SetRight sets the right content of the status line
func (s *StatusLine) SetRight(content string) *StatusLine {
	s.rightContent = content
	return s
}

// SetHelp sets the help content of the status line
func (s *StatusLine) SetHelp(content string) *StatusLine {
	s.helpContent = content
	return s
}

// SetStyle sets the style of the status line
func (s *StatusLine) SetStyle(style lipgloss.Style) *StatusLine {
	s.style = style
	return s
}

// Render renders the status line
func (s *StatusLine) Render() string {
	if s.width <= 0 {
		return ""
	}

	// Calculate available space
	leftLen := lipgloss.Width(s.leftContent)
	rightLen := lipgloss.Width(s.rightContent)
	helpLen := lipgloss.Width(s.helpContent)
	
	// Create the main status line
	var statusContent string
	
	if s.helpContent != "" {
		// If we have help content, show it in the middle with dimmed style
		availableForHelp := s.width - leftLen - rightLen - 4 // Account for padding
		if availableForHelp > helpLen {
			// We have enough space for everything
			middlePadding := strings.Repeat(" ", availableForHelp - helpLen)
			statusContent = fmt.Sprintf("%s  %s%s  %s", 
				s.leftContent, 
				s.helpStyle.Render(s.helpContent),
				middlePadding,
				s.rightContent)
		} else if availableForHelp > 10 {
			// Truncate help content if needed
			truncatedHelp := truncateWithEllipsis(s.helpContent, availableForHelp)
			statusContent = fmt.Sprintf("%s  %s  %s", 
				s.leftContent, 
				s.helpStyle.Render(truncatedHelp),
				s.rightContent)
		} else {
			// Not enough space for help, just show left and right
			padding := strings.Repeat(" ", s.width - leftLen - rightLen)
			statusContent = fmt.Sprintf("%s%s%s", s.leftContent, padding, s.rightContent)
		}
	} else {
		// No help content, just left and right
		padding := strings.Repeat(" ", s.width - leftLen - rightLen)
		statusContent = fmt.Sprintf("%s%s%s", s.leftContent, padding, s.rightContent)
	}

	// Ensure the content fits the width
	if lipgloss.Width(statusContent) > s.width {
		statusContent = truncateWithEllipsis(statusContent, s.width)
	}

	return s.style.Width(s.width).Render(statusContent)
}

// truncateWithEllipsis truncates a string to fit within maxWidth with ellipsis
func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	return s[:maxWidth-3] + "..."
}

// DashboardStatusLine creates a status line for the dashboard
func DashboardStatusLine(width int, layoutName string, dataFreshness string, shortHelp string) string {
	statusLine := NewStatusLine().
		SetWidth(width).
		SetLeft(fmt.Sprintf("Dashboard: %s", layoutName)).
		SetRight(dataFreshness).
		SetHelp(shortHelp)
	
	return statusLine.Render()
}

// RunListStatusLine creates a status line for the run list view
func RunListStatusLine(width int, totalRuns int, filterStatus string, shortHelp string) string {
	left := fmt.Sprintf("Runs: %d total", totalRuns)
	if filterStatus != "" {
		left = fmt.Sprintf("Runs: %d total (%s)", totalRuns, filterStatus)
	}
	
	statusLine := NewStatusLine().
		SetWidth(width).
		SetLeft(left).
		SetHelp(shortHelp)
	
	return statusLine.Render()
}

// CreateRunStatusLine creates a status line for the create run view
func CreateRunStatusLine(width int, step string, shortHelp string) string {
	statusLine := NewStatusLine().
		SetWidth(width).
		SetLeft(fmt.Sprintf("Create Run: %s", step)).
		SetHelp(shortHelp)
	
	return statusLine.Render()
}

// DetailsStatusLine creates a status line for the run details view
func DetailsStatusLine(width int, runID string, status string, shortHelp string) string {
	statusLine := NewStatusLine().
		SetWidth(width).
		SetLeft(fmt.Sprintf("Run Details: %s", runID)).
		SetRight(status).
		SetHelp(shortHelp)
	
	return statusLine.Render()
}