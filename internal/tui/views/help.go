// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/components"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/messages"
)

// HelpView displays help documentation as a standalone view
type HelpView struct {
	client APIClient
	cache  *cache.SimpleCache

	// Embed the existing help component to preserve scrolling functionality
	helpComponent *components.HelpView

	// Standard view fields
	layout *components.WindowLayout
	width  int
	height int
	keys   components.KeyMap

	// Implement CoreViewKeymap for proper key handling
	disabledKeys map[string]bool

	// Copy feedback message
	copiedMessage string
}

// NewHelpView creates a new standalone help view instance
func NewHelpView(client APIClient, cache *cache.SimpleCache) *HelpView {
	return &HelpView{
		client:        client,
		cache:         cache,
		helpComponent: components.NewHelpView(),
		keys:          components.DefaultKeyMap,
		disabledKeys:  map[string]bool{
			// Help view doesn't need to disable any keys
		},
		// Don't initialize layout here - wait for WindowSizeMsg
		layout: nil,
		width:  0,
		height: 0,
	}
}

// Init initializes the help view
func (h *HelpView) Init() tea.Cmd {
	debug.LogToFilef("📚 HELP: Initializing standalone help view\n")
	// Don't send WindowSizeMsg here - wait for app to send it
	return nil
}

// Update handles all messages for the help view
func (h *HelpView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return h.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		return h.handleKeyMsg(msg)

	case components.ClipboardBlinkMsg:
		// Pass through to help component for blink animation
		updatedHelp, cmd := h.helpComponent.Update(msg)
		h.helpComponent = updatedHelp
		return h, cmd

	case helpClearMessageMsg:
		h.copiedMessage = ""
		return h, nil
	}

	// Pass other messages to the help component
	updatedHelp, cmd := h.helpComponent.Update(msg)
	h.helpComponent = updatedHelp
	return h, cmd
}

// View renders the help view
func (h *HelpView) View() string {
	// Safety check for nil layout
	if h.layout == nil || h.width == 0 || h.height == 0 {
		return ""
	}

	if !h.layout.IsValidDimensions() {
		return h.layout.GetMinimalView("Help - Terminal too small")
	}

	// The help component already has excellent rendering with scrollbar
	// Just use it directly since it handles all the complex scrolling logic
	return h.helpComponent.View()
}

// handleWindowSizeMsg handles terminal resize events
func (h *HelpView) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	h.width = msg.Width
	h.height = msg.Height

	// Initialize layout on first WindowSizeMsg only
	if h.layout == nil {
		h.layout = components.NewWindowLayout(msg.Width, msg.Height)
		debug.LogToFilef("📐 HELP INIT: Created layout with %dx%d 📐\n", msg.Width, msg.Height)
	} else {
		h.layout.Update(msg.Width, msg.Height)
	}

	// Update the help component size
	h.helpComponent.SetSize(msg.Width, msg.Height)

	debug.LogToFilef("🔄 HELP: Window resized to %dx%d\n", msg.Width, msg.Height)
	return h, nil
}

// handleKeyMsg handles keyboard input
func (h *HelpView) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Check for navigation keys first
	switch msg.String() {
	case "q", "esc":
		// Navigate back to previous view
		debug.LogToFilef("🔙 HELP: Navigating back from help view\n")
		return h, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}

	case "Q":
		// Force quit
		debug.LogToFilef("⛔ HELP: Force quit from help view\n")
		return h, tea.Quit

	case "?":
		// Toggle help (which means go back since we're already in help)
		debug.LogToFilef("❓ HELP: Help toggle pressed, navigating back\n")
		return h, func() tea.Msg {
			return messages.NavigateBackMsg{}
		}

	case "J":
		// Capital J for half-page down scrolling
		debug.LogToFilef("📄 HELP: Half-page down scroll\n")
		updatedHelp, cmd := h.helpComponent.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
		h.helpComponent = updatedHelp
		return h, cmd

	case "K":
		// Capital K for half-page up scrolling
		debug.LogToFilef("📄 HELP: Half-page up scroll\n")
		updatedHelp, cmd := h.helpComponent.Update(tea.KeyMsg{Type: tea.KeyCtrlU})
		h.helpComponent = updatedHelp
		return h, cmd

	default:
		// Pass all other keys to the help component for scrolling and copying
		updatedHelp, cmd := h.helpComponent.Update(msg)
		h.helpComponent = updatedHelp

		// The help component handles its own status line and copying

		return h, cmd
	}
}

// Implement CoreViewKeymap interface

// IsKeyDisabled returns whether a key is disabled in this view
func (h *HelpView) IsKeyDisabled(keyString string) bool {
	return h.disabledKeys[keyString]
}

// HandleKey provides custom key handling for the help view
func (h *HelpView) HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	// Let the standard Update method handle all keys
	return false, h, nil
}

// Helper methods


// helpClearMessageMsg is used to clear temporary messages in help view
type helpClearMessageMsg struct{}
