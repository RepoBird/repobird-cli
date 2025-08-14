package interfaces

import "github.com/repobird/repobird-cli/internal/tui/components"

// LayoutAware represents views that use the global layout system
type LayoutAware interface {
	// GetLayout returns the view's layout instance
	GetLayout() *components.WindowLayout

	// UpdateLayout updates the view's layout with new dimensions
	UpdateLayout(width, height int)
}

// ViewWithConsistentSizing represents views that should have consistent borders and sizing
type ViewWithConsistentSizing interface {
	LayoutAware

	// ApplyStandardSizing applies consistent box, title, and content sizing
	ApplyStandardSizing()
}
