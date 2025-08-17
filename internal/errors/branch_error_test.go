package errors

import (
	"strings"
	"testing"
)

func TestBranchNotFoundError(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		responseBody []byte
		expectedMsg  string
		shouldContain string // For partial matches
	}{
		{
			name:       "Target branch not found with detailed message",
			statusCode: 400,
			responseBody: []byte(`{
				"error": "BRANCH_NOT_FOUND",
				"message": "Target branch 'jdfsjdfksdf' does not exist in test-acc-254/youtube-music. Please create it first, or leave target empty to merge back to source branch."
			}`),
			expectedMsg: "Target branch 'jdfsjdfksdf' does not exist in test-acc-254/youtube-music. Please create it first, or leave target empty to merge back to source branch.",
		},
		{
			name:       "Source branch not found",
			statusCode: 400,
			responseBody: []byte(`{
				"error": "BRANCH_NOT_FOUND",
				"message": "Source branch 'feature/nonexistent' does not exist in acme/webapp"
			}`),
			expectedMsg: "Source branch 'feature/nonexistent' does not exist in acme/webapp",
		},
		{
			name:       "Branch not found with minimal message",
			statusCode: 400,
			responseBody: []byte(`{
				"error": "BRANCH_NOT_FOUND",
				"message": "Branch 'develop' not found"
			}`),
			expectedMsg: "Branch 'develop' not found",
		},
		{
			name:       "Branch not found with no message",
			statusCode: 400,
			responseBody: []byte(`{
				"error": "BRANCH_NOT_FOUND"
			}`),
			expectedMsg: "Branch not found in repository",
		},
		{
			name:       "Branch not found using code field instead of error",
			statusCode: 400,
			responseBody: []byte(`{
				"code": "BRANCH_NOT_FOUND",
				"message": "The specified branch does not exist"
			}`),
			expectedMsg: "The specified branch does not exist",
		},
		{
			name:       "Branch validation error in bulk operation",
			statusCode: 400,
			responseBody: []byte(`{
				"error": "BRANCH_NOT_FOUND",
				"message": "Source branch 'main' does not exist in test-org/test-repo. All bulk runs rejected."
			}`),
			shouldContain: "All bulk runs rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the API error
			err := ParseAPIError(tt.statusCode, tt.responseBody)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			// Check that it's an APIError with the right type
			apiErr, ok := err.(*APIError)
			if !ok {
				t.Fatalf("Expected APIError type, got %T", err)
			}

			// Verify error type is NotFound
			if apiErr.ErrorType != ErrorTypeNotFound {
				t.Errorf("Expected ErrorType to be ErrorTypeNotFound, got %v", apiErr.ErrorType)
			}

			// Format the error for user display
			formatted := FormatUserError(err)
			
			// Check the formatted message
			if tt.shouldContain != "" {
				if !strings.Contains(formatted, tt.shouldContain) {
					t.Errorf("Expected message to contain %q, got: %q", tt.shouldContain, formatted)
				}
			} else if formatted != tt.expectedMsg {
				t.Errorf("Expected message: %q, got: %q", tt.expectedMsg, formatted)
			}
		})
	}
}

func TestBranchNotFoundErrorIntegration(t *testing.T) {
	// Test that BRANCH_NOT_FOUND errors work correctly in the run command context
	testCases := []struct {
		name         string
		apiResponse  []byte
		expectedUser string // What the user should see
	}{
		{
			name: "User-friendly target branch error",
			apiResponse: []byte(`{
				"error": "BRANCH_NOT_FOUND",
				"message": "Target branch 'release/v2.0' does not exist in myorg/myapp. Please create it first, or leave target empty to merge back to source branch."
			}`),
			expectedUser: "Target branch 'release/v2.0' does not exist in myorg/myapp. Please create it first, or leave target empty to merge back to source branch.",
		},
		{
			name: "User-friendly source branch error",
			apiResponse: []byte(`{
				"error": "BRANCH_NOT_FOUND",
				"message": "Source branch 'main' does not exist in myorg/myapp. The repository's default branch may be different (e.g., 'master')."
			}`),
			expectedUser: "Source branch 'main' does not exist in myorg/myapp. The repository's default branch may be different (e.g., 'master').",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse error as would happen in the API client
			err := ParseAPIError(400, tc.apiResponse)
			
			// Format as would be shown to user in run command
			// e.g., fmt.Errorf("failed to create run: %s", errors.FormatUserError(err))
			userMessage := FormatUserError(err)
			
			if userMessage != tc.expectedUser {
				t.Errorf("User would see:\n  %q\nExpected:\n  %q", userMessage, tc.expectedUser)
			}
		})
	}
}
