// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/services"
	"github.com/repobird/repobird-cli/internal/tui/cache"
	"github.com/repobird/repobird-cli/internal/tui/debug"
	"github.com/repobird/repobird-cli/internal/tui/keymap"
	"github.com/repobird/repobird-cli/internal/tui/messages"
	"github.com/repobird/repobird-cli/internal/tui/views"
)

type App struct {
	client        APIClient
	viewStack     []tea.Model // Navigation history
	current       tea.Model
	cache         *cache.SimpleCache
	keyRegistry   *keymap.CoreKeyRegistry // Central key processing
	width         int                     // Current window width
	height        int                     // Current window height
	authenticated bool                    // Whether initial auth is complete
	debugLoading  bool                    // Debug mode to stay on loading screen
}

// authCompleteMsg is sent when authentication and cache initialization is complete
type authCompleteMsg struct {
	userInfo *models.UserInfo
	err      error
}

func NewApp(client APIClient) *App {
	// Don't create cache yet - wait until we authenticate
	return &App{
		client:      client,
		cache:       nil, // Will be initialized after authentication
		keyRegistry: keymap.NewCoreKeyRegistry(),
	}
}

// NewAppWithDebugLoading creates a new App with debug loading mode enabled
func NewAppWithDebugLoading(client APIClient) *App {
	app := NewApp(client)
	app.debugLoading = true
	return app
}

// Init implements tea.Model interface - initializes with dashboard view
func (a *App) Init() tea.Cmd {
	// If in debug loading mode, skip authentication and go straight to dashboard loading view
	if a.debugLoading {
		debug.LogToFile("🐛 DEBUG LOADING: Skipping authentication, showing dashboard loading view 🐛\n")
		a.authenticated = true
		// Create minimal cache without authentication
		a.cache = cache.NewSimpleCache()
		// Create dashboard in loading state
		a.current = views.NewDashboardViewDebugLoading(a.client, a.cache)
		return a.current.Init()
	}
	
	// Just authenticate, don't request window size yet
	// The terminal will send the real window size automatically
	return a.authenticateAndInitCache()
}

// Update implements tea.Model interface - handles navigation and delegates to current view
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle authentication completion first
	if authMsg, ok := msg.(authCompleteMsg); ok {
		debug.LogToFile("🔐 APP: Authentication complete, initializing dashboard 🔐\n")
		a.authenticated = true

		// Log any authentication issues (but continue anyway)
		if authMsg.err != nil {
			debug.LogToFilef("⚠️ APP: Auth had error (continuing anyway): %v\n", authMsg.err)
		}

		// Clear cache on initial startup to ensure fresh data
		debug.LogToFile("🔄 APP: Initial startup - clearing cache to fetch fresh data 🔄\n")
		a.cache.Clear()

		// Initialize dashboard view now that we have user context
		a.current = views.NewDashboardView(a.client, a.cache)

		// Initialize the view with current window size if available
		var cmds []tea.Cmd
		cmds = append(cmds, a.current.Init())

		// Only send window size if we have valid dimensions
		if a.width > 0 && a.height > 0 {
			cmds = append(cmds, func() tea.Msg {
				return tea.WindowSizeMsg{Width: a.width, Height: a.height}
			})
		} else {
			debug.LogToFile("📐 APP: No valid window size yet, waiting for terminal to send it\n")
			// Don't send an empty WindowSizeMsg - wait for the terminal to send the real one
		}
		return a, tea.Batch(cmds...)
	}

	// Don't process other messages until authenticated (except quit)
	if !a.authenticated {
		// Still handle window size to store dimensions
		if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
			a.width = wsMsg.Width
			a.height = wsMsg.Height
		}
		// Allow quitting even during loading
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if keyMsg.String() == "q" || keyMsg.String() == "ctrl+c" {
				return a, tea.Quit
			}
		}
		return a, nil
	}

	// Handle navigation messages first
	if navMsg, ok := msg.(messages.NavigationMsg); ok {
		return a.handleNavigation(navMsg)
	}

	// Handle window size messages centrally
	if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
		// Store current dimensions
		a.width = wsMsg.Width
		a.height = wsMsg.Height
		// Delegate to current view - all views should handle this
		newModel, cmd := a.current.Update(msg)
		if newModel != a.current {
			a.current = newModel
		}
		return a, cmd
	}

	// Centralized key processing
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if handled, model, cmd := a.processKeyWithFiltering(keyMsg); handled {
			// If the model is the app itself and we have a command, execute it
			if appModel, isApp := model.(*App); isApp && cmd != nil {
				return appModel, cmd
			}
			return model, cmd
		}
	}

	// Otherwise delegate to current view
	// Skip debug logging for spinner messages (too spammy)
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
		return a.navigateToCreate(msg)

	case messages.NavigateToDetailsMsg:
		return a.navigateToDetails(msg)

	case messages.NavigateToDashboardMsg:
		return a.navigateToDashboard()

	case messages.NavigateBackMsg:
		return a.navigateBack()

	case messages.NavigateToListMsg:
		return a.navigateToList(msg)

	case messages.NavigateToStatusMsg:
		return a.navigateToStatus()

	case messages.NavigateToBulkMsg:
		return a.navigateToBulk()

	case messages.NavigateToBulkResultsMsg:
		return a.navigateToBulkResults()

	case messages.NavigateToFileViewerMsg:
		return a.navigateToFileViewer()

	case messages.NavigateToHelpMsg:
		return a.navigateToHelp()

	case messages.NavigateToExamplesMsg:
		return a.navigateToExamples()

	case messages.NavigateToErrorMsg:
		return a.navigateToError(msg)
	}

	return a, nil
}

// Helper methods for navigation to reduce cyclomatic complexity

// pushToStack adds the current view to the navigation stack
func (a *App) pushToStack() {
	a.viewStack = append(a.viewStack, a.current)
}

// initViewWithDimensions initializes a view and sends window dimensions if available
func (a *App) initViewWithDimensions() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, a.current.Init())
	if a.width > 0 && a.height > 0 {
		cmds = append(cmds, func() tea.Msg {
			return tea.WindowSizeMsg{Width: a.width, Height: a.height}
		})
	}
	return tea.Batch(cmds...)
}

// navigateToCreate handles navigation to the create run view
func (a *App) navigateToCreate(msg messages.NavigateToCreateMsg) (tea.Model, tea.Cmd) {
	debug.LogToFile("DEBUG: App - handling NavigateToCreateMsg\n")
	a.pushToStack()

	debug.LogToFile("DEBUG: App - creating new CreateRunView\n")
	a.current = views.NewCreateRunView(a.client, a.cache)

	if msg.SelectedRepository != "" {
		debug.LogToFilef("DEBUG: App - setting navigation context: selected_repo=%s\n", msg.SelectedRepository)
		a.setNavigationContext("selected_repo", msg.SelectedRepository)
	}

	return a, a.initViewWithDimensions()
}

// navigateToDetails handles navigation to the run details view
func (a *App) navigateToDetails(msg messages.NavigateToDetailsMsg) (tea.Model, tea.Cmd) {
	a.pushToStack()

	if msg.FromCreate {
		debug.LogToFile("📝 APP: Setting from_create flag in navigation context\n")
		a.cache.SetNavigationContext("from_create", true)
	}

	if msg.RunData != nil {
		a.current = views.NewRunDetailsViewWithData(a.client, a.cache, *msg.RunData)
	} else {
		debug.LogToFile("📡 APP: Creating Details view with RunID only - will load from cache/API 📡\n")
		a.current = views.NewRunDetailsView(a.client, a.cache, msg.RunID)
	}

	return a, a.initViewWithDimensions()
}

// navigateToDashboard handles navigation to the dashboard view
func (a *App) navigateToDashboard() (tea.Model, tea.Cmd) {
	a.viewStack = nil // Clear stack - dashboard is home

	if stateData := a.cache.GetNavigationContext("dashboardState"); stateData != nil {
		debug.LogToFilef("🔍 APP: Found dashboard state in navigation context: %+v 🔍\n", stateData)
		if state, ok := stateData.(map[string]interface{}); ok {
			// Extract state values with type assertions
			selectedRepoIdx, _ := state["selectedRepoIdx"].(int)
			selectedRunIdx, _ := state["selectedRunIdx"].(int)
			selectedDetailLine, _ := state["selectedDetailLine"].(int)
			focusedColumn, _ := state["focusedColumn"].(int)

			debug.LogToFilef("🏠 APP: Restoring Dashboard with saved state - repo=%d, run=%d, detail=%d, column=%d 🏠\n",
				selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn)
			a.current = views.NewDashboardViewWithState(a.client, a.cache, selectedRepoIdx, selectedRunIdx, selectedDetailLine, focusedColumn)
		} else {
			debug.LogToFile("🏠 APP: Invalid dashboard state format, creating fresh dashboard 🏠\n")
			a.current = views.NewDashboardView(a.client, a.cache)
		}
	} else {
		debug.LogToFile("🔍 APP: No dashboard state found in navigation context 🔍\n")
		debug.LogToFile("🏠 APP: Creating Dashboard view - hybrid cache will handle data caching 🏠\n")
		a.current = views.NewDashboardView(a.client, a.cache)
	}

	a.clearAllNavigationContext()
	return a, a.initViewWithDimensions()
}

// navigateBack handles navigation to the previous view
func (a *App) navigateBack() (tea.Model, tea.Cmd) {
	debug.LogToFilef("🔙 HANDLE NAV: NavigateBackMsg - stack length=%d\n", len(a.viewStack))
	if len(a.viewStack) > 0 {
		previousView := a.viewStack[len(a.viewStack)-1]
		debug.LogToFilef("🔙 HANDLE NAV: Popping from stack, going back to %T\n", previousView)
		a.current = previousView
		a.viewStack = a.viewStack[:len(a.viewStack)-1]

		debug.LogToFilef("🔄 HANDLE NAV: Initializing previous view %T\n", a.current)
		return a, a.current.Init()
	}
	debug.LogToFilef("🏠 HANDLE NAV: No history, going to dashboard\n")
	return a.navigateToDashboard()
}

// navigateToList handles navigation to the run list view
func (a *App) navigateToList(msg messages.NavigateToListMsg) (tea.Model, tea.Cmd) {
	a.pushToStack()
	a.current = views.NewRunListView(a.client)

	if msg.SelectedIndex > 0 {
		a.setNavigationContext("list_selected_index", msg.SelectedIndex)
	}

	return a, a.current.Init()
}

// navigateToStatus handles navigation to the status view
func (a *App) navigateToStatus() (tea.Model, tea.Cmd) {
	debug.LogToFilef("🏥 STATUS NAV: Navigating to status view 🏥\n")
	a.pushToStack()
	a.current = views.NewStatusView(a.client)
	return a, a.initViewWithDimensions()
}

// navigateToBulk handles navigation to the bulk view
func (a *App) navigateToBulk() (tea.Model, tea.Cmd) {
	debug.LogToFilef("🏗️ BULK NAV: Attempting to navigate to bulk view 🏗️\n")
	debug.LogToFilef("🔍 BULK NAV: Client type: %T 🔍\n", a.client)
	a.pushToStack()

	// BulkView requires a concrete *api.Client, not the interface
	if apiClient, ok := a.client.(*api.Client); ok {
		debug.LogToFilef("✅ BULK NAV: Client type is correct, creating BulkView ✅\n")
		a.current = views.NewBulkView(apiClient, a.cache)
		return a, a.initViewWithDimensions()
	}

	debug.LogToFilef("❌ BULK NAV: Client type is WRONG - cannot create BulkView! ❌\n")
	return a, nil
}

// navigateToBulkResults handles navigation to the bulk results view
func (a *App) navigateToBulkResults() (tea.Model, tea.Cmd) {
	debug.LogToFilef("📊 BULK RESULTS NAV: Navigating to bulk results view 📊\n")
	a.pushToStack()

	// BulkResultsView requires a concrete *api.Client
	if apiClient, ok := a.client.(*api.Client); ok {
		debug.LogToFilef("✅ BULK RESULTS NAV: Creating BulkResultsView ✅\n")
		a.current = views.NewBulkResultsView(apiClient, a.cache)
		return a, a.initViewWithDimensions()
	}

	debug.LogToFilef("❌ BULK RESULTS NAV: Client type is wrong - cannot create BulkResultsView! ❌\n")
	return a, nil
}

// navigateToFileViewer handles navigation to the file viewer view
func (a *App) navigateToFileViewer() (tea.Model, tea.Cmd) {
	a.pushToStack()
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
}

// navigateToHelp handles navigation to the help view
func (a *App) navigateToHelp() (tea.Model, tea.Cmd) {
	debug.LogToFilef("📚 HELP NAV: Navigating to help view 📚\n")
	a.pushToStack()
	a.current = views.NewHelpView(a.client, a.cache)
	return a, a.initViewWithDimensions()
}

// navigateToExamples handles navigation to the examples view
func (a *App) navigateToExamples() (tea.Model, tea.Cmd) {
	debug.LogToFilef("📚 EXAMPLES NAV: Navigating to examples view 📚\n")
	a.pushToStack()
	a.current = views.NewExamplesView(a.client, a.cache)
	return a, a.initViewWithDimensions()
}

// navigateToError handles navigation to the error view
func (a *App) navigateToError(msg messages.NavigateToErrorMsg) (tea.Model, tea.Cmd) {
	if msg.Recoverable {
		a.pushToStack()
	} else {
		a.viewStack = nil
	}

	a.current = views.NewErrorView(msg.Error, msg.Message, msg.Recoverable)
	return a, a.initViewWithDimensions()
}

// View implements tea.Model interface - delegates rendering to current view
func (a *App) View() string {
	if !a.authenticated {
		return a.renderAuthLoadingView()
	}

	if a.current == nil {
		return a.renderInitializingView()
	}

	view := a.current.View()

	return view
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

	// Check if current view implements CoreViewKeymap
	if viewKeymap, hasKeymap := a.current.(keymap.CoreViewKeymap); hasKeymap {
		// First check if view wants to disable this key entirely
		if viewKeymap.IsKeyDisabled(keyString) {
			debug.LogToFilef("🚫 PROCESSOR: Key '%s' is DISABLED by view - passing to Update for typing 🚫\n", keyString)
			// Key is disabled for NAVIGATION but should still go to Update() for typing
			// Return false so the key reaches the view's Update method
			return false, a, nil
		}

		// Check if view wants to handle this key with custom logic
		if handled, model, cmd := viewKeymap.HandleKey(keyMsg); handled {
			debug.LogToFilef("🎯 PROCESSOR: Key '%s' handled by view's custom handler 🎯\n", keyString)
			debug.LogToFilef("🔍 PROCESSOR: handled=%v, model type=%T, cmd is nil=%v\n", handled, model, cmd == nil)

			// IMPORTANT: If the view returns itself as the model, we need to update a.current
			// This ensures the view's state changes are preserved
			if model != nil && model != a {
				debug.LogToFilef("📝 PROCESSOR: Updating a.current from %T to %T\n", a.current, model)
				a.current = model
			}

			// View provided custom handling - return the app as the model so commands work
			return true, a, cmd
		}
	}

	// Get the default action for this key from registry
	action := a.keyRegistry.GetAction(keyString)

	// Handle global actions that should always work regardless of view state
	if keymap.IsGlobalAction(action) {
		return a.handleGlobalAction(action, keyMsg)
	}

	// Handle navigation actions
	if keymap.IsNavigationAction(action) {
		debug.LogToFilef("🧭 PROCESSOR: Key '%s' is NAVIGATION action - handling 🧭\n", keyString)
		return a.handleNavigationAction(action, keyMsg)
	}

	// For ActionViewSpecific or unknown keys, let the view handle them
	return false, a, nil
}

// handleGlobalAction processes global actions like force quit
func (a *App) handleGlobalAction(action keymap.KeyAction, _ tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	switch action {
	case keymap.ActionGlobalQuit:
		// Force quit - always works
		_ = a.cache.SaveToDisk()
		return true, a, tea.Quit
	case keymap.ActionGlobalHelp:
		// Global help - could be implemented later
		return false, a, nil
	case keymap.ActionNavigateBack, keymap.ActionNavigateBulk, keymap.ActionNavigateNew,
		keymap.ActionNavigateRefresh, keymap.ActionNavigateQuit, keymap.ActionNavigateHelp,
		keymap.ActionNavigateToDashboard, keymap.ActionViewSpecific, keymap.ActionIgnore:
		// These are navigation actions, not global actions
		return false, a, nil
	default:
		return false, a, nil
	}
}

// handleNavigationAction processes navigation actions like back, new, etc.
func (a *App) handleNavigationAction(action keymap.KeyAction, keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd) {
	debug.LogToFilef("🎯 NAV ACTION: Processing action %v for key '%s' 🎯\n", action, keyMsg.String())
	debug.LogToFilef("🔍 NAV ACTION: Current view type: %T\n", a.current)
	var navMsg messages.NavigationMsg

	switch action {
	case keymap.ActionNavigateBack:
		debug.LogToFilef("⬅️ NAV ACTION: Creating NavigateBackMsg ⬅️\n")
		navMsg = messages.NavigateBackMsg{}
	case keymap.ActionNavigateToDashboard:
		debug.LogToFilef("🏠 NAV ACTION: Creating NavigateToDashboardMsg 🏠\n")
		navMsg = messages.NavigateToDashboardMsg{}
	case keymap.ActionNavigateBulk:
		debug.LogToFilef("📦 NAV ACTION: Creating NavigateToBulkMsg 📦\n")
		navMsg = messages.NavigateToBulkMsg{}
	case keymap.ActionNavigateNew:
		debug.LogToFilef("➕ NAV ACTION: Ignoring ActionNavigateNew (let view handle) ➕\n")
		// For 'n' key, only handle if we're not in an input field or specific context
		// This could be enhanced with more context awareness
		navMsg = nil // Let view handle 'n' for now
	case keymap.ActionNavigateRefresh:
		debug.LogToFilef("🔄 NAV ACTION: Ignoring ActionNavigateRefresh (let view handle) 🔄\n")
		// Let view handle refresh for now
		navMsg = nil
	case keymap.ActionNavigateQuit:
		debug.LogToFilef("🚪 NAV ACTION: Quitting application 🚪\n")
		// Regular quit - save and quit
		_ = a.cache.SaveToDisk()
		return true, a, tea.Quit
	case keymap.ActionNavigateHelp:
		debug.LogToFilef("❓ NAV ACTION: Processing ActionNavigateHelp - creating NavigateToHelpMsg ❓\n")
		navMsg = messages.NavigateToHelpMsg{}
	case keymap.ActionViewSpecific, keymap.ActionIgnore, keymap.ActionGlobalQuit, keymap.ActionGlobalHelp:
		// These are not navigation actions
		debug.LogToFilef("❓ NAV ACTION: Non-navigation action %v ❓\n", action)
		return false, a, nil
	default:
		debug.LogToFilef("❓ NAV ACTION: Unknown action %v ❓\n", action)
		return false, a, nil
	}

	if navMsg != nil {
		debug.LogToFilef("📨 NAV ACTION: Calling handleNavigation with %T 📨\n", navMsg)
		debug.LogToFilef("🔍 NAV ACTION: ViewStack length: %d\n", len(a.viewStack))
		model, cmd := a.handleNavigation(navMsg)
		debug.LogToFilef("✅ NAV ACTION: handleNavigation returned model type=%T, cmd nil=%v\n", model, cmd == nil)
		return true, model, cmd
	}

	debug.LogToFilef("🔄 NAV ACTION: No message to send, returning false 🔄\n")
	return false, a, nil
}

// renderAuthLoadingView renders the authentication loading screen with proper layout
func (a *App) renderAuthLoadingView() string {
	if a.width <= 0 || a.height <= 0 {
		return "🔐 Authenticating..."
	}

	// Simple centered message without creating a layout (follows critical pattern)
	return lipgloss.NewStyle().
		Width(a.width).
		Height(a.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render("🔐 Authenticating and initializing cache...")
}

// renderInitializingView renders the initialization screen with proper layout
func (a *App) renderInitializingView() string {
	if a.width <= 0 || a.height <= 0 {
		return "Initializing..."
	}

	// Simple centered message without creating a layout (follows critical pattern)
	return lipgloss.NewStyle().
		Width(a.width).
		Height(a.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render("⏳ Initializing views...")
}

// authenticateAndInitCache authenticates with the API and initializes the cache with user context
func (a *App) authenticateAndInitCache() tea.Cmd {
	return func() tea.Msg {
		// First try to authenticate to get user ID
		debug.LogToFile("🔐 AUTH: Starting authentication process...\n")

		// Use a shorter timeout to prevent hanging
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Check if we have a method to get user info
		type userInfoGetter interface {
			GetUserInfoWithContext(ctx context.Context) (*models.UserInfo, error)
		}

		var userInfo *models.UserInfo
		var err error
		var needsAuth = true

		// Try to get user info first (to set context before creating cache)
		if getter, ok := a.client.(userInfoGetter); ok {
			// First, try a quick check if we have cached auth by creating temp cache
			tempCache := cache.NewSimpleCache()
			_ = tempCache.LoadFromDisk()

			if tempCache.IsAuthCacheValid() {
				if authCache, found := tempCache.GetAuthCache(); found && authCache.UserInfo != nil {
					// Use cached user info
					services.SetCurrentUser(authCache.UserInfo)
					userInfo = authCache.UserInfo
					needsAuth = false
					debug.LogToFilef("🔐 AUTH: Using cached auth (valid for %v) - user ID=%d, email=%s 🔐\n",
						authCache.CacheDuration, authCache.UserInfo.ID, authCache.UserInfo.Email)
				}
			}

			if needsAuth {
				debug.LogToFile("🔐 AUTH: No valid cached auth, authenticating with API...\n")
				userInfo, err = getter.GetUserInfoWithContext(ctx)
				if err == nil && userInfo != nil {
					// Set the current user globally for cache operations
					services.SetCurrentUser(userInfo)
					debug.LogToFilef("🔐 AUTH: Successfully authenticated user ID=%d, email=%s 🔐\n", userInfo.ID, userInfo.Email)
				} else if err != nil {
					// Log the error but continue - we'll use anonymous mode
					debug.LogToFilef("⚠️ AUTH: Authentication failed (will use anonymous mode): %v ⚠️\n", err)
					// Don't return error - continue with anonymous cache
					err = nil
				}
			}
		} else {
			debug.LogToFile("⚠️ AUTH: Client doesn't support GetUserInfoWithContext, using anonymous cache ⚠️\n")
		}

		// Now create the cache with the correct user context
		debug.LogToFile("📦 AUTH: Creating cache with user context...\n")
		a.cache = cache.NewSimpleCache()
		_ = a.cache.LoadFromDisk()
		debug.LogToFile("✅ AUTH: Cache created and loaded\n")

		// If we just authenticated (not from cache), save the auth info
		if needsAuth && userInfo != nil {
			// Cache the auth info for 2 weeks
			if cacheErr := a.cache.SetAuthCache(userInfo); cacheErr != nil {
				debug.LogToFilef("⚠️ AUTH: Failed to cache auth info: %v ⚠️\n", cacheErr)
			} else {
				debug.LogToFile("✅ AUTH: Cached auth info for 2 weeks\n")
			}
		}

		if !needsAuth {
			debug.LogToFile("🔄 AUTH: Refreshing cache data (runs, repos, etc)...\n")
		}

		return authCompleteMsg{
			userInfo: userInfo,
			err:      err,
		}
	}
}
