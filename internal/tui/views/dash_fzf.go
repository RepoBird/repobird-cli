// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

// activateFZFMode activates inline FZF mode for the current column
// This function is replaced by startFZFMode() in dashboard.go
// Keeping it for compatibility but it now calls the new implementation
func (d *DashboardView) activateFZFMode() {
	d.startFZFMode()
}

// renderWithFZFOverlay renders the dashboard with FZF dropdown overlay
// This function is no longer used - inline FZF is rendered within columns
func (d *DashboardView) renderWithFZFOverlay(baseView string) string {
	// Inline FZF is handled directly in column rendering now
	return baseView
}
