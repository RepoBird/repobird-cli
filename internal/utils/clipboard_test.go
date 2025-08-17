// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import (
	"testing"
)

func TestInitClipboard(t *testing.T) {
	// Reset state for testing
	clipboardInitialized = false
	cgoAvailable = false

	err := InitClipboard()
	if err != nil {
		t.Errorf("InitClipboard() returned error: %v", err)
	}

	if !clipboardInitialized {
		t.Error("InitClipboard() did not set clipboardInitialized to true")
	}
}

func TestWriteToClipboard(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantErr bool
	}{
		{
			name:    "simple text",
			text:    "Hello, World!",
			wantErr: false,
		},
		{
			name:    "text with newlines",
			text:    "Line 1\nLine 2\nLine 3",
			wantErr: false,
		},
		{
			name:    "empty text",
			text:    "",
			wantErr: false,
		},
		{
			name:    "text with special characters",
			text:    "Special: !@#$%^&*()_+-=[]{}|;':\",./<>?",
			wantErr: false,
		},
		{
			name:    "unicode text",
			text:    "Unicode: 📋 ✅ ❌ 🔄",
			wantErr: false,
		},
	}

	// Initialize clipboard once for all tests
	_ = InitClipboard()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteToClipboard(tt.text)

			// In CI environments, clipboard operations might fail
			// We allow failures but test that the function doesn't panic
			if err != nil && !tt.wantErr {
				t.Logf("WriteToClipboard() error (may be expected in CI): %v", err)
			}
		})
	}
}

func TestWriteToClipboardFallback(t *testing.T) {
	tests := []struct {
		name string
		text string
	}{
		{
			name: "fallback with simple text",
			text: "Fallback test",
		},
		{
			name: "fallback with multiline",
			text: "Line 1\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the fallback directly
			err := writeToClipboardFallback(tt.text)

			// In CI environments or systems without clipboard tools,
			// this might fail. We just ensure it doesn't panic.
			if err != nil {
				t.Logf("writeToClipboardFallback() error (may be expected): %v", err)
			}
		})
	}
}

// TestClipboardWithoutCGO tests that clipboard operations don't panic
// when CGO is not available (simulated by using the fallback directly)
func TestClipboardWithoutCGO(t *testing.T) {
	// Force use of fallback
	clipboardInitialized = true
	cgoAvailable = false

	testText := "Test without CGO"
	err := WriteToClipboard(testText)

	// We don't expect this to succeed in all environments,
	// but it should not panic
	if err != nil {
		t.Logf("WriteToClipboard() without CGO error (may be expected): %v", err)
	}
}
