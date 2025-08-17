// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/stretchr/testify/assert"
)

func TestNewErrorView(t *testing.T) {
	t.Run("Recoverable error", func(t *testing.T) {
		err := errors.New("test error")
		view := NewErrorView(err, "Something went wrong", true)

		assert.NotNil(t, view)
		assert.Equal(t, err, view.err)
		assert.Equal(t, "Something went wrong", view.message)
		assert.True(t, view.recoverable)
		assert.NotNil(t, view.keymaps)
	})

	t.Run("Non-recoverable error", func(t *testing.T) {
		err := errors.New("fatal error")
		view := NewErrorView(err, "Fatal error occurred", false)

		assert.NotNil(t, view)
		assert.Equal(t, err, view.err)
		assert.Equal(t, "Fatal error occurred", view.message)
		assert.False(t, view.recoverable)
	})

	t.Run("Error with nil error object", func(t *testing.T) {
		view := NewErrorView(nil, "An error occurred", true)

		assert.NotNil(t, view)
		assert.Nil(t, view.err)
		assert.Equal(t, "An error occurred", view.message)
	})
}

func TestErrorViewInit(t *testing.T) {
	view := NewErrorView(errors.New("test"), "Test error", true)
	cmd := view.Init()

	assert.Nil(t, cmd) // Init should return nil
}

func TestErrorViewUpdate(t *testing.T) {
	t.Run("Window resize", func(t *testing.T) {
		view := NewErrorView(errors.New("test"), "Test error", true)

		msg := tea.WindowSizeMsg{
			Width:  100,
			Height: 30,
		}

		model, cmd := view.Update(msg)
		updatedView := model.(*ErrorView)

		assert.Equal(t, 100, updatedView.width)
		assert.Equal(t, 30, updatedView.height)
		assert.Nil(t, cmd)
	})

	t.Run("Recoverable - Enter key navigates back", func(t *testing.T) {
		view := NewErrorView(errors.New("test"), "Test error", true)
		view.width = 80
		view.height = 24

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		model, cmd := view.Update(msg)

		assert.Equal(t, view, model)
		assert.NotNil(t, cmd)

		// Execute the command to get the navigation message
		navMsg := cmd()
		_, ok := navMsg.(messages.NavigateBackMsg)
		assert.True(t, ok, "Should return NavigateBackMsg")
	})

	t.Run("Recoverable - ESC key navigates back", func(t *testing.T) {
		view := NewErrorView(errors.New("test"), "Test error", true)
		view.width = 80
		view.height = 24

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		model, cmd := view.Update(msg)

		assert.Equal(t, view, model)
		assert.NotNil(t, cmd)

		// Execute the command to get the navigation message
		navMsg := cmd()
		_, ok := navMsg.(messages.NavigateBackMsg)
		assert.True(t, ok, "Should return NavigateBackMsg")
	})

	t.Run("Non-recoverable - Enter key navigates to dashboard", func(t *testing.T) {
		view := NewErrorView(errors.New("fatal"), "Fatal error", false)
		view.width = 80
		view.height = 24

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		model, cmd := view.Update(msg)

		assert.Equal(t, view, model)
		assert.NotNil(t, cmd)

		// Execute the command to get the navigation message
		navMsg := cmd()
		_, ok := navMsg.(messages.NavigateToDashboardMsg)
		assert.True(t, ok, "Should return NavigateToDashboardMsg")
	})

	t.Run("Non-recoverable - ESC key navigates to dashboard", func(t *testing.T) {
		view := NewErrorView(errors.New("fatal"), "Fatal error", false)
		view.width = 80
		view.height = 24

		msg := tea.KeyMsg{Type: tea.KeyEsc}
		model, cmd := view.Update(msg)

		assert.Equal(t, view, model)
		assert.NotNil(t, cmd)

		// Execute the command to get the navigation message
		navMsg := cmd()
		_, ok := navMsg.(messages.NavigateToDashboardMsg)
		assert.True(t, ok, "Should return NavigateToDashboardMsg")
	})

	t.Run("Quit commands", func(t *testing.T) {
		view := NewErrorView(errors.New("test"), "Test error", true)
		view.width = 80
		view.height = 24

		tests := []tea.KeyMsg{
			{Type: tea.KeyRunes, Runes: []rune{'q'}},
			{Type: tea.KeyCtrlC},
		}

		for _, msg := range tests {
			model, cmd := view.Update(msg)
			assert.Equal(t, view, model)
			assert.NotNil(t, cmd)
			// Should return quit command
		}
	})

	t.Run("Other keys do nothing", func(t *testing.T) {
		view := NewErrorView(errors.New("test"), "Test error", true)
		view.width = 80
		view.height = 24

		// Random key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
		model, cmd := view.Update(msg)

		assert.Equal(t, view, model)
		assert.Nil(t, cmd)
	})
}

func TestErrorViewRendering(t *testing.T) {
	t.Run("Before window size is set", func(t *testing.T) {
		view := NewErrorView(errors.New("test"), "Test error", true)

		// Should return empty string when dimensions are not set
		output := view.View()
		assert.Equal(t, "", output)
	})

	t.Run("Recoverable error rendering", func(t *testing.T) {
		view := NewErrorView(errors.New("test error"), "Something went wrong", true)
		// Send window size message to initialize layout
		updatedView, _ := view.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		view = updatedView.(*ErrorView)

		output := view.View()

		assert.Contains(t, output, "⚠ Error")
		assert.Contains(t, output, "Something went wrong")
		assert.Contains(t, output, "Details: test error")
		assert.Contains(t, output, "Press Enter or ESC to go back")
	})

	t.Run("Non-recoverable error rendering", func(t *testing.T) {
		view := NewErrorView(errors.New("fatal error"), "Fatal error occurred", false)
		// Send window size message to initialize layout
		updatedView, _ := view.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		view = updatedView.(*ErrorView)

		output := view.View()

		assert.Contains(t, output, "⚠ Error")
		assert.Contains(t, output, "Fatal error occurred")
		assert.Contains(t, output, "Details: fatal error")
		assert.Contains(t, output, "Press Enter to return to dashboard")
		assert.NotContains(t, output, "go back") // Should not mention going back
	})

	t.Run("Error without error object", func(t *testing.T) {
		view := NewErrorView(nil, "An error occurred", true)
		// Send window size message to initialize layout
		updatedView, _ := view.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		view = updatedView.(*ErrorView)

		output := view.View()

		assert.Contains(t, output, "⚠ Error")
		assert.Contains(t, output, "An error occurred")
		assert.NotContains(t, output, "Details:") // No details when err is nil
	})

	t.Run("Long error message", func(t *testing.T) {
		longMessage := "This is a very long error message that might wrap or be truncated depending on the terminal width and the rendering logic of the error view component"
		view := NewErrorView(errors.New("error"), longMessage, true)
		// Send window size message to initialize layout
		updatedView, _ := view.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		view = updatedView.(*ErrorView)

		output := view.View()

		// At least part of the message should be visible
		assert.Contains(t, output, "This is a very long error message")
	})
}

func TestErrorViewStyling(t *testing.T) {
	view := NewErrorView(errors.New("test"), "Test error", true)
	// Send window size message to initialize layout
	updatedView, _ := view.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view = updatedView.(*ErrorView)

	output := view.View()

	// Check that styling is applied (output will contain ANSI codes)
	assert.NotEqual(t, "Error\nTest error\nDetails: test\nPress Enter or ESC to go back", output)
	// The output should be styled, so it will be longer than plain text
	assert.Greater(t, len(output), len("Error Test error Details: test Press Enter or ESC to go back"))
}

func TestErrorViewDimensions(t *testing.T) {
	view := NewErrorView(errors.New("test"), "Test error", true)

	tests := []struct {
		width  int
		height int
	}{
		{80, 24},
		{120, 40},
		{60, 20},
		{200, 50},
	}

	for _, tt := range tests {
		msg := tea.WindowSizeMsg{
			Width:  tt.width,
			Height: tt.height,
		}

		model, _ := view.Update(msg)
		updatedView := model.(*ErrorView)

		assert.Equal(t, tt.width, updatedView.width)
		assert.Equal(t, tt.height, updatedView.height)

		// View should render with these dimensions
		output := updatedView.View()
		assert.NotEmpty(t, output)
	}
}

func TestErrorViewModelInterface(t *testing.T) {
	view := NewErrorView(errors.New("test"), "Test error", true)

	// Verify it implements tea.Model interface
	var _ tea.Model = view

	// Test all required methods
	assert.NotPanics(t, func() {
		_ = view.Init()
		_, _ = view.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		_ = view.View()
	})
}
