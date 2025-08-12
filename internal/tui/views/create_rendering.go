package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/tui/components"
)

// UI Rendering and layout management

func (m *CreateRunView) View() string {
	if m.width <= 0 || m.height <= 0 {
		return "Initializing..."
	}

	// Calculate available space for content
	statusBarHeight := 2
	availableHeight := m.height - statusBarHeight

	var baseView string
	
	// Render based on current mode
	if m.inputMode == components.ErrorMode {
		baseView = m.renderErrorLayout(availableHeight)
	} else {
		baseView = m.renderSinglePanelLayout(availableHeight)
	}

	// Add FZF overlay if active
	if m.fzfActive && m.fzfMode != nil {
		baseView = m.renderWithFZFOverlay(baseView)
	}

	// Add file selection modal if active
	if m.configFileSelectorActive {
		statusBar := m.renderStatusBar()
		return m.renderFileSelectionModal(statusBar)
	}

	// Combine with status bar
	statusBar := m.renderStatusBar()
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		baseView,
		statusBar,
	)
}

func (m *CreateRunView) renderSinglePanelLayout(availableHeight int) string {
	// Use compact form for smaller screens
	if m.width < 120 || availableHeight < 25 {
		return m.renderCompactForm(m.width, availableHeight)
	}

	// TODO: Implement full single panel layout
	return m.renderCompactForm(m.width, availableHeight)
}

func (m *CreateRunView) renderCompactForm(width, height int) string {
	// TODO: Implement compact form rendering
	return "Compact form layout"
}

func (m *CreateRunView) renderErrorLayout(availableHeight int) string {
	// TODO: Implement error layout rendering
	return "Error layout"
}

func (m *CreateRunView) renderFileSelectionModal(statusBar string) string {
	// TODO: Implement file selection modal rendering
	return "File selection modal"
}

func (m *CreateRunView) renderWithFZFOverlay(baseView string) string {
	if m.fzfMode == nil {
		return baseView
	}

	fzfView := m.fzfMode.View()
	
	// Calculate position for FZF overlay
	yOffset := 5  // Position below form fields
	xOffset := 2  // Small left margin
	
	return m.renderOverlayDropdown(baseView, fzfView, yOffset, xOffset)
}

func (m *CreateRunView) renderOverlayDropdown(baseView, overlayView string, yOffset, xOffset int) string {
	// Split base view into lines
	baseLines := strings.Split(baseView, "\n")
	overlayLines := strings.Split(overlayView, "\n")
	
	// Ensure we have enough base lines
	for len(baseLines) < yOffset+len(overlayLines) {
		baseLines = append(baseLines, "")
	}
	
	// Overlay the dropdown
	for i, overlayLine := range overlayLines {
		lineIndex := yOffset + i
		if lineIndex < len(baseLines) {
			// Insert overlay at the specified offset
			baseLine := baseLines[lineIndex]
			if len(baseLine) >= xOffset {
				// Replace part of the base line with overlay
				prefix := baseLine[:xOffset]
				suffix := ""
				if len(baseLine) > xOffset+len(overlayLine) {
					suffix = baseLine[xOffset+len(overlayLine):]
				}
				baseLines[lineIndex] = prefix + overlayLine + suffix
			} else {
				// Pad the base line and add overlay
				padding := strings.Repeat(" ", xOffset-len(baseLine))
				baseLines[lineIndex] = baseLine + padding + overlayLine
			}
		}
	}
	
	return strings.Join(baseLines, "\n")
}

func (m *CreateRunView) renderFieldIndicator() string {
	if m.inputMode == components.InsertMode {
		return "🟢 INSERT"
	} else if m.inputMode == components.NormalMode {
		return "🔵 NORMAL"
	}
	return "🔴 ERROR"
}

func (m *CreateRunView) renderFileInputMode() string {
	if m.useFileInput {
		return "📁 FILE"
	}
	return "✏️ FORM"
}

func (m *CreateRunView) renderStatusBar() string {
	leftSide := fmt.Sprintf("%s | %s", m.renderFieldIndicator(), m.renderFileInputMode())
	
	rightSide := ""
	if m.yankBlink {
		rightSide = "📋 Copied!"
	}
	
	// Create status bar with proper styling
	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("255")).
		Padding(0, 1)
	
	leftStyled := statusStyle.Render(leftSide)
	rightStyled := statusStyle.Render(rightSide)
	
	// Calculate spacing
	totalUsed := lipgloss.Width(leftStyled) + lipgloss.Width(rightStyled)
	spacing := m.width - totalUsed
	if spacing < 0 {
		spacing = 0
	}
	
	return leftStyled + strings.Repeat(" ", spacing) + rightStyled
}