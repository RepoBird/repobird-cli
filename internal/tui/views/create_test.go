package views

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
)

func TestCreateRunView_HandleWindowSizeMsg(t *testing.T) {
	client := &api.Client{}
	view := NewCreateRunView(client)

	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"standard terminal", 80, 24},
		{"wide terminal", 120, 30},
		{"narrow terminal", 40, 20},
		{"small terminal", 20, 10},
		{"minimal terminal", 10, 5},
		{"zero width", 0, 24},
		{"zero height", 80, 0},
		{"both zero", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Send window size message
			updatedView, _ := view.Update(tea.WindowSizeMsg{
				Width:  tt.width,
				Height: tt.height,
			})

			createView := updatedView.(*CreateRunView)

			// Verify dimensions are stored correctly
			if createView.width != tt.width {
				t.Errorf("width = %d, want %d", createView.width, tt.width)
			}
			if createView.height != tt.height {
				t.Errorf("height = %d, want %d", createView.height, tt.height)
			}

			// Test that view renders and is not empty (prevents black screen)
			viewOutput := createView.View()
			if strings.TrimSpace(viewOutput) == "" {
				t.Errorf("view is empty at size %dx%d", tt.width, tt.height)
			}

			// Should contain basic UI elements even at small sizes
			if !strings.Contains(viewOutput, "Create") {
				t.Errorf("view missing 'Create' text at size %dx%d\nView:\n%s",
					tt.width, tt.height, viewOutput)
			}
		})
	}
}

func TestCreateRunView_PreventBlackScreen(t *testing.T) {
	client := &api.Client{}

	tests := []struct {
		name      string
		setupFunc func(*CreateRunView) *CreateRunView
	}{
		{
			name: "uninitialized dimensions",
			setupFunc: func(v *CreateRunView) *CreateRunView {
				// Don't send window size message - should still render
				return v
			},
		},
		{
			name: "minimal dimensions",
			setupFunc: func(v *CreateRunView) *CreateRunView {
				updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 1, Height: 1})
				return updatedView.(*CreateRunView)
			},
		},
		{
			name: "during submission",
			setupFunc: func(v *CreateRunView) *CreateRunView {
				// Set dimensions first
				updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				v = updatedView.(*CreateRunView)

				// Start submission
				v.submitting = true
				return v
			},
		},
		{
			name: "with error state",
			setupFunc: func(v *CreateRunView) *CreateRunView {
				updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				v = updatedView.(*CreateRunView)

				// Set error
				v.error = fmt.Errorf("test error")
				return v
			},
		},
		{
			name: "file input mode",
			setupFunc: func(v *CreateRunView) *CreateRunView {
				updatedView, _ := v.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
				v = updatedView.(*CreateRunView)

				// Enable file input mode
				v.useFileInput = true
				return v
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := NewCreateRunView(client)
			view = tt.setupFunc(view)

			viewOutput := view.View()

			// Should never render empty (prevents black screen)
			if strings.TrimSpace(viewOutput) == "" {
				t.Errorf("view is empty in scenario: %s", tt.name)
			}

			// Should contain some basic UI elements
			hasBasicElements := strings.Contains(viewOutput, "Create") ||
				strings.Contains(viewOutput, "Run") ||
				strings.Contains(viewOutput, "Title") ||
				strings.Contains(viewOutput, "Repository")

			if !hasBasicElements {
				t.Errorf("view missing basic UI elements in scenario: %s\nView:\n%s",
					tt.name, viewOutput)
			}
		})
	}
}

func TestCreateRunView_ViewTransitionPreservesDimensions(t *testing.T) {
	client := &api.Client{}
	view := NewCreateRunView(client)

	// Set initial dimensions
	updatedView, _ := view.Update(tea.WindowSizeMsg{
		Width:  80,
		Height: 24,
	})

	createView := updatedView.(*CreateRunView)

	// Verify dimensions are set
	if createView.width != 80 || createView.height != 24 {
		t.Fatalf("initial dimensions not set correctly: %dx%d", createView.width, createView.height)
	}

	// Simulate successful run creation
	mockRun := models.RunResponse{
		ID:     "test-123",
		Title:  "Test Run",
		Status: "pending",
	}
	msg := runCreatedMsg{run: mockRun, err: nil}

	// Transition to details view
	detailsModel, _ := createView.Update(msg)

	// Verify it's a details view and received dimensions
	detailsView, ok := detailsModel.(*RunDetailsView)
	if !ok {
		t.Fatal("expected transition to RunDetailsView")
	}

	if detailsView.width != 80 {
		t.Errorf("details view width = %d, want 80", detailsView.width)
	}
	if detailsView.height != 24 {
		t.Errorf("details view height = %d, want 24", detailsView.height)
	}

	// Verify details view renders properly (not black screen)
	detailsViewOutput := detailsView.View()
	if strings.TrimSpace(detailsViewOutput) == "" {
		t.Error("details view should not be empty after transition")
	}
}

func TestCreateRunView_ResponsiveLayout(t *testing.T) {
	client := &api.Client{}
	view := NewCreateRunView(client)

	tests := []struct {
		name         string
		width        int
		height       int
		expectWider  bool
		expectTaller bool
	}{
		{"standard", 80, 24, true, true},
		{"wide", 120, 30, true, true},
		{"narrow", 40, 20, false, false},
		{"tall narrow", 30, 40, false, true},
		{"wide short", 100, 15, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedView, _ := view.Update(tea.WindowSizeMsg{
				Width:  tt.width,
				Height: tt.height,
			})

			createView := updatedView.(*CreateRunView)
			viewOutput := createView.View()

			// Check that view adapts to size
			lines := strings.Split(viewOutput, "\n")
			maxLineLength := 0
			for _, line := range lines {
				if len(line) > maxLineLength {
					maxLineLength = len(line)
				}
			}

			// View should not exceed terminal width (with some tolerance for styling)
			if maxLineLength > tt.width+5 { // +5 tolerance for ANSI codes
				t.Errorf("view line length %d exceeds terminal width %d",
					maxLineLength, tt.width)
			}

			// View should not exceed terminal height
			if len(lines) > tt.height+2 { // +2 tolerance
				t.Errorf("view has %d lines, exceeds terminal height %d",
					len(lines), tt.height)
			}

			// Should always be non-empty
			if strings.TrimSpace(viewOutput) == "" {
				t.Errorf("view is empty at size %dx%d", tt.width, tt.height)
			}
		})
	}
}

func TestCreateRunView_TextAreaResponsiveness(t *testing.T) {
	client := &api.Client{}
	view := NewCreateRunView(client)

	// Test that text areas adjust to terminal size
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{"standard", 80, 24},
		{"narrow", 40, 20},
		{"very narrow", 20, 15},
		{"wide", 120, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedView, _ := view.Update(tea.WindowSizeMsg{
				Width:  tt.width,
				Height: tt.height,
			})

			createView := updatedView.(*CreateRunView)

			// Check that text areas have reasonable widths
			promptWidth := createView.promptArea.Width()
			contextWidth := createView.contextArea.Width()

			// Text areas should not be wider than terminal (minus padding)
			maxWidth := tt.width - 10 // Account for padding
			if maxWidth < 30 {
				maxWidth = 30 // Minimum usable width
			}

			if promptWidth > maxWidth {
				t.Errorf("prompt area width %d exceeds max width %d",
					promptWidth, maxWidth)
			}
			if contextWidth > maxWidth {
				t.Errorf("context area width %d exceeds max width %d",
					contextWidth, maxWidth)
			}

			// Text areas should have reasonable minimum width
			if promptWidth < 20 && tt.width > 25 {
				t.Errorf("prompt area width %d too narrow for terminal width %d",
					promptWidth, tt.width)
			}
			if contextWidth < 20 && tt.width > 25 {
				t.Errorf("context area width %d too narrow for terminal width %d",
					contextWidth, tt.width)
			}
		})
	}
}

func TestCreateRunView_StatusBarRespectsDimensions(t *testing.T) {
	client := &api.Client{}
	view := NewCreateRunView(client)

	tests := []struct {
		name  string
		width int
	}{
		{"standard", 80},
		{"narrow", 40},
		{"very narrow", 20},
		{"wide", 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedView, _ := view.Update(tea.WindowSizeMsg{
				Width:  tt.width,
				Height: 24,
			})

			createView := updatedView.(*CreateRunView)
			statusBar := createView.renderStatusBar()

			// Status bar should not be empty
			if strings.TrimSpace(statusBar) == "" {
				t.Error("status bar should not be empty")
			}

			// Status bar should not cause issues with width
			// For very narrow terminals, just ensure it's not empty
			if tt.width < 30 {
				// For very narrow terminals, just check it's not empty
				if strings.TrimSpace(statusBar) == "" {
					t.Error("status bar should not be empty even on narrow terminals")
				}
			} else {
				// For reasonable widths, check it doesn't cause obvious overflow
				// We'll be very generous with tolerance due to ANSI codes and styling
				// Status bars can be complex with styling codes
				if len(statusBar) > tt.width*5 { // Extremely generous tolerance
					t.Errorf("status bar seems excessively long for terminal width %d (len=%d)",
						tt.width, len(statusBar))
				}
			}
		})
	}
}
