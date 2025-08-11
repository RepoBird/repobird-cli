package utils

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParseYAMLConfig(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid minimal YAML",
			file:    "../../tests/testdata/valid/minimal.yaml",
			wantErr: false,
		},
		{
			name:    "valid multiline YAML",
			file:    "../../tests/testdata/valid/multiline_yaml.yaml",
			wantErr: false,
		},
		{
			name:    "invalid - missing required fields",
			file:    "../../tests/testdata/invalid/missing_required.yaml",
			wantErr: true,
			errMsg:  "validation errors",
		},
		{
			name:    "invalid - bad syntax",
			file:    "../../tests/testdata/invalid/invalid_syntax.yaml",
			wantErr: true,
			errMsg:  "failed to parse YAML",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := filepath.Abs(tt.file)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			config, err := ParseYAMLConfig(absPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseYAMLConfig() expected error but got none")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ParseYAMLConfig() error = %v, want error containing %v", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ParseYAMLConfig() unexpected error = %v", err)
				}
				if config == nil {
					t.Errorf("ParseYAMLConfig() returned nil config")
				}
			}
		})
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	tests := []struct {
		name              string
		file              string
		wantErr           bool
		checkContext      bool
		expectedInContext string
	}{
		{
			name:    "load JSON config",
			file:    "../../tests/testdata/valid/test_config.json",
			wantErr: false,
		},
		{
			name:    "load YAML config",
			file:    "../../tests/testdata/valid/test_config.yaml",
			wantErr: false,
		},
		{
			name:              "load Markdown config with body",
			file:              "../../tests/testdata/valid/example_task.md",
			wantErr:           false,
			checkContext:      true,
			expectedInContext: "Database Connection Refactoring Task",
		},
		{
			name:              "load comprehensive Markdown",
			file:              "../../tests/testdata/valid/comprehensive_task.md",
			wantErr:           false,
			checkContext:      true,
			expectedInContext: "Microservices Architecture Implementation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := filepath.Abs(tt.file)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			config, additionalContext, err := LoadConfigFromFile(absPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadConfigFromFile() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("LoadConfigFromFile() unexpected error = %v", err)
				}
				if config == nil {
					t.Errorf("LoadConfigFromFile() returned nil config")
				}
				if tt.checkContext && !strings.Contains(additionalContext, tt.expectedInContext) {
					t.Errorf("LoadConfigFromFile() additionalContext = %v, want containing %v", additionalContext, tt.expectedInContext)
				}
			}
		})
	}
}

func TestValidateRunConfigYAML(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{
			name:    "valid config with defaults",
			file:    "../../tests/testdata/valid/minimal.yaml",
			wantErr: false,
		},
		{
			name:    "invalid config - missing required",
			file:    "../../tests/testdata/invalid/missing_required.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			absPath, err := filepath.Abs(tt.file)
			if err != nil {
				t.Fatalf("Failed to get absolute path: %v", err)
			}

			// Try to load and validate
			config, _, _ := LoadConfigFromFile(absPath)
			if config != nil {
				err = ValidateRunConfig(config)
				if tt.wantErr && err == nil {
					t.Errorf("ValidateRunConfig() expected error but got none")
				} else if !tt.wantErr && err != nil {
					t.Errorf("ValidateRunConfig() unexpected error = %v", err)
				}
			}
		})
	}
}