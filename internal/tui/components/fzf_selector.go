// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package components provides reusable UI components for the TUI
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// FZFMode represents the state of the FZF selector
type FZFMode struct {
	Active        bool
	Input         textinput.Model
	Items         []string
	FilteredItems []string
	SelectedIndex int
	MaxDisplay    int // Maximum items to display
	Width         int
	Height        int
}

// FZFResult represents the result of an FZF selection
type FZFResult struct {
	Selected string
	Index    int
	Canceled bool
}

// FZFSelectedMsg is sent when an item is selected in FZF mode
type FZFSelectedMsg struct {
	Result FZFResult
	Column int // Which column triggered the selection (for dashboard)
}

// NewFZFMode creates a new FZF mode selector
func NewFZFMode(items []string, width, height int) *FZFMode {
	input := textinput.New()
	input.Placeholder = "Type to filter..."
	input.Focus()
	input.CharLimit = 100
	input.Width = width - 4

	return &FZFMode{
		Active:        false,
		Input:         input,
		Items:         items,
		FilteredItems: items,
		SelectedIndex: 0,
		MaxDisplay:    10,
		Width:         width,
		Height:        height,
	}
}

// Activate enables FZF mode
func (f *FZFMode) Activate() {
	f.Active = true
	f.Input.Focus()
	f.Input.SetValue("")
	f.FilteredItems = f.Items
	f.SelectedIndex = 0
}

// Deactivate disables FZF mode
func (f *FZFMode) Deactivate() {
	f.Active = false
	f.Input.Blur()
	f.Input.SetValue("")
}

// SetItems updates the items list
func (f *FZFMode) SetItems(items []string) {
	f.Items = items
	f.FilteredItems = items
	f.SelectedIndex = 0
}

// Update handles FZF mode input
func (f *FZFMode) Update(msg tea.Msg) (*FZFMode, tea.Cmd) {
	if !f.Active {
		return f, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			f.Deactivate()
			return f, func() tea.Msg {
				return FZFSelectedMsg{
					Result: FZFResult{Canceled: true},
				}
			}

		case "enter":
			if len(f.FilteredItems) > 0 && f.SelectedIndex < len(f.FilteredItems) {
				selected := f.FilteredItems[f.SelectedIndex]
				f.Deactivate()
				return f, func() tea.Msg {
					return FZFSelectedMsg{
						Result: FZFResult{
							Selected: selected,
							Index:    f.findOriginalIndex(selected),
							Canceled: false,
						},
					}
				}
			}

		case "up", "ctrl+p", "ctrl+k":
			if f.SelectedIndex > 0 {
				f.SelectedIndex--
			}

		case "down", "ctrl+n", "ctrl+j":
			if f.SelectedIndex < len(f.FilteredItems)-1 {
				f.SelectedIndex++
			}

		default:
			// Update the input field
			var cmd tea.Cmd
			f.Input, cmd = f.Input.Update(msg)

			// Filter items based on input
			f.filterItems()

			return f, cmd
		}
	}

	return f, nil
}

// filterItems filters the items based on the current input
func (f *FZFMode) filterItems() {
	query := f.Input.Value()
	if query == "" {
		f.FilteredItems = f.Items
		f.SelectedIndex = 0
		return
	}

	// Use fuzzy matching
	matches := fuzzy.Find(query, f.Items)
	f.FilteredItems = make([]string, len(matches))
	for i, match := range matches {
		f.FilteredItems[i] = f.Items[match.Index]
	}

	// Reset selection if it's out of bounds
	if f.SelectedIndex >= len(f.FilteredItems) {
		f.SelectedIndex = 0
	}
}

// findOriginalIndex finds the original index of an item
func (f *FZFMode) findOriginalIndex(item string) int {
	for i, original := range f.Items {
		if original == item {
			return i
		}
	}
	return -1
}

// View renders the FZF selector as a compact dropdown
func (f *FZFMode) View() string {
	if !f.Active {
		return ""
	}

	var b strings.Builder

	// Style definitions for dropdown
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(min(f.Width-4, 50)) // Compact width

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("63")).
		Foreground(lipgloss.Color("15")).
		Width(min(f.Width-8, 46))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("235")).
		Width(min(f.Width-8, 46))

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Background(lipgloss.Color("235")).
		Bold(true)

	// Compact input field with search icon
	b.WriteString(inputStyle.Render("🔍 " + f.Input.View()))
	b.WriteString("\n")

	// Display filtered items (reduce max display for dropdown)
	displayCount := min(f.MaxDisplay, 8) // Max 8 items in dropdown
	if len(f.FilteredItems) < displayCount {
		displayCount = len(f.FilteredItems)
	}

	// Calculate scroll offset
	scrollOffset := 0
	if f.SelectedIndex >= displayCount {
		scrollOffset = f.SelectedIndex - displayCount + 1
	}

	// Show items
	for i := scrollOffset; i < scrollOffset+displayCount && i < len(f.FilteredItems); i++ {
		item := f.FilteredItems[i]
		// Truncate long items
		if len(item) > 44 {
			item = item[:41] + "..."
		}

		if i == f.SelectedIndex {
			b.WriteString(selectedStyle.Render("▶ " + item))
		} else {
			b.WriteString(normalStyle.Render("  " + item))
		}
		if i < scrollOffset+displayCount-1 && i < len(f.FilteredItems)-1 {
			b.WriteString("\n")
		}
	}

	// Compact scroll indicator
	if len(f.FilteredItems) > displayCount {
		scrollInfo := fmt.Sprintf(" [%d/%d]", f.SelectedIndex+1, len(f.FilteredItems))
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Background(lipgloss.Color("235")).
			Render(scrollInfo))
	}

	// Add hint at bottom
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Background(lipgloss.Color("235")).
		Italic(true)
	b.WriteString("\n")
	b.WriteString(hintStyle.Render("↑↓:navigate ↵:select esc:cancel"))

	return borderStyle.Render(b.String())
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// IsActive returns whether FZF mode is active
func (f *FZFMode) IsActive() bool {
	return f.Active
}

// GetSelected returns the currently selected item
func (f *FZFMode) GetSelected() (string, bool) {
	if len(f.FilteredItems) > 0 && f.SelectedIndex < len(f.FilteredItems) {
		return f.FilteredItems[f.SelectedIndex], true
	}
	return "", false
}
