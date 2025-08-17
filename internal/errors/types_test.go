// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package errors

import (
	"errors"
	"net/url"
	"testing"
)

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name: "with custom message",
			err: &APIError{
				StatusCode: 400,
				Status:     "Bad Request",
				Message:    "Invalid input",
				ErrorType:  ErrorTypeValidation,
			},
			expected: "Invalid input (status 400)",
		},
		{
			name: "without custom message",
			err: &APIError{
				StatusCode: 500,
				Status:     "Internal Server Error",
				ErrorType:  ErrorTypeAPI,
			},
			expected: "API error: Internal Server Error (status 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("APIError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNetworkError_Error(t *testing.T) {
	baseErr := errors.New("connection refused")

	tests := []struct {
		name     string
		err      *NetworkError
		expected string
	}{
		{
			name: "with operation",
			err: &NetworkError{
				Err:       baseErr,
				Operation: "POST /api/v1/runs",
				URL:       "https://repobird.ai/api/v1/runs",
			},
			expected: "network error during POST /api/v1/runs: connection refused",
		},
		{
			name: "without operation",
			err: &NetworkError{
				Err: baseErr,
				URL: "https://repobird.ai",
			},
			expected: "network error: connection refused",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("NetworkError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestQuotaError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *QuotaError
		expected string
	}{
		{
			name: "with upgrade URL",
			err: &QuotaError{
				Tier:          "Free",
				Limit:         10,
				RemainingRuns: 0,
				UpgradeURL:    "https://repobird.ai/dashboard",
			},
			expected: "no runs remaining (Tier: Free, Limit: 10/month). Upgrade at: https://repobird.ai/dashboard",
		},
		{
			name: "without upgrade URL",
			err: &QuotaError{
				Tier:  "Pro",
				Limit: 100,
				Used:  100,
			},
			expected: "quota exceeded: 100 of 100 runs used (Tier: Pro)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("QuotaError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "retryable API error (503)",
			err: &APIError{
				StatusCode: 503,
				ErrorType:  ErrorTypeAPI,
			},
			expected: true,
		},
		{
			name: "retryable API error (429)",
			err: &APIError{
				StatusCode: 429,
				ErrorType:  ErrorTypeRateLimit,
			},
			expected: true,
		},
		{
			name: "non-retryable API error (401)",
			err: &APIError{
				StatusCode: 401,
				ErrorType:  ErrorTypeAuth,
			},
			expected: false,
		},
		{
			name: "network error",
			err: &NetworkError{
				Err: errors.New("connection refused"),
			},
			expected: true,
		},
		{
			name: "rate limit error",
			err: &RateLimitError{
				RetryAfter: "60s",
			},
			expected: true,
		},
		{
			name:     "generic error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsTemporary(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "server error (500)",
			err: &APIError{
				StatusCode: 500,
				ErrorType:  ErrorTypeAPI,
			},
			expected: true,
		},
		{
			name: "client error (400)",
			err: &APIError{
				StatusCode: 400,
				ErrorType:  ErrorTypeValidation,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTemporary(tt.err); got != tt.expected {
				t.Errorf("IsTemporary() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsQuotaExceeded(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "quota error",
			err: &QuotaError{
				Tier: "Free",
			},
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsQuotaExceeded(tt.err); got != tt.expected {
				t.Errorf("IsQuotaExceeded() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "auth error",
			err: &AuthError{
				Message: "Invalid API key",
				Reason:  "invalid_key",
			},
			expected: true,
		},
		{
			name: "API error 401",
			err: &APIError{
				StatusCode: 401,
				ErrorType:  ErrorTypeAuth,
			},
			expected: true,
		},
		{
			name: "API error 403",
			err: &APIError{
				StatusCode: 403,
				ErrorType:  ErrorTypeAuth,
			},
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "network error",
			err: &NetworkError{
				Err: errors.New("connection failed"),
			},
			expected: true,
		},
		{
			name: "url error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: errors.New("timeout"),
			},
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkError(tt.err); got != tt.expected {
				t.Errorf("IsNetworkError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name: "404 API error",
			err: &APIError{
				StatusCode: 404,
				ErrorType:  ErrorTypeNotFound,
			},
			expected: true,
		},
		{
			name: "other API error",
			err: &APIError{
				StatusCode: 500,
				ErrorType:  ErrorTypeAPI,
			},
			expected: false,
		},
		{
			name:     "other error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.expected {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// mockNetError implements net.Error for testing
type mockNetError struct {
	msg       string
	timeout   bool
	temporary bool
}

func (e mockNetError) Error() string   { return e.msg }
func (e mockNetError) Timeout() bool   { return e.timeout }
func (e mockNetError) Temporary() bool { return e.temporary }

func TestIsRetryableWithNetError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "temporary net error",
			err:      mockNetError{msg: "temporary failure", temporary: true},
			expected: true,
		},
		{
			name:     "timeout net error",
			err:      mockNetError{msg: "timeout", timeout: true},
			expected: true,
		},
		{
			name:     "permanent net error",
			err:      mockNetError{msg: "permanent failure", timeout: false, temporary: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}
