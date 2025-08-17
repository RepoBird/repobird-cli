// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package components

import (
	"github.com/charmbracelet/lipgloss"
)

// Header represents a universal header component
type Header struct {
	width int
	title string
	style lipgloss.Style
}

// NewHeader creates a new header component
func NewHeader(title string) *Header {
	return &Header{
		title: title,
		style: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63")).
			PaddingLeft(1),
	}
}

// SetWidth sets the width of the header
func (h *Header) SetWidth(width int) *Header {
	h.width = width
	return h
}

// SetTitle sets the title of the header
func (h *Header) SetTitle(title string) *Header {
	h.title = title
	return h
}

// SetStyle sets the style of the header
func (h *Header) SetStyle(style lipgloss.Style) *Header {
	h.style = style
	return h
}

// Render renders the header
func (h *Header) Render() string {
	if h.width <= 0 {
		return h.style.Render(h.title)
	}
	return h.style.Width(h.width).Render(h.title)
}

// DefaultHeader creates the default header for Repobird CLI
func DefaultHeader(width int) string {
	return NewHeader("Repobird.ai CLI").SetWidth(width).Render()
}
