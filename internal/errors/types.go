// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later


package errors

import (
	"errors"
	"fmt"
	"net"
	"net/url"
)

// NoAPIKeyError returns a consistent error message for missing API key
func NoAPIKeyError() error {
	return fmt.Errorf(`API key not configured. You have 3 options:

A) Run 'repobird login' for interactive setup (recommended)
B) Run 'repobird config set api-key YOUR_KEY'  
C) Set REPOBIRD_API_KEY environment variable in your shell config (e.g., ~/.bashrc, ~/.zshrc)

Get your API key at: https://repobird.ai/dashboard/user-profile/api-keys`)
}

type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeAPI
	ErrorTypeNetwork
	ErrorTypeAuth
	ErrorTypeQuota
	ErrorTypeValidation
	ErrorTypeRateLimit
	ErrorTypeTimeout
	ErrorTypeNotFound
)

type APIError struct {
	StatusCode int
	Status     string
	Message    string
	ErrorType  ErrorType
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s (status %d)", e.Message, e.StatusCode)
	}
	return fmt.Sprintf("API error: %s (status %d)", e.Status, e.StatusCode)
}

func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	return e.StatusCode == t.StatusCode || e.ErrorType == t.ErrorType
}

type NetworkError struct {
	Err       error
	Operation string
	URL       string
}

func (e *NetworkError) Error() string {
	if e.Operation != "" {
		return fmt.Sprintf("network error during %s: %v", e.Operation, e.Err)
	}
	return fmt.Sprintf("network error: %v", e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

type AuthError struct {
	Message string
	Reason  string
}

func (e *AuthError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("authentication failed: %s (%s)", e.Message, e.Reason)
	}
	return fmt.Sprintf("authentication failed: %s", e.Message)
}

type QuotaError struct {
	Tier          string
	Limit         int
	Used          int
	RemainingRuns int
	UpgradeURL    string
}

func (e *QuotaError) Error() string {
	if e.UpgradeURL != "" {
		return fmt.Sprintf("no runs remaining (Tier: %s, Limit: %d/month). Upgrade at: %s",
			e.Tier, e.Limit, e.UpgradeURL)
	}
	return fmt.Sprintf("quota exceeded: %d of %d runs used (Tier: %s)", e.Used, e.Limit, e.Tier)
}

type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error for field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

type RateLimitError struct {
	RetryAfter string
	Limit      int
	Reset      string
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("rate limit exceeded. Please wait %s before retrying", e.RetryAfter)
	}
	return "rate limit exceeded"
}

var (
	ErrNoAPIKey          = &AuthError{Message: "No API key configured", Reason: "missing_api_key"}
	ErrInvalidAPIKey     = &AuthError{Message: "Invalid API key", Reason: "invalid_api_key"}
	ErrExpiredAPIKey     = &AuthError{Message: "API key has expired", Reason: "expired_api_key"}
	ErrServerUnavailable = &APIError{StatusCode: 503, Status: "Service Unavailable", ErrorType: ErrorTypeAPI}
	ErrTimeout           = &NetworkError{Err: errors.New("request timeout"), Operation: "API request"}
)

func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check for specific retryable error types
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 429, 500, 502, 503, 504:
			return true
		case 408:
			return true
		}
		return false
	}

	// Check for network errors
	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return true
	}

	// Check for standard net errors
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true
		}
		if urlErr.Temporary() {
			return true
		}
	}

	// Check for net.Error
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Temporary() || ne.Timeout()
	}

	// Rate limit errors are retryable
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	return false
}

func IsTemporary(err error) bool {
	if err == nil {
		return false
	}

	// Check for temporary network errors
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Temporary()
	}

	// Check for retryable API errors
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode >= 500 && apiErr.StatusCode < 600
	}

	return IsRetryable(err)
}

func IsQuotaExceeded(err error) bool {
	if err == nil {
		return false
	}

	var quotaErr *QuotaError
	return errors.As(err, &quotaErr)
}

func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	var authErr *AuthError
	if errors.As(err, &authErr) {
		return true
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 401 || apiErr.StatusCode == 403
	}

	return false
}

func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}

	var ne net.Error
	return errors.As(err, &ne)
}

func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == 404
	}

	return false
}
