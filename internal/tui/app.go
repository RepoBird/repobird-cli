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
	client    APIClient
	viewStack []tea.Model // Navigation history
	current   tea.Model
	cache     *cache.SimpleCache
}

func NewApp(client APIClient) *App {
	return &App{
		client: client,
		cache:  cache.NewSimpleCache(),
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
		return a.handleNavigation(navMsg)
	}

	// Handle quit message at app level
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.String() == "ctrl+c" {
			return a, tea.Quit
		}

		// Check if current view implements ViewKeymap and if navigation keys should be intercepted
		if viewWithKeymap, hasKeymap := a.current.(keymap.ViewKeymap); hasKeymap {
			// Convert key press to NavigationKey and check if it's enabled
			if navKey := a.keyMsgToNavigationKey(keyMsg); navKey != "" {
				if !viewWithKeymap.IsNavigationKeyEnabled(navKey) {
					// Key is disabled for this view - ignore it
					return a, nil
				}
				// Key is enabled - convert to navigation message if appropriate
				if navMsg := a.navigationKeyToMessage(navKey); navMsg != nil {
					return a.handleNavigation(navMsg)
				}
			}
		}
	}

	// Otherwise delegate to current view
	newModel, cmd := a.current.Update(msg)

	// Check if the model changed (old pattern - view created child)
	// We should handle this gracefully for backward compatibility
	if newModel != a.current {
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

// keyMsgToNavigationKey converts a tea.KeyMsg to a NavigationKey if it matches
func (a *App) keyMsgToNavigationKey(keyMsg tea.KeyMsg) keymap.NavigationKey {
	switch keyMsg.String() {
	case "b":
		return keymap.NavigationKeyBack
	case "B":
		return keymap.NavigationKeyBulk
	case "n":
		return keymap.NavigationKeyNew
	case "r":
		return keymap.NavigationKeyRefresh
	case "s":
		return keymap.NavigationKeyStatus
	case "?":
		return keymap.NavigationKeyHelp
	case "q":
		return keymap.NavigationKeyQuit
	default:
		return ""
	}
}

// navigationKeyToMessage converts a NavigationKey to the appropriate navigation message
func (a *App) navigationKeyToMessage(navKey keymap.NavigationKey) messages.NavigationMsg {
	switch navKey {
	case keymap.NavigationKeyBack:
		return messages.NavigateBackMsg{}
	case keymap.NavigationKeyBulk:
		return messages.NavigateToBulkMsg{}
	default:
		// For other keys (n, r, s, ?, q), let the view handle them
		// These are not global navigation keys that should be intercepted
		return nil
	}
}
