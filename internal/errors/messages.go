// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/repobird/repobird-cli/internal/config"
)

// GetStatusMessages returns status messages with dynamic URLs
func GetStatusMessages() map[string]string {
	urls := config.GetURLs()
	return map[string]string{
		"NO_RUNS_REMAINING":   "You've used all your available runs. Upgrade your plan at " + urls.PricingURL,
		"REPO_NOT_FOUND":      "Repository not found or not connected. Please connect it at " + urls.ReposURL,
		"INVALID_API_KEY":     "Invalid API key. Get a new one at " + urls.SettingsURL,
		"RATE_LIMIT_EXCEEDED": "Rate limit exceeded. Please wait before retrying",
		"SERVER_ERROR":        "RepoBird servers are experiencing issues. Please try again later",
		"UNAUTHORIZED":        "You don't have permission to access this resource",
		"FORBIDDEN":           "Access to this resource is forbidden",
		"TIMEOUT":             "Request timed out. The operation may still be processing",
		"NETWORK_ERROR":       "Network connectivity issue. Please check your connection",
		"VALIDATION_ERROR":    "Invalid input provided. Please check your request",
		"QUOTA_EXCEEDED":      "You have exceeded your quota limits. Upgrade at " + urls.PricingURL,
	}
}

type APIErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message"`
	Status  string                 `json:"status"`
	Code    string                 `json:"code"`
	Details map[string]interface{} `json:"details"`
}

func ParseAPIError(statusCode int, body []byte) error {
	// Try to parse JSON error response
	var apiErr APIErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil {
		return createErrorFromAPIResponse(statusCode, apiErr)
	}

	// Fallback to raw body as error message
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = fmt.Sprintf("API request failed with status %d", statusCode)
	}

	return createErrorFromStatusCode(statusCode, message)
}

func createErrorFromAPIResponse(statusCode int, apiErr APIErrorResponse) error {
	// Check for specific error codes
	// Note: API may send error code in either "code" or "error" field
	errorCode := strings.ToUpper(apiErr.Code)
	if errorCode == "" {
		errorCode = strings.ToUpper(apiErr.Error)
	}

	switch errorCode {
	case "NO_RUNS_REMAINING":
		tier := ""
		limit := 0
		remaining := 0

		if apiErr.Details != nil {
			if t, ok := apiErr.Details["tier"].(string); ok {
				tier = t
			}
			if l, ok := apiErr.Details["limit"].(float64); ok {
				limit = int(l)
			}
			if r, ok := apiErr.Details["remaining"].(float64); ok {
				remaining = int(r)
			}
		}

		return &QuotaError{
			Tier:          tier,
			Limit:         limit,
			RemainingRuns: remaining,
			UpgradeURL:    config.GetPricingURL(),
		}

	case "INVALID_API_KEY", "UNAUTHORIZED":
		return &AuthError{
			Message: GetStatusMessages()["INVALID_API_KEY"],
			Reason:  apiErr.Code,
		}

	case "REPO_NOT_FOUND":
		message := GetStatusMessages()["REPO_NOT_FOUND"]
		if apiErr.Details != nil {
			if repo, ok := apiErr.Details["repository"].(string); ok {
				message = fmt.Sprintf("Repository '%s' not found or not connected. Connect it at: %s", repo, config.GetURLs().ReposURL)
			}
		}
		return &APIError{
			StatusCode: statusCode,
			Status:     apiErr.Status,
			Message:    message,
			ErrorType:  ErrorTypeNotFound,
		}

	case "BRANCH_NOT_FOUND":
		// Use the full error message from the API which includes branch name and repository
		// Example: "Target branch 'jdfsjdfksdf' does not exist in test-acc-254/youtube-music. Please create it first, or leave target empty to merge back to source branch."
		message := apiErr.Message
		if message == "" {
			message = "Branch not found in repository"
		}
		return &APIError{
			StatusCode: statusCode,
			Status:     apiErr.Status,
			Message:    message,
			ErrorType:  ErrorTypeNotFound,
		}

	case "RATE_LIMIT_EXCEEDED":
		retryAfter := ""
		if apiErr.Details != nil {
			if ra, ok := apiErr.Details["retry_after"].(string); ok {
				retryAfter = ra
			}
		}
		return &RateLimitError{
			RetryAfter: retryAfter,
		}

	case "VALIDATION_ERROR":
		field := ""
		if apiErr.Details != nil {
			if f, ok := apiErr.Details["field"].(string); ok {
				field = f
			}
		}
		return &ValidationError{
			Field:   field,
			Message: apiErr.Message,
		}
	}

	// Default to generic API error
	message := apiErr.Message
	if message == "" {
		message = apiErr.Error
	}

	return createErrorFromStatusCode(statusCode, message)
}

func createErrorFromStatusCode(statusCode int, message string) error {
	var errorType ErrorType

	switch statusCode {
	case 401:
		errorType = ErrorTypeAuth
		if message == "" {
			message = GetStatusMessages()["INVALID_API_KEY"]
		}
		return &AuthError{
			Message: message,
			Reason:  "http_401",
		}

	case 403:
		errorType = ErrorTypeAuth
		if message == "" {
			message = GetStatusMessages()["FORBIDDEN"]
		}
		return &AuthError{
			Message: message,
			Reason:  "http_403",
		}

	case 404:
		errorType = ErrorTypeNotFound
		if message == "" {
			message = "Resource not found"
		}

	case 429:
		errorType = ErrorTypeRateLimit
		if message == "" {
			message = GetStatusMessages()["RATE_LIMIT_EXCEEDED"]
		}
		// Return APIError with status code for consistency
		return &APIError{
			StatusCode: statusCode,
			Status:     "Too Many Requests",
			Message:    message,
			ErrorType:  errorType,
		}

	case 408:
		errorType = ErrorTypeTimeout
		if message == "" {
			message = GetStatusMessages()["TIMEOUT"]
		}

	case 422:
		errorType = ErrorTypeValidation
		if message == "" {
			message = GetStatusMessages()["VALIDATION_ERROR"]
		}
		return &ValidationError{
			Message: message,
		}

	case 500, 502, 503, 504:
		errorType = ErrorTypeAPI
		if message == "" {
			message = GetStatusMessages()["SERVER_ERROR"]
		}

	default:
		errorType = ErrorTypeUnknown
		if message == "" {
			message = fmt.Sprintf("Unexpected error (status %d)", statusCode)
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Status:     getHTTPStatusText(statusCode),
		Message:    message,
		ErrorType:  errorType,
	}
}

func getHTTPStatusText(code int) string {
	switch code {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 408:
		return "Request Timeout"
	case 422:
		return "Unprocessable Entity"
	case 429:
		return "Too Many Requests"
	case 500:
		return "Internal Server Error"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	case 504:
		return "Gateway Timeout"
	default:
		return fmt.Sprintf("HTTP %d", code)
	}
}

func FormatUserError(err error) string {
	if err == nil {
		return ""
	}

	// Check for quota error
	var quotaErr *QuotaError
	if errors.As(err, &quotaErr) {
		return quotaErr.Error()
	}

	// Check for auth error
	var authErr *AuthError
	if errors.As(err, &authErr) {
		return authErr.Error()
	}

	// Check for rate limit error
	var rateLimitErr *RateLimitError
	if errors.As(err, &rateLimitErr) {
		return rateLimitErr.Error()
	}

	// Check for validation error
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr.Error()
	}

	// Check for network error
	var netErr *NetworkError
	if errors.As(err, &netErr) {
		return fmt.Sprintf("Network error: %v. Please check your connection and try again.", netErr.Err)
	}

	// Check for API error
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.Message != "" {
			return apiErr.Message
		}
		return apiErr.Error()
	}

	// Default
	return err.Error()
}
