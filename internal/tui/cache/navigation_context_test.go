// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package cache

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNavigationContext(t *testing.T) {
	cache := NewSimpleCache()

	t.Run("Set and Get Context", func(t *testing.T) {
		cache.SetContext("key1", "value1")
		cache.SetContext("key2", 42)
		cache.SetContext("key3", []string{"a", "b", "c"})

		assert.Equal(t, "value1", cache.GetContext("key1"))
		assert.Equal(t, 42, cache.GetContext("key2"))
		assert.Equal(t, []string{"a", "b", "c"}, cache.GetContext("key3"))
	})

	t.Run("Get non-existent context", func(t *testing.T) {
		value := cache.GetContext("nonexistent")
		assert.Nil(t, value)
	})

	t.Run("Clear specific context", func(t *testing.T) {
		cache.SetContext("temp", "temporary")
		assert.Equal(t, "temporary", cache.GetContext("temp"))

		cache.ClearContext("temp")
		assert.Nil(t, cache.GetContext("temp"))
	})

	t.Run("Update existing context", func(t *testing.T) {
		cache.SetContext("mutable", "initial")
		assert.Equal(t, "initial", cache.GetContext("mutable"))

		cache.SetContext("mutable", "updated")
		assert.Equal(t, "updated", cache.GetContext("mutable"))
	})
}

func TestNavigationSpecificContext(t *testing.T) {
	cache := NewSimpleCache()

	t.Run("Set and Get Navigation Context", func(t *testing.T) {
		cache.SetNavigationContext("selected_repo", "org/repo")
		cache.SetNavigationContext("selected_index", 5)

		assert.Equal(t, "org/repo", cache.GetNavigationContext("selected_repo"))
		assert.Equal(t, 5, cache.GetNavigationContext("selected_index"))

		// Verify they're stored with nav: prefix internally
		assert.Equal(t, "org/repo", cache.GetContext("nav:selected_repo"))
		assert.Equal(t, 5, cache.GetContext("nav:selected_index"))
	})

	t.Run("Clear All Navigation Context", func(t *testing.T) {
		// Set mix of navigation and regular context
		cache.SetContext("regular1", "value1")
		cache.SetContext("regular2", "value2")
		cache.SetNavigationContext("nav1", "navvalue1")
		cache.SetNavigationContext("nav2", "navvalue2")

		// Clear only navigation context
		cache.ClearAllNavigationContext()

		// Regular context should remain
		assert.Equal(t, "value1", cache.GetContext("regular1"))
		assert.Equal(t, "value2", cache.GetContext("regular2"))

		// Navigation context should be cleared
		assert.Nil(t, cache.GetNavigationContext("nav1"))
		assert.Nil(t, cache.GetNavigationContext("nav2"))
	})

	t.Run("Navigation context isolation", func(t *testing.T) {
		// Set context with and without nav prefix
		cache.SetContext("item", "regular")
		cache.SetNavigationContext("item", "navigation")

		// They should be separate
		assert.Equal(t, "regular", cache.GetContext("item"))
		assert.Equal(t, "navigation", cache.GetNavigationContext("item"))
		assert.Equal(t, "navigation", cache.GetContext("nav:item"))
	})
}

func TestContextThreadSafety(t *testing.T) {
	cache := NewSimpleCache()

	t.Run("Concurrent Set and Get", func(t *testing.T) {
		var wg sync.WaitGroup
		iterations := 100

		// Concurrent writers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					key := "key" + string(rune('A'+id))
					cache.SetContext(key, j)
				}
			}(i)
		}

		// Concurrent readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					key := "key" + string(rune('A'+id))
					_ = cache.GetContext(key)
				}
			}(i)
		}

		wg.Wait()

		// Verify final state
		for i := 0; i < 10; i++ {
			key := "key" + string(rune('A'+i))
			value := cache.GetContext(key)
			assert.NotNil(t, value)
			// Should be one of the iteration values
			intValue, ok := value.(int)
			assert.True(t, ok)
			assert.GreaterOrEqual(t, intValue, 0)
			assert.Less(t, intValue, iterations)
		}
	})

	t.Run("Concurrent Navigation Context Operations", func(t *testing.T) {
		var wg sync.WaitGroup

		// Writer goroutines
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					cache.SetNavigationContext("nav"+string(rune('0'+id)), j)
				}
			}(i)
		}

		// Reader goroutines
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					_ = cache.GetNavigationContext("nav" + string(rune('0'+id)))
				}
			}(i)
		}

		// Clear goroutine
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				cache.ClearAllNavigationContext()
			}
		}()

		wg.Wait()

		// No assertions on final state since clear may have run last
		// Just verify no panic or deadlock occurred
	})
}

func TestContextDataTypes(t *testing.T) {
	cache := NewSimpleCache()

	// Test various data types
	testCases := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"String", "string", "test string"},
		{"Int", "int", 42},
		{"Float", "float", 3.14},
		{"Bool", "bool", true},
		{"Slice", "slice", []int{1, 2, 3}},
		{"Map", "map", map[string]int{"a": 1, "b": 2}},
		{"Struct", "struct", struct{ Name string }{"test"}},
		{"Nil", "nil", nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cache.SetContext(tc.key, tc.value)
			retrieved := cache.GetContext(tc.key)
			assert.Equal(t, tc.value, retrieved)
		})
	}
}

func TestNavigationContextPrefixing(t *testing.T) {
	cache := NewSimpleCache()

	t.Run("Verify nav: prefix", func(t *testing.T) {
		// Set navigation context
		cache.SetNavigationContext("test", "value")

		// Should be retrievable with prefix
		assert.Equal(t, "value", cache.GetContext("nav:test"))

		// Should not be retrievable without prefix
		assert.Nil(t, cache.GetContext("test"))
	})

	t.Run("ClearAllNavigationContext only clears nav: prefixed", func(t *testing.T) {
		// Set various keys
		cache.SetContext("nav:auto", "should be cleared")
		cache.SetContext("navigate", "should not be cleared")
		cache.SetContext("naval", "should not be cleared")
		cache.SetNavigationContext("manual", "should be cleared")

		cache.ClearAllNavigationContext()

		// nav: prefixed should be cleared
		assert.Nil(t, cache.GetContext("nav:auto"))
		assert.Nil(t, cache.GetNavigationContext("manual"))

		// Non nav: prefixed should remain
		assert.Equal(t, "should not be cleared", cache.GetContext("navigate"))
		assert.Equal(t, "should not be cleared", cache.GetContext("naval"))
	})
}

func TestContextMemoryManagement(t *testing.T) {
	cache := NewSimpleCache()

	t.Run("Large number of context entries", func(t *testing.T) {
		// Add many entries
		for i := 0; i < 1000; i++ {
			key := "key" + string(rune(i))
			cache.SetContext(key, i)
		}

		// Verify a sample
		assert.Equal(t, 0, cache.GetContext("key"+string(rune(0))))
		assert.Equal(t, 500, cache.GetContext("key"+string(rune(500))))
		assert.Equal(t, 999, cache.GetContext("key"+string(rune(999))))
	})

	t.Run("Clear navigation doesn't affect regular context", func(t *testing.T) {
		// Set many navigation and regular entries
		for i := 0; i < 100; i++ {
			cache.SetNavigationContext("nav"+string(rune(i)), i)
			cache.SetContext("reg"+string(rune(i)), i*2)
		}

		// Clear navigation
		cache.ClearAllNavigationContext()

		// Regular entries should remain
		for i := 0; i < 100; i++ {
			assert.Nil(t, cache.GetNavigationContext("nav"+string(rune(i))))
			assert.Equal(t, i*2, cache.GetContext("reg"+string(rune(i))))
		}
	})
}

func TestContextEdgeCases(t *testing.T) {
	cache := NewSimpleCache()

	t.Run("Empty string key", func(t *testing.T) {
		cache.SetContext("", "empty key")
		assert.Equal(t, "empty key", cache.GetContext(""))

		cache.ClearContext("")
		assert.Nil(t, cache.GetContext(""))
	})

	t.Run("Special characters in key", func(t *testing.T) {
		specialKeys := []string{
			"key with spaces",
			"key/with/slashes",
			"key:with:colons",
			"key.with.dots",
			"key-with-dashes",
			"key_with_underscores",
			"key@with#special$chars",
		}

		for _, key := range specialKeys {
			cache.SetContext(key, "value")
			assert.Equal(t, "value", cache.GetContext(key))
		}
	})

	t.Run("Overwrite existing value", func(t *testing.T) {
		cache.SetContext("key", "initial")
		cache.SetContext("key", "overwritten")
		assert.Equal(t, "overwritten", cache.GetContext("key"))
	})

	t.Run("Clear non-existent key", func(t *testing.T) {
		// Should not panic
		cache.ClearContext("nonexistent")
	})

	t.Run("Multiple ClearAllNavigationContext calls", func(t *testing.T) {
		cache.SetNavigationContext("test", "value")

		// Multiple clears should not panic
		cache.ClearAllNavigationContext()
		cache.ClearAllNavigationContext()
		cache.ClearAllNavigationContext()

		assert.Nil(t, cache.GetNavigationContext("test"))
	})
}
