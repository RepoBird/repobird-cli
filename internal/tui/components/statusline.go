package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// MessageType represents different types of temporary messages
type MessageType int

const (
	MessageSuccess MessageType = iota
	MessageError
	MessageInfo
	MessageWarning
)

// StatusLine represents a universal status line component
type StatusLine struct {
	width               int
	leftContent         string
	rightContent        string
	helpContent         string
	style               lipgloss.Style
	helpStyle           lipgloss.Style
	tempMessage         string
	tempMessageTime     time.Time
	tempMessageDuration time.Duration
	tempMessageColor    lipgloss.Color
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

// ResetStyle resets the style to the default
func (s *StatusLine) ResetStyle() *StatusLine {
	s.style = lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)
	return s
}

// SetTemporaryMessage sets a temporary message with color and duration
func (s *StatusLine) SetTemporaryMessage(message string, color lipgloss.Color, duration time.Duration) *StatusLine {
	s.tempMessage = message
	s.tempMessageTime = time.Now()
	s.tempMessageDuration = duration
	s.tempMessageColor = color
	return s
}

// SetTemporaryMessageWithType sets a temporary message with predefined color for message type
func (s *StatusLine) SetTemporaryMessageWithType(message string, msgType MessageType, duration time.Duration) *StatusLine {
	color := GetMessageColor(msgType)
	return s.SetTemporaryMessage(message, color, duration)
}

// GetMessageColor returns the color for a given message type
func GetMessageColor(msgType MessageType) lipgloss.Color {
	switch msgType {
	case MessageSuccess:
		return lipgloss.Color("46") // Green
	case MessageError:
		return lipgloss.Color("196") // Red
	case MessageInfo:
		return lipgloss.Color("33") // Blue
	case MessageWarning:
		return lipgloss.Color("226") // Yellow
	default:
		return lipgloss.Color("252") // Default
	}
}

// HasActiveMessage returns true if there's an active temporary message
func (s *StatusLine) HasActiveMessage() bool {
	if s.tempMessage == "" {
		return false
	}
	return time.Since(s.tempMessageTime) < s.tempMessageDuration
}

// Render renders the status line
func (s *StatusLine) Render() string {
	if s.width <= 0 {
		return ""
	}

	// Check for active temporary message
	if s.HasActiveMessage() {
		// Create temporary message style - keep background consistent
		tempStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("235")). // Keep same background as normal
			Foreground(s.tempMessageColor).    // Only change text color
			Padding(0, 1)

		// For temporary messages, show the message in the help area
		// Keep left content for context
		statusContent := fmt.Sprintf("%s  %s",
			s.leftContent,
			s.tempMessage)

		// Pad to full width
		contentWidth := lipgloss.Width(statusContent)
		if contentWidth < s.width {
			statusContent += strings.Repeat(" ", s.width-contentWidth)
		} else if contentWidth > s.width {
			statusContent = truncateWithEllipsis(statusContent, s.width)
		}

		return tempStyle.
			Width(s.width).
			MaxWidth(s.width).
			MaxHeight(1).
			Render(statusContent)
	}

	// Truncate individual parts if they're too long
	maxPartWidth := s.width / 3
	leftContent := s.leftContent
	rightContent := s.rightContent
	helpContent := s.helpContent

	if lipgloss.Width(leftContent) > maxPartWidth {
		leftContent = truncateWithEllipsis(leftContent, maxPartWidth)
	}
	if lipgloss.Width(rightContent) > maxPartWidth {
		rightContent = truncateWithEllipsis(rightContent, maxPartWidth)
	}

	leftLen := lipgloss.Width(leftContent)
	rightLen := lipgloss.Width(rightContent)

	// Create the main status line
	var statusContent string

	if helpContent != "" {
		// Calculate available space for help
		availableForHelp := s.width - leftLen - rightLen - 4 // Account for padding
		if availableForHelp > 10 {
			// Truncate help to fit
			if lipgloss.Width(helpContent) > availableForHelp {
				helpContent = truncateWithEllipsis(helpContent, availableForHelp)
			}
			helpLen := lipgloss.Width(helpContent)
			middlePadding := strings.Repeat(" ", availableForHelp-helpLen)
			// Build the content without applying help style inline
			// This prevents style inheritance issues
			statusContent = fmt.Sprintf("%s  %s%s  %s",
				leftContent,
				helpContent,
				middlePadding,
				rightContent)
		} else {
			// Not enough space for help, just show left and right
			padding := s.width - leftLen - rightLen
			if padding < 0 {
				padding = 0
			}
			statusContent = fmt.Sprintf("%s%s%s",
				leftContent,
				strings.Repeat(" ", padding),
				rightContent)
		}
	} else {
		// No help content, just left and right
		padding := s.width - leftLen - rightLen
		if padding < 0 {
			padding = 0
		}
		statusContent = fmt.Sprintf("%s%s%s",
			leftContent,
			strings.Repeat(" ", padding),
			rightContent)
	}

	// Ensure content is exactly the right width by padding with spaces if needed
	contentWidth := lipgloss.Width(statusContent)
	if contentWidth < s.width {
		// Pad with spaces to fill the entire width
		statusContent += strings.Repeat(" ", s.width-contentWidth)
	} else if contentWidth > s.width {
		// Truncate if too long
		statusContent = truncateWithEllipsis(statusContent, s.width)
	}

	// Apply style without padding since we're handling width manually
	// This ensures the background color fills the entire line properly
	finalStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Width(s.width).
		MaxWidth(s.width).
		MaxHeight(1)
	
	return finalStyle.Render(statusContent)
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
	// Keep left side concise with bracket notation
	left := fmt.Sprintf("[%s]", layoutName)

	// Data freshness without brackets to save space
	right := dataFreshness

	// Calculate available space for help text
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	availableForHelp := width - leftWidth - rightWidth - 4 // 4 for padding/spacing

	// Dynamically adjust help text based on available space
	if availableForHelp < lipgloss.Width(shortHelp) {
		// Prioritize most important commands based on available space
		if availableForHelp < 30 {
			shortHelp = "?:help q:quit"
		} else if availableForHelp < 40 {
			shortHelp = "s:status ?:help q:quit"
		} else if availableForHelp < 50 {
			shortHelp = "n:new s:status ?:help q:quit"
		} else if availableForHelp < 60 {
			shortHelp = "n:new s:status y:copy ?:help q:quit"
		}
		// Otherwise use the full shortHelp passed in
	}

	statusLine := NewStatusLine().
		SetWidth(width).
		SetLeft(left).
		SetRight(right).
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
