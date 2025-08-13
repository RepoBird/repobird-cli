package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/debug"
)

// WindowLayout provides consistent window sizing and border calculations for all views
type WindowLayout struct {
	terminalWidth  int
	terminalHeight int
	
	// Calculated dimensions
	contentWidth   int
	contentHeight  int
	boxWidth       int
	boxHeight      int
	
	// Layout constants
	statusLineHeight int
	borderMargin     int
	topMargin        int
}

// NewWindowLayout creates a new window layout calculator
func NewWindowLayout(terminalWidth, terminalHeight int) *WindowLayout {
	layout := &WindowLayout{
		terminalWidth:    terminalWidth,
		terminalHeight:   terminalHeight,
		statusLineHeight: 1,
		borderMargin:     2, // Lipgloss boxes render 2 pixels wider than set width
		topMargin:        2, // Additional top margin for border visibility (increased)
	}
	
	layout.calculateDimensions()
	return layout
}

// calculateDimensions computes all the derived dimensions
func (w *WindowLayout) calculateDimensions() {
	// Box dimensions account for lipgloss border expansion
	w.boxWidth = w.terminalWidth - w.borderMargin
	w.boxHeight = w.terminalHeight - w.statusLineHeight - w.topMargin // Removed extra -1 to use more vertical space
	
	// Content dimensions are inside the box (account for borders + padding)
	w.contentWidth = w.boxWidth - 4  // Border (2) + padding (2)
	w.contentHeight = w.boxHeight - 3 // Title (1) + borders/padding (2)
	
	// Minimum dimensions
	if w.boxWidth < 10 {
		w.boxWidth = 10
	}
	if w.boxHeight < 3 {
		w.boxHeight = 3
	}
	if w.contentWidth < 5 {
		w.contentWidth = 5
	}
	if w.contentHeight < 1 {
		w.contentHeight = 1
	}
	
	// Debug logging
	debug.LogToFilef("🏗️ LAYOUT: Terminal %dx%d → Box %dx%d → Content %dx%d 🏗️\n",
		w.terminalWidth, w.terminalHeight,
		w.boxWidth, w.boxHeight,
		w.contentWidth, w.contentHeight)
}

// GetBoxDimensions returns the box width and height for lipgloss container
func (w *WindowLayout) GetBoxDimensions() (width, height int) {
	return w.boxWidth, w.boxHeight
}

// GetContentDimensions returns the content area dimensions (inside the box)
func (w *WindowLayout) GetContentDimensions() (width, height int) {
	return w.contentWidth, w.contentHeight
}

// GetViewportDimensions returns dimensions suitable for bubble tea viewports
func (w *WindowLayout) GetViewportDimensions() (width, height int) {
	// Viewport gets the content area dimensions
	return w.contentWidth, w.contentHeight
}

// CreateStandardBox creates a lipgloss box style with standard dimensions and border
func (w *WindowLayout) CreateStandardBox() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(w.boxWidth).
		Height(w.boxHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63"))
}

// CreateTitleStyle creates a standard title style for boxes
func (w *WindowLayout) CreateTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Width(w.boxWidth - 2). // Account for border
		Align(lipgloss.Center).
		Padding(0, 1)
}

// CreateContentStyle creates a standard content area style
func (w *WindowLayout) CreateContentStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(w.boxWidth - 2). // Account for border
		Height(w.contentHeight).
		Padding(0, 1)
}

// IsValidDimensions checks if the terminal is large enough for the UI
func (w *WindowLayout) IsValidDimensions() bool {
	return w.terminalWidth >= 20 && w.terminalHeight >= 5
}

// GetMinimalView returns a minimal view for very small terminals
func (w *WindowLayout) GetMinimalView(message string) string {
	if w.terminalWidth <= 2 {
		return "" // Terminal too small to display anything
	}
	if len(message) > w.terminalWidth-2 {
		message = message[:w.terminalWidth-2]
	}
	return message
}

// Update recalculates dimensions when terminal size changes
func (w *WindowLayout) Update(terminalWidth, terminalHeight int) {
	w.terminalWidth = terminalWidth
	w.terminalHeight = terminalHeight
	w.calculateDimensions()
}

// LayoutType represents different layout configurations
type LayoutType int

const (
	LayoutStandard LayoutType = iota // Standard single box with statusline
	LayoutDashboard                  // Multi-column dashboard layout
	LayoutSplit                      // Split pane layout
)

// GetLayoutForType returns a configured layout for specific view types
func (w *WindowLayout) GetLayoutForType(layoutType LayoutType) *WindowLayout {
	// For now, return the same layout, but this can be extended
	// for different layout types (dashboard columns, split views, etc.)
	switch layoutType {
	case LayoutDashboard:
		// Dashboard might need different calculations for multi-column
		return w
	case LayoutSplit:
		// Split views might need different height calculations
		return w
	default:
		return w
	}
}