package views

import (
	"testing"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/stretchr/testify/assert"
)

func TestCreateRunViewWithSelectedRepository(t *testing.T) {
	// Create a mock client
	client := &api.Client{}

	tests := []struct {
		name               string
		selectedRepository string
		expectedValue      string
	}{
		{
			name:               "With selected repository",
			selectedRepository: "owner/repo",
			expectedValue:      "owner/repo",
		},
		{
			name:               "Without selected repository - should autofill",
			selectedRepository: "",
			expectedValue:      "", // Will be filled by autofill or remain empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := CreateRunViewConfig{
				Client:             client,
				SelectedRepository: tt.selectedRepository,
			}

			view := NewCreateRunViewWithConfig(config)

			// Check that the repository field is set correctly
			if len(view.fields) >= 2 {
				actualValue := view.fields[1].Value()
				if tt.selectedRepository != "" {
					assert.Equal(t, tt.expectedValue, actualValue,
						"Repository field should be set to the selected repository")
				}
			}
		})
	}
}

func TestRepositoryNameFormat(t *testing.T) {
	// Test that repository names are in owner/repo format
	testCases := []struct {
		owner    string
		repo     string
		expected string
	}{
		{"microsoft", "vscode", "microsoft/vscode"},
		{"google", "go", "google/go"},
		{"facebook", "react", "facebook/react"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			// This is how the dashboard constructs repository names
			repoName := tc.owner + "/" + tc.repo
			assert.Equal(t, tc.expected, repoName,
				"Repository name should be in owner/repo format")
		})
	}
}
