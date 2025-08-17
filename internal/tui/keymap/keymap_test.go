// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package keymap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultKeymap(t *testing.T) {
	t.Run("default keymap enables all keys", func(t *testing.T) {
		km := NewDefaultKeymap()

		// Test all defined navigation keys
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyBack))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyBulk))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyNew))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyRefresh))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyStatus))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyHelp))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyQuit))
	})
}

func TestKeymapWithDisabled(t *testing.T) {
	t.Run("keymap with disabled back key", func(t *testing.T) {
		km := NewKeymapWithDisabled(NavigationKeyBack)

		// Back key should be disabled
		assert.False(t, km.IsNavigationKeyEnabled(NavigationKeyBack))

		// Other keys should still be enabled
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyBulk))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyNew))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyRefresh))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyStatus))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyHelp))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyQuit))
	})

	t.Run("keymap with multiple disabled keys", func(t *testing.T) {
		km := NewKeymapWithDisabled(NavigationKeyBack, NavigationKeyBulk, NavigationKeyStatus)

		// Disabled keys should be false
		assert.False(t, km.IsNavigationKeyEnabled(NavigationKeyBack))
		assert.False(t, km.IsNavigationKeyEnabled(NavigationKeyBulk))
		assert.False(t, km.IsNavigationKeyEnabled(NavigationKeyStatus))

		// Enabled keys should be true
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyNew))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyRefresh))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyHelp))
		assert.True(t, km.IsNavigationKeyEnabled(NavigationKeyQuit))
	})
}

func TestNavigationKeyConstants(t *testing.T) {
	t.Run("navigation key constants have correct values", func(t *testing.T) {
		assert.Equal(t, NavigationKey("b"), NavigationKeyBack)
		assert.Equal(t, NavigationKey("B"), NavigationKeyBulk)
		assert.Equal(t, NavigationKey("n"), NavigationKeyNew)
		assert.Equal(t, NavigationKey("r"), NavigationKeyRefresh)
		assert.Equal(t, NavigationKey("s"), NavigationKeyStatus)
		assert.Equal(t, NavigationKey("?"), NavigationKeyHelp)
		assert.Equal(t, NavigationKey("q"), NavigationKeyQuit)
	})
}

// TestViewKeymapInterface tests that DefaultKeymap implements ViewKeymap
func TestViewKeymapInterface(t *testing.T) {
	t.Run("DefaultKeymap implements ViewKeymap interface", func(t *testing.T) {
		var km ViewKeymap = NewDefaultKeymap()

		// Should be able to call interface method
		enabled := km.IsNavigationKeyEnabled(NavigationKeyBack)
		assert.True(t, enabled)
	})
}
