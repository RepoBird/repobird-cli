package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/keymap"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/tui/views"
)

type App struct {
	client      APIClient
	viewStack   []tea.Model // Navigation history
	current     tea.Model
	cache       *cache.SimpleCache
	keyRegistry *keymap.CoreKeyRegistry // Central key processing
}

func NewApp(client APIClient) *App {
	return &App{
		client:      client,
		cache:       cache.NewSimpleCache(),
		keyRegistry: keymap.NewCoreKeyRegistry(),
	}
}

// Init implements tea.Model interface - initializes with dashboard view
func (a *App) Init() tea.Cmd {
	// Initialize shared cache
	_ = a.cache.LoadFromDisk()

	// Initialize with dashboard view
	a.current = views.NewDashboardView(a.client)
	return a.current.Init()
}

// Update implements tea.Model interface - handles navigation and delegates to current view
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle navigation messages first
	if navMsg, ok := msg.(messages.NavigationMsg); ok {
		debug.LogToFilef("🚀 APP: Received NavigationMsg: %T 🚀\n", navMsg)
		return a.handleNavigation(navMsg)
	}

	// Centralized key processing
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		debug.LogToFilef("⌨️ APP: Received KeyMsg: '%s' ⌨️\n", keyMsg.String())
		if handled, model, cmd := a.processKeyWithFiltering(keyMsg); handled {
			debug.LogToFilef("✋ APP: Key '%s' was HANDLED by centralized processor ✋\n", keyMsg.String())
			return model, cmd
		}
		debug.LogToFilef("➡️ APP: Key '%s' NOT handled by centralized processor, delegating to view ➡️\n", keyMsg.String())
	}

	// Otherwise delegate to current view
	debug.LogToFilef("📤 APP: Delegating to current view: %T 📤\n", a.current)
	newModel, cmd := a.current.Update(msg)

	// Check if the model changed (old pattern - view created child)
	// We should handle this gracefully for backward compatibility
	if newModel != a.current {
		debug.LogToFilef("🔄 APP: View returned different model (old pattern) 🔄\n")
		// View returned a different model (old navigation pattern)
		// Accept it but this should be migrated to use messages
		a.current = newModel
	}

	return a, cmd
}

// handleNavigation processes navigation messages and manages view transitions
func (a *App) handleNavigation(msg messages.NavigationMsg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.NavigateToCreateMsg:
		// Save current view to stack
		a.viewStack = append(a.viewStack, a.current)

		// Create new view with minimal params
		a.current = views.NewCreateRunView(a.client)

		// Set navigation context if provided
		if msg.SelectedRepository != "" {
			a.setNavigationContext("selected_repo", msg.SelectedRepository)
		}

		return a, a.current.Init()

	case messages.NavigateToDetailsMsg:
		a.viewStack = append(a.viewStack, a.current)

		// Create with new minimal constructor pattern
		a.current = views.NewRunDetailsView(a.client, a.cache, msg.RunID)

		return a, a.current.Init()

	case messages.NavigateToDashboardMsg:
		// Clear stack - dashboard is home
		a.viewStack = nil
		a.current = views.NewDashboardView(a.client)
		// Clear navigation context when going home
		a.clearAllNavigationContext()
		return a, a.current.Init()

	case messages.NavigateBackMsg:
		if len(a.viewStack) > 0 {
			// Pop from stack
			a.current = a.viewStack[len(a.viewStack)-1]
			a.viewStack = a.viewStack[:len(a.viewStack)-1]

			// Refresh the view
			return a, a.current.Init()
		}
		// No history - go to dashboard
		return a.handleNavigation(messages.NavigateToDashboardMsg{})

	case messages.NavigateToListMsg:
		a.viewStack = append(a.viewStack, a.current)
		a.current = views.NewRunListView(a.client)

		// Restore selection if provided
		if msg.SelectedIndex > 0 {
			a.setNavigationContext("list_selected_index", msg.SelectedIndex)
		}

		return a, a.current.Init()

	case messages.NavigateToBulkMsg:
		a.viewStack = append(a.viewStack, a.current)
		// BulkView requires a concrete *api.Client, not the interface
		// For now, we'll skip bulk view if client is not the right type
		// This should be refactored to accept the interface
		if apiClient, ok := a.client.(*api.Client); ok {
			a.current = views.NewBulkView(apiClient)
			return a, a.current.Init()
		}
		// If not the right client type, just return without navigation
		return a, nil

	case messages.NavigateToFileViewerMsg:
		a.viewStack = append(a.viewStack, a.current)
		fileViewer, err := views.NewFileViewerView(a.client)
		if err != nil {
			// If file viewer creation fails, navigate to error view
			return a.handleNavigation(messages.NavigateToErrorMsg{
				Error:       err,
				Message:     "Failed to open file viewer",
				Recoverable: true,
			})
		}
		a.current = fileViewer
		return a, a.current.Init()

	case messages.NavigateToErrorMsg:
		if msg.Recoverable {
			// Push to stack so user can go back
			a.viewStack = append(a.viewStack, a.current)
		} else {
			// Replace current view, clear stack
			a.viewStack = nil
		}

		a.current = views.NewErrorView(msg.Error, msg.Message, msg.Recoverable)
		return a, a.current.Init()
	}

	return a, nil
}

// View implements tea.Model interface - delegates rendering to current view
func (a *App) View() string {
	if a.current == nil {
		return "Initializing..."
	}
	return a.current.View()
}

// Run starts the TUI application
func (a *App) Run() error {
	// Use App itself as the Model
	p := tea.NewProgram(a, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// Navigation context helper methods - delegate to cache

func (a *App) setNavigationContext(key string, value interface{}) {
	a.cache.SetNavigationContext(key, value)
}

func (a *App) clearAllNavigationContext() {
	a.cache.ClearAllNavigationContext()
}

func (a *App) getNavigationContext(key string) interface{} {
	return a.cache.GetNavigationContext(key)
}

// processKeyWithFiltering is the centralized key processor that handles all key filtering and routing
func (a *App) processKeyWithFiltering(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	keyString := keyMsg.String()
	debug.LogToFilef("🔧 PROCESSOR: Processing key '%s' 🔧\n", keyString)

	// Check if current view implements CoreViewKeymap
	if viewKeymap, hasKeymap := a.current.(keymap.CoreViewKeymap); hasKeymap {
		debug.LogToFilef("✅ PROCESSOR: View %T implements CoreViewKeymap ✅\n", a.current)
		
		// First check if view wants to disable this key entirely
		if viewKeymap.IsKeyDisabled(keyString) {
			debug.LogToFilef("🚫 PROCESSOR: Key '%s' is DISABLED by view - returning handled=true 🚫\n", keyString)
			// Key is disabled - ignore it completely
			return true, a, nil
		}
		debug.LogToFilef("✅ PROCESSOR: Key '%s' is NOT disabled by view ✅\n", keyString)

		// Check if view wants to handle this key with custom logic
		if handled, model, cmd := viewKeymap.HandleKey(keyMsg); handled {
			debug.LogToFilef("🎯 PROCESSOR: Key '%s' handled by view's custom handler 🎯\n", keyString)
			// View provided custom handling
			return true, model, cmd
		}
		debug.LogToFilef("➡️ PROCESSOR: Key '%s' not handled by view's custom handler ➡️\n", keyString)
	} else {
		debug.LogToFilef("❌ PROCESSOR: View %T does NOT implement CoreViewKeymap ❌\n", a.current)
	}

	// Get the default action for this key from registry
	action := a.keyRegistry.GetAction(keyString)
	debug.LogToFilef("🗂️ PROCESSOR: Key '%s' maps to action: %v 🗂️\n", keyString, action)

	// Handle global actions that should always work regardless of view state
	if keymap.IsGlobalAction(action) {
		debug.LogToFilef("🌍 PROCESSOR: Key '%s' is GLOBAL action - handling 🌍\n", keyString)
		return a.handleGlobalAction(action, keyMsg)
	}

	// Handle navigation actions
	if keymap.IsNavigationAction(action) {
		debug.LogToFilef("🧭 PROCESSOR: Key '%s' is NAVIGATION action - handling 🧭\n", keyString)
		return a.handleNavigationAction(action, keyMsg)
	}

	debug.LogToFilef("📋 PROCESSOR: Key '%s' is VIEW-SPECIFIC - returning handled=false 📋\n", keyString)
	// For ActionViewSpecific or unknown keys, let the view handle them
	return false, a, nil
}

// handleGlobalAction processes global actions like force quit
func (a *App) handleGlobalAction(action keymap.KeyAction, keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	switch action {
	case keymap.ActionGlobalQuit:
		// Force quit - always works
		a.cache.SaveToDisk()
		return true, a, tea.Quit
	case keymap.ActionGlobalHelp:
		// Global help - could be implemented later
		return false, a, nil
	default:
		return false, a, nil
	}
}

// handleNavigationAction processes navigation actions like back, new, etc.
func (a *App) handleNavigationAction(action keymap.KeyAction, keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	var navMsg messages.NavigationMsg

	switch action {
	case keymap.ActionNavigateBack:
		navMsg = messages.NavigateBackMsg{}
	case keymap.ActionNavigateBulk:
		navMsg = messages.NavigateToBulkMsg{}
	case keymap.ActionNavigateNew:
		// For 'n' key, only handle if we're not in an input field or specific context
		// This could be enhanced with more context awareness
		navMsg = nil // Let view handle 'n' for now
	case keymap.ActionNavigateRefresh:
		// Let view handle refresh for now
		navMsg = nil
	case keymap.ActionNavigateQuit:
		// Regular quit - save and quit
		a.cache.SaveToDisk()
		return true, a, tea.Quit
	case keymap.ActionNavigateHelp:
		// Let view handle help for now
		navMsg = nil
	default:
		return false, a, nil
	}

	if navMsg != nil {
		model, cmd := a.handleNavigation(navMsg)
		return true, model, cmd
	}

	return false, a, nil
}
