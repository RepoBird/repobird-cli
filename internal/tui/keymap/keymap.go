// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package keymap

// NavigationKey represents a navigation key that can be enabled/disabled per view
type NavigationKey string

const (
	// NavigationKeyBack represents the 'b' key for back navigation
	NavigationKeyBack NavigationKey = "b"
	// NavigationKeyBulk represents the 'B' key for bulk operations
	NavigationKeyBulk NavigationKey = "B"
	// NavigationKeyNew represents the 'n' key for creating new items
	NavigationKeyNew NavigationKey = "n"
	// NavigationKeyRefresh represents the 'r' key for refreshing
	NavigationKeyRefresh NavigationKey = "r"
	// NavigationKeyStatus represents the 's' key for status info
	NavigationKeyStatus NavigationKey = "s"
	// NavigationKeyHelp represents the '?' key for help
	NavigationKeyHelp NavigationKey = "?"
	// NavigationKeyQuit represents the 'q' key for quitting
	NavigationKeyQuit NavigationKey = "q"
)

// ViewKeymap defines which navigation keys are available for a view
type ViewKeymap interface {
	// IsNavigationKeyEnabled returns true if the given navigation key is enabled for this view
	IsNavigationKeyEnabled(key NavigationKey) bool
}

// DefaultKeymap provides default key mappings that most views can use
type DefaultKeymap struct {
	enabledKeys map[NavigationKey]bool
}

// NewDefaultKeymap creates a keymap with all navigation keys enabled by default
func NewDefaultKeymap() *DefaultKeymap {
	return &DefaultKeymap{
		enabledKeys: map[NavigationKey]bool{
			NavigationKeyBack:    true,
			NavigationKeyBulk:    true,
			NavigationKeyNew:     true,
			NavigationKeyRefresh: true,
			NavigationKeyStatus:  true,
			NavigationKeyHelp:    true,
			NavigationKeyQuit:    true,
		},
	}
}

// NewKeymapWithDisabled creates a keymap with specified keys disabled
func NewKeymapWithDisabled(disabledKeys ...NavigationKey) *DefaultKeymap {
	keymap := NewDefaultKeymap()
	for _, key := range disabledKeys {
		keymap.enabledKeys[key] = false
	}
	return keymap
}

// IsNavigationKeyEnabled implements ViewKeymap interface
func (k *DefaultKeymap) IsNavigationKeyEnabled(key NavigationKey) bool {
	enabled, exists := k.enabledKeys[key]
	return exists && enabled
}
