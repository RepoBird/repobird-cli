package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// This file contains only the help overlay functionality for the dashboard
// The status info overlay has been removed

// handleHelpNavigation handles keyboard navigation in the help overlay
func (d *DashboardView) handleHelpNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle special keys for closing help
	switch msg.String() {
	case "?", "q", "b", "escape":
		// Close the help overlay
		d.showDocs = false
		return d, nil
	case "Q":
		// Force quit
		_ = d.cache.SaveToDisk()
		d.cache.Stop()
		return d, tea.Quit
	}

	// Pass other keys to the help view
	updatedHelp, helpCmd := d.helpView.Update(msg)
	d.helpView = updatedHelp
	return d, helpCmd
}

// renderHelp renders the help overlay using the scrollable help view
func (d *DashboardView) renderHelp() string {
	// Set the size for the help view
	d.helpView.SetSize(d.width, d.height)
	// Return the rendered help view
	return d.helpView.View()
}
