package errors

import (
	"errors"
	"testing"
)

func TestParseAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       []byte
		wantType   string
		wantMsg    string
	}{
		{
			name:       "quota error with JSON response",
			statusCode: 400,
			body:       []byte(`{"code":"NO_RUNS_REMAINING","message":"No runs remaining","details":{"tier":"Free","limit":10,"remaining":0}}`),
			wantType:   "*errors.QuotaError",
			wantMsg:    "no runs remaining (Tier: Free, Limit: 10/month). Upgrade at: https://repobird.ai/dashboard",
		},
		{
			name:       "auth error with JSON response",
			statusCode: 401,
			body:       []byte(`{"code":"INVALID_API_KEY","message":"Invalid API key provided"}`),
			wantType:   "*errors.AuthError",
			wantMsg:    "authentication failed: Invalid API key. Get a new one at https://repobird.ai/settings/api (INVALID_API_KEY)",
		},
		{
			name:       "rate limit error",
			statusCode: 429,
			body:       []byte(`{"code":"RATE_LIMIT_EXCEEDED","details":{"retry_after":"60s"}}`),
			wantType:   "*errors.RateLimitError",
			wantMsg:    "rate limit exceeded. Please wait 60s before retrying",
		},
		{
			name:       "server error with plain text",
			statusCode: 500,
			body:       []byte("Internal server error"),
			wantType:   "*errors.APIError",
			wantMsg:    "Internal server error (status 500)",
		},
		{
			name:       "not found error",
			statusCode: 404,
			body:       []byte(""),
			wantType:   "*errors.APIError",
			wantMsg:    "API request failed with status 404 (status 404)",
		},
		{
			name:       "validation error",
			statusCode: 422,
			body:       []byte(`{"code":"VALIDATION_ERROR","message":"Invalid field value","details":{"field":"prompt"}}`),
			wantType:   "*errors.ValidationError",
			wantMsg:    "validation error for field 'prompt': Invalid field value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseAPIError(tt.statusCode, tt.body)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Check error type by checking the error message structure
			if err.Error() != tt.wantMsg {
				t.Errorf("ParseAPIError() error message = %v, want %v", err.Error(), tt.wantMsg)
			}

			// Check specific error types
			switch tt.wantType {
			case "*errors.QuotaError":
				var quotaErr *QuotaError
				if !errors.As(err, &quotaErr) {
					t.Errorf("expected QuotaError, got %T", err)
				}
			case "*errors.AuthError":
				var authErr *AuthError
				if !errors.As(err, &authErr) {
					t.Errorf("expected AuthError, got %T", err)
				}
			case "*errors.RateLimitError":
				var rateLimitErr *RateLimitError
				if !errors.As(err, &rateLimitErr) {
					t.Errorf("expected RateLimitError, got %T", err)
				}
			case "*errors.APIError":
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Errorf("expected APIError, got %T", err)
				}
			case "*errors.ValidationError":
				var validationErr *ValidationError
				if !errors.As(err, &validationErr) {
					t.Errorf("expected ValidationError, got %T", err)
				}
			}
		})
	}
}

func TestFormatUserError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name: "quota error",
			err: &QuotaError{
				Tier:       "Free",
				Limit:      10,
				UpgradeURL: "https://repobird.ai/dashboard",
			},
			expected: "no runs remaining (Tier: Free, Limit: 10/month). Upgrade at: https://repobird.ai/dashboard",
		},
		{
			name: "auth error",
			err: &AuthError{
				Message: "Invalid API key",
				Reason:  "invalid_key",
			},
			expected: "authentication failed: Invalid API key (invalid_key)",
		},
		{
			name: "rate limit error",
			err: &RateLimitError{
				RetryAfter: "30s",
			},
			expected: "rate limit exceeded. Please wait 30s before retrying",
		},
		{
			name: "validation error with field",
			err: &ValidationError{
				Field:   "prompt",
				Message: "cannot be empty",
			},
			expected: "validation error for field 'prompt': cannot be empty",
		},
		{
			name: "network error",
			err: &NetworkError{
				Err:       errors.New("connection refused"),
				Operation: "POST /api/v1/runs",
			},
			expected: "Network error: connection refused. Please check your connection and try again.",
		},
		{
			name: "API error with message",
			err: &APIError{
				StatusCode: 500,
				Message:    "Internal server error",
				ErrorType:  ErrorTypeAPI,
			},
			expected: "Internal server error",
		},
		{
			name:     "generic error",
			err:      errors.New("something went wrong"),
			expected: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatUserError(tt.err); got != tt.expected {
				t.Errorf("FormatUserError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetHTTPStatusText(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{400, "Bad Request"},
		{401, "Unauthorized"},
		{403, "Forbidden"},
		{404, "Not Found"},
		{408, "Request Timeout"},
		{422, "Unprocessable Entity"},
		{429, "Too Many Requests"},
		{500, "Internal Server Error"},
		{502, "Bad Gateway"},
		{503, "Service Unavailable"},
		{504, "Gateway Timeout"},
		{999, "HTTP 999"},
	}

	for _, tt := range tests {
		t.Run("status_"+string(rune(tt.code)), func(t *testing.T) {
			if got := getHTTPStatusText(tt.code); got != tt.expected {
				t.Errorf("getHTTPStatusText(%d) = %v, want %v", tt.code, got, tt.expected)
			}
		})
	}
}

func TestCreateErrorFromStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
		wantType   interface{}
	}{
		{
			name:       "401 unauthorized",
			statusCode: 401,
			message:    "Custom auth message",
			wantType:   &AuthError{},
		},
		{
			name:       "403 forbidden",
			statusCode: 403,
			message:    "",
			wantType:   &AuthError{},
		},
		{
			name:       "429 rate limited",
			statusCode: 429,
			message:    "",
			wantType:   &APIError{}, // Changed to return APIError with status code
		},
		{
			name:       "422 validation error",
			statusCode: 422,
			message:    "Invalid input",
			wantType:   &ValidationError{},
		},
		{
			name:       "500 server error",
			statusCode: 500,
			message:    "Server error",
			wantType:   &APIError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := createErrorFromStatusCode(tt.statusCode, tt.message)

			switch tt.wantType.(type) {
			case *AuthError:
				var authErr *AuthError
				if !errors.As(err, &authErr) {
					t.Errorf("expected AuthError, got %T", err)
				}
			case *RateLimitError:
				var rateLimitErr *RateLimitError
				if !errors.As(err, &rateLimitErr) {
					t.Errorf("expected RateLimitError, got %T", err)
				}
			case *ValidationError:
				var validationErr *ValidationError
				if !errors.As(err, &validationErr) {
					t.Errorf("expected ValidationError, got %T", err)
				}
			case *APIError:
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Errorf("expected APIError, got %T", err)
				}
			}
		})
	}
}
