// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package utils

import (
	"fmt"
	"strings"
)

// MaskAPIKey masks an API key for secure display, showing only the first 4 characters
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	if len(apiKey) <= 8 {
		// Very short key, mask everything except first 2 chars
		if len(apiKey) <= 2 {
			return strings.Repeat("*", len(apiKey))
		}
		return apiKey[:2] + strings.Repeat("*", len(apiKey)-2)
	}

	// Show first 4 characters, mask the rest
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-4)
}

// MaskSensitiveString masks any sensitive string for display
func MaskSensitiveString(value string, showChars int) string {
	if value == "" {
		return ""
	}

	if len(value) <= showChars {
		return strings.Repeat("*", len(value))
	}

	return value[:showChars] + strings.Repeat("*", len(value)-showChars)
}

// RedactAuthHeader redacts the Authorization header value for logging
func RedactAuthHeader(headerValue string) string {
	if headerValue == "" {
		return ""
	}

	// Check if it's a Bearer token
	if strings.HasPrefix(headerValue, "Bearer ") {
		token := headerValue[7:]
		return "Bearer " + MaskAPIKey(token)
	}

	// For other auth types, mask most of it
	parts := strings.SplitN(headerValue, " ", 2)
	if len(parts) == 2 {
		return parts[0] + " " + MaskAPIKey(parts[1])
	}

	return MaskAPIKey(headerValue)
}

// SanitizeErrorMessage removes sensitive information from error messages
func SanitizeErrorMessage(err error, apiKey string) string {
	if err == nil {
		return ""
	}

	errMsg := err.Error()

	// Replace API key if it appears in the error
	if apiKey != "" && strings.Contains(errMsg, apiKey) {
		errMsg = strings.ReplaceAll(errMsg, apiKey, MaskAPIKey(apiKey))
	}

	// Look for common patterns that might contain sensitive data
	// Bearer tokens
	if idx := strings.Index(errMsg, "Bearer "); idx >= 0 {
		endIdx := idx + 7
		for endIdx < len(errMsg) && errMsg[endIdx] != ' ' && errMsg[endIdx] != '"' {
			endIdx++
		}
		if endIdx > idx+7 {
			token := errMsg[idx+7 : endIdx]
			errMsg = strings.ReplaceAll(errMsg, "Bearer "+token, "Bearer "+MaskAPIKey(token))
		}
	}

	return errMsg
}

// ClearString attempts to clear a string from memory (best effort)
// Note: In Go, this is not guaranteed due to garbage collection and string immutability
func ClearString(s *string) {
	if s == nil {
		return
	}

	// Overwrite the string with zeros
	// This is best effort - Go's GC may have copies elsewhere
	*s = ""

	// Force a new allocation to help clear the old memory
	*s = strings.Repeat("\x00", len(*s))
	*s = ""
}

// ClearByteSlice overwrites a byte slice with zeros
func ClearByteSlice(b []byte) {
	if b == nil {
		return
	}

	for i := range b {
		b[i] = 0
	}
}

// ValidateAPIKeyFormat performs basic validation on API key format
// without revealing specifics about what's wrong
func ValidateAPIKeyFormat(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key is required")
	}

	// Basic length check (adjust based on your API key format)
	if len(apiKey) < 10 {
		return fmt.Errorf("invalid API key format")
	}

	// Check for common mistakes
	if strings.Contains(apiKey, " ") && !strings.HasPrefix(apiKey, "Bearer ") {
		return fmt.Errorf("API key contains invalid characters")
	}

	if strings.HasPrefix(apiKey, "Bearer ") {
		// If they included "Bearer " prefix, extract the actual key
		actualKey := apiKey[7:]
		if len(actualKey) < 10 {
			return fmt.Errorf("invalid API key format")
		}
	}

	return nil
}
