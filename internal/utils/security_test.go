package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "empty key",
			apiKey:   "",
			expected: "",
		},
		{
			name:     "very short key",
			apiKey:   "ab",
			expected: "**",
		},
		{
			name:     "short key",
			apiKey:   "abcd",
			expected: "ab**",
		},
		{
			name:     "medium key",
			apiKey:   "abcdefgh",
			expected: "ab******",
		},
		{
			name:     "normal key",
			apiKey:   "sk-1234567890abcdefghijklmnop",
			expected: "sk-1*************************",
		},
		{
			name:     "long key",
			apiKey:   "verylongapikeythatshouldbemaskdproperly123456789",
			expected: "very********************************************",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.apiKey, result, tt.expected)
			}
		})
	}
}

func TestMaskSensitiveString(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		showChars int
		expected  string
	}{
		{
			name:      "empty string",
			value:     "",
			showChars: 4,
			expected:  "",
		},
		{
			name:      "shorter than show chars",
			value:     "ab",
			showChars: 4,
			expected:  "**",
		},
		{
			name:      "exact show chars",
			value:     "abcd",
			showChars: 4,
			expected:  "****",
		},
		{
			name:      "longer than show chars",
			value:     "abcdefghij",
			showChars: 4,
			expected:  "abcd******",
		},
		{
			name:      "show 2 chars",
			value:     "sensitive",
			showChars: 2,
			expected:  "se*******",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveString(tt.value, tt.showChars)
			if result != tt.expected {
				t.Errorf("MaskSensitiveString(%q, %d) = %q, want %q",
					tt.value, tt.showChars, result, tt.expected)
			}
		})
	}
}

func TestRedactAuthHeader(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
		expected    string
	}{
		{
			name:        "empty header",
			headerValue: "",
			expected:    "",
		},
		{
			name:        "bearer token",
			headerValue: "Bearer sk-1234567890abcdefghijklmnop",
			expected:    "Bearer sk-1*************************",
		},
		{
			name:        "basic auth",
			headerValue: "Basic dXNlcjpwYXNzd29yZA==",
			expected:    "Basic dXNl****************",
		},
		{
			name:        "custom auth",
			headerValue: "CustomAuth mysecrettoken123",
			expected:    "CustomAuth myse************",
		},
		{
			name:        "no space",
			headerValue: "Tokenwithoutspace",
			expected:    "Toke*************",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactAuthHeader(tt.headerValue)
			if result != tt.expected {
				t.Errorf("RedactAuthHeader(%q) = %q, want %q",
					tt.headerValue, result, tt.expected)
			}
		})
	}
}

func TestSanitizeErrorMessage(t *testing.T) {
	apiKey := "sk-secretapikey123456789"
	maskedKey := MaskAPIKey(apiKey)

	tests := []struct {
		name     string
		err      error
		apiKey   string
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			apiKey:   apiKey,
			expected: "",
		},
		{
			name:     "error without api key",
			err:      errors.New("connection failed"),
			apiKey:   apiKey,
			expected: "connection failed",
		},
		{
			name:     "error with api key",
			err:      errors.New("authentication failed with key: " + apiKey),
			apiKey:   apiKey,
			expected: "authentication failed with key: " + maskedKey,
		},
		{
			name:     "error with bearer token",
			err:      errors.New("Invalid Bearer " + apiKey + " in request"),
			apiKey:   apiKey,
			expected: "Invalid Bearer " + maskedKey + " in request",
		},
		{
			name:     "multiple occurrences",
			err:      errors.New("key " + apiKey + " failed, retry with " + apiKey),
			apiKey:   apiKey,
			expected: "key " + maskedKey + " failed, retry with " + maskedKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeErrorMessage(tt.err, tt.apiKey)
			if result != tt.expected {
				t.Errorf("SanitizeErrorMessage(%v, %q) = %q, want %q",
					tt.err, tt.apiKey, result, tt.expected)
			}
		})
	}
}

func TestClearString(t *testing.T) {
	secret := "mysecretvalue"
	ClearString(&secret)

	if secret != "" {
		t.Errorf("ClearString failed to clear string, got %q", secret)
	}

	// Test with nil pointer
	var nilStr *string
	ClearString(nilStr) // Should not panic
}

func TestClearByteSlice(t *testing.T) {
	secret := []byte("mysecretbytes")
	original := make([]byte, len(secret))
	copy(original, secret)

	ClearByteSlice(secret)

	for i, b := range secret {
		if b != 0 {
			t.Errorf("ClearByteSlice failed at index %d: got %d, want 0", i, b)
		}
	}

	// Test with nil slice
	var nilSlice []byte
	ClearByteSlice(nilSlice) // Should not panic
}

func TestValidateAPIKeyFormat(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "empty key",
			apiKey:    "",
			wantError: true,
			errorMsg:  "API key is required",
		},
		{
			name:      "too short",
			apiKey:    "short",
			wantError: true,
			errorMsg:  "invalid API key format",
		},
		{
			name:      "valid key",
			apiKey:    "sk-1234567890abcdefghijklmnop",
			wantError: false,
		},
		{
			name:      "key with spaces",
			apiKey:    "sk 1234567890 abcdef",
			wantError: true,
			errorMsg:  "API key contains invalid characters",
		},
		{
			name:      "bearer prefix valid",
			apiKey:    "Bearer sk-1234567890abcdefghijklmnop",
			wantError: false,
		},
		{
			name:      "bearer prefix too short",
			apiKey:    "Bearer short",
			wantError: true,
			errorMsg:  "invalid API key format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKeyFormat(tt.apiKey)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateAPIKeyFormat(%q) expected error, got nil", tt.apiKey)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("ValidateAPIKeyFormat(%q) error = %v, want containing %q",
						tt.apiKey, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAPIKeyFormat(%q) unexpected error: %v", tt.apiKey, err)
				}
			}
		})
	}
}
