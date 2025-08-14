package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/utils"
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
	isLoading           bool
	loadingSpinner      spinner.Model
}

// NewStatusLine creates a new status line component
func NewStatusLine() *StatusLine {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // Bright yellow for better visibility

	return &StatusLine{
		style: lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1),
		helpStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),
		loadingSpinner: s,
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

// SetLoading sets the loading state of the status line
func (s *StatusLine) SetLoading(loading bool) *StatusLine {
	s.isLoading = loading
	return s
}

// UpdateSpinner updates the loading spinner animation
func (s *StatusLine) UpdateSpinner() *StatusLine {
	if s.isLoading {
		var cmd tea.Cmd
		s.loadingSpinner, cmd = s.loadingSpinner.Update(spinner.TickMsg{})
		_ = cmd // Ignore the command since we're just updating the view
	}
	return s
}

// UpdateSpinnerWithTick updates the loading spinner with the actual tick message
func (s *StatusLine) UpdateSpinnerWithTick(msg spinner.TickMsg) *StatusLine {
	if s.isLoading {
		var cmd tea.Cmd
		s.loadingSpinner, cmd = s.loadingSpinner.Update(msg)
		_ = cmd // Ignore the command since we're just updating the view
	}
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

	// Add loading spinner to right content if loading
	rightContent := s.rightContent
	if s.isLoading {
		if rightContent != "" {
			rightContent = s.loadingSpinner.View() + " " + rightContent
		} else {
			rightContent = s.loadingSpinner.View()
		}
	}

	// Check for active temporary message
	if s.HasActiveMessage() {
		// Keep the same layout structure but replace help text with temporary message
		leftContent := s.leftContent
		tempMessage := s.tempMessage

		// Truncate parts if needed
		maxPartWidth := s.width / 3
		if lipgloss.Width(leftContent) > maxPartWidth {
			leftContent = truncateWithEllipsis(leftContent, maxPartWidth)
		}
		if lipgloss.Width(rightContent) > maxPartWidth {
			rightContent = truncateWithEllipsis(rightContent, maxPartWidth)
		}

		leftLen := lipgloss.Width(leftContent)
		rightLen := lipgloss.Width(rightContent)

		// Calculate available space for the temporary message (in place of help text)
		availableForMessage := s.width - leftLen - rightLen - 4 // Account for padding
		if availableForMessage > 10 {
			// Truncate message to fit
			if lipgloss.Width(tempMessage) > availableForMessage {
				tempMessage = truncateWithEllipsis(tempMessage, availableForMessage)
			}
			messageLen := lipgloss.Width(tempMessage)
			middlePadding := strings.Repeat(" ", availableForMessage-messageLen)

			// Create colored message with the temporary message color
			coloredMessage := lipgloss.NewStyle().
				Foreground(s.tempMessageColor).
				Render(tempMessage)

			// Build status content maintaining the same layout
			statusContent := fmt.Sprintf("%s  %s%s  %s",
				leftContent,
				coloredMessage,
				middlePadding,
				rightContent)

			// Apply the base style (background)
			return lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Width(s.width).
				MaxWidth(s.width).
				MaxHeight(1).
				Render(statusContent)
		} else {
			// Not enough space, just show left, message (truncated), and right
			availableForMessage = s.width - leftLen - rightLen - 4
			if availableForMessage < 0 {
				availableForMessage = 0
			}
			if lipgloss.Width(tempMessage) > availableForMessage {
				tempMessage = truncateWithEllipsis(tempMessage, availableForMessage)
			}

			coloredMessage := lipgloss.NewStyle().
				Foreground(s.tempMessageColor).
				Render(tempMessage)

			padding := s.width - leftLen - lipgloss.Width(tempMessage) - rightLen
			if padding < 0 {
				padding = 0
			}
			statusContent := fmt.Sprintf("%s  %s%s%s",
				leftContent,
				coloredMessage,
				strings.Repeat(" ", padding),
				rightContent)

			return lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Width(s.width).
				MaxWidth(s.width).
				MaxHeight(1).
				Render(statusContent)
		}
	}

	// Truncate individual parts if they're too long
	maxPartWidth := s.width / 3
	leftContent := s.leftContent
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

// truncateWithEllipsis is now replaced by utils.TruncateWithEllipsis
// Keeping this as an alias for backward compatibility
func truncateWithEllipsis(s string, maxWidth int) string {
	return utils.TruncateWithEllipsis(s, maxWidth)
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
