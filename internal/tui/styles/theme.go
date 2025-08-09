package styles

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	BaseStyle = lipgloss.NewStyle()

	TitleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("235"))

	SelectedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("63"))

	HelpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	ErrorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	SuccessStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)

	WarningStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Bold(true)

	ProcessingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true)

	QueuedStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("226"))

	BorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	TableHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	TableRowStyle = lipgloss.NewStyle().
		Padding(0, 1)

	TableSelectedRowStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)
)

func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "DONE", "COMPLETED":
		return SuccessStyle
	case "FAILED", "ERROR":
		return ErrorStyle
	case "PROCESSING", "INITIALIZING", "POST_PROCESS":
		return ProcessingStyle
	case "QUEUED", "PENDING":
		return QueuedStyle
	default:
		return BaseStyle
	}
}

func GetStatusIcon(status string) string {
	switch status {
	case "DONE", "COMPLETED":
		return "✓"
	case "FAILED", "ERROR":
		return "✗"
	case "PROCESSING", "INITIALIZING", "POST_PROCESS":
		return "⟳"
	case "QUEUED", "PENDING":
		return "⏳"
	default:
		return "•"
	}
}