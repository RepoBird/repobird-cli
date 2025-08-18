// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// InlineFZF represents an inline FZF search that doesn't create overlay windows
type InlineFZF struct {
	Active        bool
	Input         textinput.Model
	Items         []string       // Original items
	FilteredItems []string       // Filtered items
	SelectedIndex int            // Selected index in filtered items
	Query         string         // Current search query
	Placeholder   string         // Placeholder text
	Width         int            // Width for rendering
	
	// Store the last selection before deactivating
	LastSelected      string
	LastSelectedIndex int
}

// NewInlineFZF creates a new inline FZF component
func NewInlineFZF(items []string, placeholder string, width int) *InlineFZF {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = 100
	input.Width = width - 4
	
	return &InlineFZF{
		Active:        false,
		Input:         input,
		Items:         items,
		FilteredItems: items,
		SelectedIndex: 0,
		Query:         "",
		Placeholder:   placeholder,
		Width:         width,
	}
}

// Activate enables inline FZF mode
func (f *InlineFZF) Activate() {
	f.Active = true
	f.Input.Focus()
	f.Input.SetValue("")
	f.Query = ""
	f.FilteredItems = f.Items
	f.SelectedIndex = 0
}

// Deactivate disables inline FZF mode
func (f *InlineFZF) Deactivate() {
	f.Active = false
	f.Input.Blur()
	f.Input.SetValue("")
	f.Query = ""
	f.FilteredItems = f.Items
}

// IsActive returns whether FZF mode is active
func (f *InlineFZF) IsActive() bool {
	return f.Active
}

// SetItems updates the items to search through
func (f *InlineFZF) SetItems(items []string) {
	f.Items = items
	f.filterItems()
}

// Update handles messages for the inline FZF
func (f *InlineFZF) Update(msg tea.Msg) (*InlineFZF, tea.Cmd) {
	if !f.Active {
		return f, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			f.Deactivate()
			return f, nil
			
		case "enter":
			// Save selection before deactivating
			if f.SelectedIndex >= 0 && f.SelectedIndex < len(f.FilteredItems) {
				f.LastSelected = f.FilteredItems[f.SelectedIndex]
				// Find original index
				for i, item := range f.Items {
					if item == f.LastSelected {
						f.LastSelectedIndex = i
						break
					}
				}
			}
			// Now deactivate
			f.Deactivate()
			return f, nil
			
		case "up", "ctrl+p", "ctrl+k":
			if f.SelectedIndex > 0 {
				f.SelectedIndex--
			}
			return f, nil
			
		case "down", "ctrl+n", "ctrl+j":
			if f.SelectedIndex < len(f.FilteredItems)-1 {
				f.SelectedIndex++
			}
			return f, nil
			
		default:
			// Update the input field
			var cmd tea.Cmd
			f.Input, cmd = f.Input.Update(msg)
			f.Query = f.Input.Value()
			f.filterItems()
			return f, cmd
		}
	}
	
	return f, nil
}

// filterItems filters items based on current query
func (f *InlineFZF) filterItems() {
	if f.Query == "" {
		f.FilteredItems = f.Items
		return
	}
	
	// Use fuzzy matching
	matches := fuzzy.Find(f.Query, f.Items)
	f.FilteredItems = make([]string, len(matches))
	for i, match := range matches {
		f.FilteredItems[i] = f.Items[match.Index]
	}
	
	// Reset selection if out of bounds
	if f.SelectedIndex >= len(f.FilteredItems) {
		f.SelectedIndex = 0
	}
}

// GetSelected returns the currently selected item
func (f *InlineFZF) GetSelected() (string, int) {
	if f.SelectedIndex < 0 || f.SelectedIndex >= len(f.FilteredItems) {
		return "", -1
	}
	
	selected := f.FilteredItems[f.SelectedIndex]
	// Find original index
	for i, item := range f.Items {
		if item == selected {
			return selected, i
		}
	}
	return selected, -1
}

// GetLastSelection returns the last selected item before deactivation
func (f *InlineFZF) GetLastSelection() (string, int) {
	return f.LastSelected, f.LastSelectedIndex
}

// RenderSearchBar renders just the search input bar
func (f *InlineFZF) RenderSearchBar() string {
	if !f.Active {
		return ""
	}
	
	searchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)
	
	return searchStyle.Render("🔍 " + f.Input.View())
}

// GetFilteredItems returns the current filtered items
func (f *InlineFZF) GetFilteredItems() []string {
	return f.FilteredItems
}

// GetSelectedIndex returns the current selected index in filtered items
func (f *InlineFZF) GetSelectedIndex() int {
	return f.SelectedIndex
}

// HighlightMatch highlights fuzzy match in item text
func HighlightFZFMatch(text, query string) string {
	if query == "" {
		return text
	}
	
	// Simple case-insensitive highlighting
	lower := strings.ToLower(text)
	queryLower := strings.ToLower(query)
	
	if idx := strings.Index(lower, queryLower); idx >= 0 {
		before := text[:idx]
		match := text[idx : idx+len(query)]
		after := text[idx+len(query):]
		
		highlightStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)
		
		return before + highlightStyle.Render(match) + after
	}
	
	return text
}