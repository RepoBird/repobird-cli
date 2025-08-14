package keymap

import tea "github.com/charmbracelet/bubbletea"

// CoreViewKeymap is the enhanced interface that views can implement to control key behavior
type CoreViewKeymap interface {
	// IsKeyDisabled returns true if the given key should be ignored for this view
	IsKeyDisabled(keyString string) bool

	// HandleKey allows views to provide custom handling for specific keys
	// Returns (handled, model, cmd) - if handled=true, the result is used instead of default processing
	HandleKey(keyMsg tea.KeyMsg) (handled bool, model tea.Model, cmd tea.Cmd)
}

// KeyAction represents what should happen when a key is pressed
type KeyAction int

const (
	// Navigation actions
	ActionNavigateBack KeyAction = iota
	ActionNavigateBulk
	ActionNavigateNew
	ActionNavigateRefresh
	ActionNavigateQuit
	ActionNavigateHelp
	ActionNavigateToDashboard // Navigate directly to dashboard

	// View actions
	ActionViewSpecific // Let the view handle it
	ActionIgnore       // Completely ignore the key

	// Global actions
	ActionGlobalQuit
	ActionGlobalHelp
)

// KeyMapping defines how a key should be processed
type KeyMapping struct {
	Key    string
	Action KeyAction
	Help   string
}

// CoreKeyRegistry maintains the central registry of all keys and their default actions
type CoreKeyRegistry struct {
	mappings map[string]KeyMapping
}

// NewCoreKeyRegistry creates a new key registry with default mappings
func NewCoreKeyRegistry() *CoreKeyRegistry {
	registry := &CoreKeyRegistry{
		mappings: make(map[string]KeyMapping),
	}

	// Register default key mappings
	// Primary navigation: h/H for dashboard (vim/ranger style), q for dashboard/quit
	registry.Register("h", ActionNavigateToDashboard, "dashboard")
	registry.Register("H", ActionNavigateToDashboard, "dashboard") // Same as h
	registry.Register("B", ActionNavigateBulk, "bulk operations")
	registry.Register("n", ActionNavigateNew, "new run")
	registry.Register("r", ActionNavigateRefresh, "refresh")
	registry.Register("q", ActionNavigateToDashboard, "dashboard") // Goes to dashboard from child views
	registry.Register("Q", ActionGlobalQuit, "force quit")
	registry.Register("?", ActionNavigateHelp, "help")
	registry.Register("ctrl+c", ActionGlobalQuit, "force quit")
	registry.Register("esc", ActionViewSpecific, "cancel modal/overlay") // Only for modals, not navigation
	// Removed: backspace and b for back navigation

	// View-specific keys that should be handled by views
	registry.Register("s", ActionViewSpecific, "status info")
	registry.Register("f", ActionViewSpecific, "search/filter")
	registry.Register("/", ActionViewSpecific, "search")
	registry.Register("enter", ActionViewSpecific, "select/enter")
	registry.Register("tab", ActionViewSpecific, "next field")
	registry.Register("shift+tab", ActionViewSpecific, "previous field")
	registry.Register("backspace", ActionViewSpecific, "typing")
	registry.Register("b", ActionViewSpecific, "view specific") // Let views handle 'b' (dashboard uses for bulk)

	// Navigation keys (for column/list navigation within views)
	// Note: 'h' is registered above as back navigation
	registry.Register("j", ActionViewSpecific, "down")
	registry.Register("k", ActionViewSpecific, "up")
	registry.Register("l", ActionViewSpecific, "right")
	registry.Register("left", ActionViewSpecific, "left")
	registry.Register("right", ActionViewSpecific, "right")
	registry.Register("up", ActionViewSpecific, "up")
	registry.Register("down", ActionViewSpecific, "down")

	return registry
}

// Register adds a key mapping to the registry
func (r *CoreKeyRegistry) Register(key string, action KeyAction, help string) {
	r.mappings[key] = KeyMapping{
		Key:    key,
		Action: action,
		Help:   help,
	}
}

// GetAction returns the default action for a key, or ActionViewSpecific if not found
func (r *CoreKeyRegistry) GetAction(key string) KeyAction {
	if mapping, exists := r.mappings[key]; exists {
		return mapping.Action
	}
	return ActionViewSpecific
}

// GetMapping returns the full mapping for a key
func (r *CoreKeyRegistry) GetMapping(key string) (KeyMapping, bool) {
	mapping, exists := r.mappings[key]
	return mapping, exists
}

// GetAllMappings returns all registered key mappings
func (r *CoreKeyRegistry) GetAllMappings() map[string]KeyMapping {
	result := make(map[string]KeyMapping)
	for k, v := range r.mappings {
		result[k] = v
	}
	return result
}

// IsNavigationAction returns true if the action is a navigation action
func IsNavigationAction(action KeyAction) bool {
	switch action {
	case ActionNavigateBack, ActionNavigateBulk, ActionNavigateNew,
		ActionNavigateRefresh, ActionNavigateQuit, ActionNavigateHelp,
		ActionNavigateToDashboard:
		return true
	default:
		return false
	}
}

// IsGlobalAction returns true if the action should be handled globally regardless of view state
func IsGlobalAction(action KeyAction) bool {
	switch action {
	case ActionGlobalQuit, ActionGlobalHelp:
		return true
	default:
		return false
	}
}
