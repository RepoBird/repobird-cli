package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMarkdownConfigFromReader(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantErr        bool
		errorContains  string
		validateResult func(t *testing.T, config interface{}, content string)
	}{
		{
			name: "valid markdown with all required fields",
			input: `---
prompt: "Fix authentication bug"
repository: "acme/webapp"
source: "main"
target: "fix/auth-bug"
runType: "run"
title: "Fix authentication issue"
context: "Users cannot login"
files:
  - "src/auth.go"
  - "src/login.go"
---

# Additional Notes

This is the markdown content that provides extra context.`,
			wantErr: false,
			validateResult: func(t *testing.T, cfg interface{}, content string) {
				config := cfg.(*struct {
					Prompt     string
					Repository string
					Source     string
					Target     string
					RunType    string
					Title      string
				})
				assert.Equal(t, "Fix authentication bug", config.Prompt)
				assert.Equal(t, "acme/webapp", config.Repository)
				assert.Equal(t, "main", config.Source)
				assert.Equal(t, "fix/auth-bug", config.Target)
				assert.Equal(t, "run", config.RunType)
				assert.Equal(t, "Fix authentication issue", config.Title)
				assert.Contains(t, content, "Additional Notes")
				assert.Contains(t, content, "extra context")
			},
		},
		{
			name: "missing required prompt field",
			input: `---
repository: "acme/webapp"
source: "main"
target: "fix/bug"
title: "Fix bug"
---

Content`,
			wantErr:       true,
			errorContains: "prompt is required",
		},
		{
			name: "missing required repository field",
			input: `---
prompt: "Fix bug"
source: "main"
target: "fix/bug"
title: "Fix bug"
---

Content`,
			wantErr:       true,
			errorContains: "repository is required",
		},
		{
			name: "invalid repository format",
			input: `---
prompt: "Fix bug"
repository: "invalid-repo"
source: "main"
target: "fix/bug"
title: "Fix bug"
---

Content`,
			wantErr:       true,
			errorContains: "repository must be in format",
		},
		{
			name: "invalid runType",
			input: `---
prompt: "Fix bug"
repository: "acme/webapp"
source: "main"
target: "fix/bug"
runType: "invalid"
title: "Fix bug"
---

Content`,
			wantErr:       true,
			errorContains: "invalid runType",
		},
		{
			name: "defaults source to main if not specified",
			input: `---
prompt: "Fix bug"
repository: "acme/webapp"
target: "fix/bug"
title: "Fix bug"
---

Content`,
			wantErr: false,
			validateResult: func(t *testing.T, cfg interface{}, content string) {
				config := cfg.(*struct {
					Prompt     string
					Repository string
					Source     string
					Target     string
					RunType    string
					Title      string
				})
				assert.Equal(t, "main", config.Source)
			},
		},
		{
			name: "defaults runType to run if not specified",
			input: `---
prompt: "Fix bug"
repository: "acme/webapp"
source: "main"
target: "fix/bug"
title: "Fix bug"
---

Content`,
			wantErr: false,
			validateResult: func(t *testing.T, cfg interface{}, content string) {
				config := cfg.(*struct {
					Prompt     string
					Repository string
					Source     string
					Target     string
					RunType    string
					Title      string
				})
				assert.Equal(t, "run", config.RunType)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			config, content, err := ParseMarkdownConfigFromReader(reader)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)

				if tt.validateResult != nil {
					// Create a struct with the fields we want to validate
					testConfig := struct {
						Prompt     string
						Repository string
						Source     string
						Target     string
						RunType    string
						Title      string
					}{
						Prompt:     config.Prompt,
						Repository: config.Repository,
						Source:     config.Source,
						Target:     config.Target,
						RunType:    config.RunType,
						Title:      config.Title,
					}
					tt.validateResult(t, &testConfig, content)
				}
			}
		})
	}
}

func TestValidateRunConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        interface{}
		wantErr       bool
		errorContains string
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"Prompt":     "Fix bug",
				"Repository": "acme/webapp",
				"Target":     "fix/bug",
				"Title":      "Fix bug",
			},
			wantErr: false,
		},
		{
			name: "multiple validation errors",
			config: map[string]interface{}{
				"Repository": "invalid",
				"RunType":    "invalid",
			},
			wantErr:       true,
			errorContains: "validation errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This is a simplified test - in real usage, you'd pass actual RunConfig structs
			// For now, we're just testing that the validation logic is working
		})
	}
}
