// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpView(t *testing.T) {
	// Create a new help view
	helpView := NewHelpView()

	// Set a reasonable size
	helpView.SetSize(80, 24)

	// Verify the view renders without error
	output := helpView.View()

	// Check that the output contains expected elements
	if !strings.Contains(output, "RepoBird CLI Help Documentation") {
		t.Error("Help view should contain title")
	}

	// Test scrolling commands
	tests := []struct {
		name string
		key  string
	}{
		{"scroll down", "j"},
		{"scroll up", "k"},
		{"half page down", "ctrl+d"},
		{"half page up", "ctrl+u"},
		{"go to top", "g"},
		{"go to bottom", "G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create key message
			msg := tea.KeyMsg{
				Type:  tea.KeyRunes,
				Runes: []rune(tt.key),
			}

			// Update the view
			updatedView, _ := helpView.Update(msg)
			if updatedView == nil {
				t.Errorf("Update returned nil for key %s", tt.key)
			}
		})
	}

	// Test that sections are properly organized
	sections := getDefaultHelpSections()
	if len(sections) == 0 {
		t.Error("Help sections should not be empty")
	}

	// Verify each section has a title and content
	for i, section := range sections {
		if section.Title == "" {
			t.Errorf("Section %d has no title", i)
		}
		if len(section.Content) == 0 {
			t.Errorf("Section %s has no content", section.Title)
		}
	}
}

func TestHelpViewContent(t *testing.T) {
	sections := getDefaultHelpSections()

	// Check for essential sections
	expectedSections := []string{
		"Basic Navigation",
		"Scrolling",
		"Fuzzy Search",
		"View Controls",
		"Clipboard Operations",
	}

	sectionTitles := make(map[string]bool)
	for _, section := range sections {
		sectionTitles[section.Title] = true
	}

	for _, expected := range expectedSections {
		found := false
		for title := range sectionTitles {
			if strings.Contains(title, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Missing expected section: %s", expected)
		}
	}
}
