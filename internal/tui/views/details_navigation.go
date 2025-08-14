package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// handleRowNavigation handles navigation between selectable rows/fields
func (v *RunDetailsView) handleRowNavigation(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if v.selectedRow < len(v.fieldValues)-1 {
			v.selectedRow++
			v.scrollToSelectedField()
		} else if len(v.fieldValues) > 0 {
			// Wrap around to the first item
			v.selectedRow = 0
			v.scrollToSelectedField()
		}
	case "k", "up":
		if v.selectedRow > 0 {
			v.selectedRow--
			v.scrollToSelectedField()
		} else if len(v.fieldValues) > 0 {
			// Wrap around to the last item
			v.selectedRow = len(v.fieldValues) - 1
			v.scrollToSelectedField()
		}
	case "g":
		// Go to first field
		v.selectedRow = 0
		v.scrollToSelectedField()
	case "G":
		// Go to last field
		if len(v.fieldValues) > 0 {
			v.selectedRow = len(v.fieldValues) - 1
			v.scrollToSelectedField()
		}
	}
	return nil
}

// scrollToSelectedField ensures the selected field is visible in the viewport
func (v *RunDetailsView) scrollToSelectedField() {
	if v.selectedRow >= 0 && v.selectedRow < len(v.fieldRanges) {
		// Get the range of the selected field
		fieldRange := v.fieldRanges[v.selectedRow]
		startLine := fieldRange[0]
		endLine := fieldRange[1]

		viewportTop := v.viewport.YOffset
		viewportBottom := viewportTop + v.viewport.Height - 1

		// If the entire field is above the viewport, scroll to show the start
		if endLine < viewportTop {
			v.viewport.SetYOffset(startLine)
		} else if startLine > viewportBottom {
			// If the entire field is below the viewport, scroll to show as much as possible
			// Try to show the whole field if it fits
			fieldHeight := endLine - startLine + 1
			if fieldHeight <= v.viewport.Height {
				// Field fits in viewport, position it at the top
				v.viewport.SetYOffset(startLine)
			} else {
				// Field is larger than viewport, show the beginning
				v.viewport.SetYOffset(startLine)
			}
		}
		// If part of the field is visible, don't scroll
	}
}

// handleViewportNavigation handles viewport scrolling keys
func (v *RunDetailsView) handleViewportNavigation(msg tea.KeyMsg) {
	switch {
	case key.Matches(msg, v.keys.Up):
		v.viewport.ScrollUp(1)
	case key.Matches(msg, v.keys.Down):
		v.viewport.ScrollDown(1)
	case key.Matches(msg, v.keys.PageUp):
		v.viewport.HalfPageUp()
	case key.Matches(msg, v.keys.PageDown):
		v.viewport.HalfPageDown()
	case key.Matches(msg, v.keys.Home):
		v.viewport.GotoTop()
	case key.Matches(msg, v.keys.End):
		v.viewport.GotoBottom()
	}
}
