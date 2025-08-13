package tui

import (
	"context"
	"strings"
	"time"
	
	"github.com/charmbracelet/bubbles/spinner"
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
	client         APIClient
	viewStack      []tea.Model // Navigation history
	current        tea.Model
	cache          *cache.SimpleCache
	keyRegistry    *keymap.CoreKeyRegistry // Central key processing
	width          int                      // Current window width
	height         int                      // Current window height
	authenticated  bool                     // Whether initial auth is complete
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

// Init implements tea.Model interface - initializes with dashboard view
func (a *App) Init() tea.Cmd {
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
		
		// Initialize dashboard view now that we have user context
		a.current = views.NewDashboardView(a.client)
		
		// Initialize the view with current window size if available
		var cmds []tea.Cmd
		cmds = append(cmds, a.current.Init())
		
		// Only send window size if we have valid dimensions
		if a.width > 0 && a.height > 0 {
			debug.LogToFilef("📐 APP: Sending stored window size: %dx%d\n", a.width, a.height)
			cmds = append(cmds, func() tea.Msg {
				return tea.WindowSizeMsg{Width: a.width, Height: a.height}
			})
		} else {
			debug.LogToFile("📐 APP: No valid window size yet, waiting for terminal to send it\n")
			// Don't send an empty WindowSizeMsg - wait for the terminal to send the real one
		}
		return a, tea.Batch(cmds...)
	}
	
	// Don't process other messages until authenticated
	if !a.authenticated {
		// Still handle window size to store dimensions
		if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
			a.width = wsMsg.Width
			a.height = wsMsg.Height
		}
		return a, nil
	}
	
	// Handle navigation messages first
	if navMsg, ok := msg.(messages.NavigationMsg); ok {
		debug.LogToFilef("🚀 APP: Received NavigationMsg: %T 🚀\n", navMsg)
		return a.handleNavigation(navMsg)
	}

	// Handle window size messages centrally
	if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
		debug.LogToFilef("📐 APP: Received WindowSizeMsg: width=%d, height=%d 📐\n", wsMsg.Width, wsMsg.Height)
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
		debug.LogToFilef("⌨️ APP: Received KeyMsg: '%s' ⌨️\n", keyMsg.String())
		if handled, model, cmd := a.processKeyWithFiltering(keyMsg); handled {
			debug.LogToFilef("✋ APP: Key '%s' was HANDLED by centralized processor ✋\n", keyMsg.String())
			debug.LogToFilef("🔍 APP: After processKey - model type=%T, cmd is nil=%v\n", model, cmd == nil)
			
			// If the model is the app itself and we have a command, execute it
			if appModel, isApp := model.(*App); isApp && cmd != nil {
				debug.LogToFilef("📦 APP: Model is App, executing command\n")
				return appModel, cmd
			}
			return model, cmd
		}
		debug.LogToFilef("➡️ APP: Key '%s' NOT handled by centralized processor, delegating to view ➡️\n", keyMsg.String())
	}

	// Otherwise delegate to current view
	// Skip debug logging for spinner messages (too spammy)
	if _, isSpinner := msg.(spinner.TickMsg); !isSpinner {
		debug.LogToFilef("📤 APP: Delegating to current view: %T 📤\n", a.current)
	}
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
		debug.LogToFile("DEBUG: App - handling NavigateToCreateMsg\n")
		// Save current view to stack
		a.viewStack = append(a.viewStack, a.current)

		// Create new view with minimal params
		debug.LogToFile("DEBUG: App - creating new CreateRunView\n")
		a.current = views.NewCreateRunView(a.client, a.cache)

		// Set navigation context if provided
		if msg.SelectedRepository != "" {
			debug.LogToFilef("DEBUG: App - setting navigation context: selected_repo=%s\n", msg.SelectedRepository)
			a.setNavigationContext("selected_repo", msg.SelectedRepository)
		}

		// Send current window dimensions to the new view if we have them
		var cmds []tea.Cmd
		cmds = append(cmds, a.current.Init())
		if a.width > 0 && a.height > 0 {
			debug.LogToFilef("📐 CREATE NAV: Sending WindowSizeMsg to new CreateRunView: %dx%d 📐\n", a.width, a.height)
			cmds = append(cmds, func() tea.Msg {
				return tea.WindowSizeMsg{Width: a.width, Height: a.height}
			})
		} else {
			debug.LogToFile("⚠️ CREATE NAV: No stored dimensions to send to CreateRunView ⚠️\n")
		}
		return a, tea.Batch(cmds...)

	case messages.NavigateToDetailsMsg:
		a.viewStack = append(a.viewStack, a.current)

		// Create with new minimal constructor pattern
		a.current = views.NewRunDetailsView(a.client, a.cache, msg.RunID)

		// Send current window dimensions to the new view if we have them
		var cmds []tea.Cmd
		cmds = append(cmds, a.current.Init())
		if a.width > 0 && a.height > 0 {
			debug.LogToFilef("📐 DETAILS NAV: Sending WindowSizeMsg to new DetailsView: %dx%d 📐\n", a.width, a.height)
			cmds = append(cmds, func() tea.Msg {
				return tea.WindowSizeMsg{Width: a.width, Height: a.height}
			})
		} else {
			debug.LogToFile("⚠️ DETAILS NAV: No stored dimensions to send to DetailsView ⚠️\n")
		}
		return a, tea.Batch(cmds...)

	case messages.NavigateToDashboardMsg:
		// Clear stack - dashboard is home
		a.viewStack = nil
		a.current = views.NewDashboardView(a.client)
		// Clear navigation context when going home
		a.clearAllNavigationContext()
		return a, a.current.Init()

	case messages.NavigateBackMsg:
		debug.LogToFilef("🔙 HANDLE NAV: NavigateBackMsg - stack length=%d\n", len(a.viewStack))
		if len(a.viewStack) > 0 {
			// Pop from stack
			previousView := a.viewStack[len(a.viewStack)-1]
			debug.LogToFilef("🔙 HANDLE NAV: Popping from stack, going back to %T\n", previousView)
			a.current = previousView
			a.viewStack = a.viewStack[:len(a.viewStack)-1]

			// Refresh the view
			debug.LogToFilef("🔄 HANDLE NAV: Initializing previous view %T\n", a.current)
			return a, a.current.Init()
		}
		// No history - go to dashboard
		debug.LogToFilef("🏠 HANDLE NAV: No history, going to dashboard\n")
		return a.handleNavigation(messages.NavigateToDashboardMsg{})

	case messages.NavigateToListMsg:
		a.viewStack = append(a.viewStack, a.current)
		a.current = views.NewRunListView(a.client)

		// Restore selection if provided
		if msg.SelectedIndex > 0 {
			a.setNavigationContext("list_selected_index", msg.SelectedIndex)
		}

		return a, a.current.Init()

	case messages.NavigateToStatusMsg:
		debug.LogToFilef("🏥 STATUS NAV: Navigating to status view 🏥\n")
		a.viewStack = append(a.viewStack, a.current)
		a.current = views.NewStatusView(a.client)

		// Send current window dimensions to the new view if we have them
		var cmds []tea.Cmd
		cmds = append(cmds, a.current.Init())
		if a.width > 0 && a.height > 0 {
			debug.LogToFilef("📐 STATUS NAV: Sending WindowSizeMsg to new StatusView: %dx%d 📐\n", a.width, a.height)
			cmds = append(cmds, func() tea.Msg {
				return tea.WindowSizeMsg{Width: a.width, Height: a.height}
			})
		} else {
			debug.LogToFile("⚠️ STATUS NAV: No stored dimensions to send to StatusView ⚠️\n")
		}
		return a, tea.Batch(cmds...)

	case messages.NavigateToBulkMsg:
		debug.LogToFilef("🏗️ BULK NAV: Attempting to navigate to bulk view 🏗️\n")
		debug.LogToFilef("🔍 BULK NAV: Client type: %T 🔍\n", a.client)
		a.viewStack = append(a.viewStack, a.current)
		// BulkView requires a concrete *api.Client, not the interface
		// For now, we'll skip bulk view if client is not the right type
		// This should be refactored to accept the interface
		if apiClient, ok := a.client.(*api.Client); ok {
			debug.LogToFilef("✅ BULK NAV: Client type is correct, creating BulkView ✅\n")
			a.current = views.NewBulkView(apiClient)
			
			// Send current window dimensions to the new view if we have them
			var cmds []tea.Cmd
			cmds = append(cmds, a.current.Init())
			if a.width > 0 && a.height > 0 {
				debug.LogToFilef("📐 BULK NAV: Sending WindowSizeMsg to new BulkView: %dx%d 📐\n", a.width, a.height)
				cmds = append(cmds, func() tea.Msg {
					return tea.WindowSizeMsg{Width: a.width, Height: a.height}
				})
			} else {
				debug.LogToFile("⚠️ BULK NAV: No stored dimensions to send to BulkView ⚠️\n")
			}
			return a, tea.Batch(cmds...)
		}
		debug.LogToFilef("❌ BULK NAV: Client type is WRONG - cannot create BulkView! ❌\n")
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
	if !a.authenticated {
		return a.renderAuthLoadingView()
	}
	
	if a.current == nil {
		return a.renderInitializingView()
	}
	
	// Debug: Log app view rendering
	debug.LogToFilef("🎭 APP VIEW: Rendering view %T with app dimensions w=%d h=%d 🎭\n", 
		a.current, a.width, a.height)
	
	view := a.current.View()
	
	// Debug: Log the length of the returned view string
	lines := strings.Count(view, "\n") + 1
	debug.LogToFilef("🎭 APP VIEW: Returned view has %d lines 🎭\n", lines)
	
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
	debug.LogToFilef("🔧 PROCESSOR: Processing key '%s' 🔧\n", keyString)

	// Check if current view implements CoreViewKeymap
	if viewKeymap, hasKeymap := a.current.(keymap.CoreViewKeymap); hasKeymap {
		debug.LogToFilef("✅ PROCESSOR: View %T implements CoreViewKeymap ✅\n", a.current)
		
		// First check if view wants to disable this key entirely
		if viewKeymap.IsKeyDisabled(keyString) {
			debug.LogToFilef("🚫 PROCESSOR: Key '%s' is DISABLED by view - passing to Update for typing 🚫\n", keyString)
			// Key is disabled for NAVIGATION but should still go to Update() for typing
			// Return false so the key reaches the view's Update method
			return false, a, nil
		}
		debug.LogToFilef("✅ PROCESSOR: Key '%s' is NOT disabled by view ✅\n", keyString)

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
	debug.LogToFilef("🎯 NAV ACTION: Processing action %v for key '%s' 🎯\n", action, keyMsg.String())
	debug.LogToFilef("🔍 NAV ACTION: Current view type: %T\n", a.current)
	var navMsg messages.NavigationMsg

	switch action {
	case keymap.ActionNavigateBack:
		debug.LogToFilef("⬅️ NAV ACTION: Creating NavigateBackMsg ⬅️\n")
		navMsg = messages.NavigateBackMsg{}
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
		a.cache.SaveToDisk()
		return true, a, tea.Quit
	case keymap.ActionNavigateHelp:
		debug.LogToFilef("❓ NAV ACTION: Ignoring ActionNavigateHelp (let view handle) ❓\n")
		// Let view handle help for now
		navMsg = nil
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
