// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package views

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomCreateForm_FieldDefaults(t *testing.T) {
	form := NewCustomCreateForm()
	require.NotNil(t, form)

	values := form.GetValues()

	// Check all field defaults
	tests := []struct {
		field    string
		expected string
		desc     string
	}{
		{"title", "", "Title should be empty by default"},
		{"repository", "", "Repository should be empty by default"},
		{"source", "", "Source branch should be empty (no default 'main')"},
		{"target", "", "Target branch should be empty by default"},
		{"prompt", "", "Prompt should be empty by default"},
		{"context", "", "Context should be empty by default"},
		{"runtype", "run", "Run type should default to 'run'"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			assert.Equal(t, tt.expected, values[tt.field], tt.desc)
		})
	}
}

func TestCustomCreateForm_RequiredFields(t *testing.T) {
	form := NewCustomCreateForm()

	// Check which fields are required
	expectedRequired := map[string]bool{
		"title":      false, // Not required per user request
		"repository": true,  // Required
		"source":     false, // Not required per user request
		"target":     false, // Not required
		"prompt":     true,  // Required
		"context":    false, // Not required
		"runtype":    false, // Not required (has default)
	}

	for _, field := range form.fields {
		if field.Type != "button" {
			expected := expectedRequired[field.Name]
			assert.Equal(t, expected, field.Required,
				"Field %s required status should be %v", field.Name, expected)
		}
	}
}

func TestCustomCreateForm_SetAndGetValue(t *testing.T) {
	form := NewCustomCreateForm()

	// Test setting values for each field type
	testCases := []struct {
		field string
		value string
	}{
		{"title", "Test Title"},
		{"repository", "test/repo"},
		{"source", "feature-branch"},
		{"target", "main"},
		{"prompt", "Test prompt"},
		{"context", "Test context"},
		{"runtype", "plan"},
	}

	for _, tc := range testCases {
		form.SetValue(tc.field, tc.value)
		values := form.GetValues()
		assert.Equal(t, tc.value, values[tc.field],
			"Field %s should have value %s", tc.field, tc.value)
	}
}

func TestCustomCreateForm_SourceBranchNoDefault(t *testing.T) {
	form := NewCustomCreateForm()

	// Find the source field
	var sourceField *CustomFormField
	for i := range form.fields {
		if form.fields[i].Name == "source" {
			sourceField = &form.fields[i]
			break
		}
	}

	require.NotNil(t, sourceField, "Source field should exist")

	// Verify source branch has no default value
	assert.Equal(t, "", sourceField.Value, "Source branch should have empty value by default")
	assert.Equal(t, "main", sourceField.Placeholder, "Source branch should have 'main' as placeholder")
	assert.False(t, sourceField.Required, "Source branch should not be required")
}

func TestCustomCreateForm_Validation(t *testing.T) {
	form := NewCustomCreateForm()

	// Initially invalid (missing required fields)
	valid := form.validate()
	assert.False(t, valid, "Form should be invalid without required fields")

	// Set only repository (still missing prompt)
	form.SetValue("repository", "test/repo")
	valid = form.validate()
	assert.False(t, valid, "Form should be invalid without prompt")

	// Set prompt (now should be valid)
	form.SetValue("prompt", "Test prompt")
	valid = form.validate()
	assert.True(t, valid, "Form should be valid with all required fields")

	// Empty repository should make it invalid again
	form.SetValue("repository", "")
	valid = form.validate()
	assert.False(t, valid, "Form should be invalid with empty repository")
}

func TestCustomCreateForm_ClearCurrentField(t *testing.T) {
	form := NewCustomCreateForm()

	// Set values for text fields
	form.SetValue("title", "Test Title")
	form.SetValue("repository", "test/repo")
	form.SetValue("prompt", "Test prompt")

	// Focus on repository field (index 1)
	form.SetFocusIndex(1)
	assert.Equal(t, "repository", form.GetCurrentFieldName())

	// Clear current field
	form.ClearCurrentField()

	values := form.GetValues()
	assert.Equal(t, "", values["repository"], "Repository should be cleared")
	assert.Equal(t, "Test Title", values["title"], "Title should remain unchanged")
	assert.Equal(t, "Test prompt", values["prompt"], "Prompt should remain unchanged")
}

func TestCustomCreateForm_InsertMode(t *testing.T) {
	form := NewCustomCreateForm()

	// Initially in normal mode
	assert.False(t, form.IsInsertMode())

	// Enter insert mode
	form.SetInsertMode(true)
	assert.True(t, form.IsInsertMode())

	// Exit insert mode
	form.SetInsertMode(false)
	assert.False(t, form.IsInsertMode())
}

func TestCustomCreateForm_RunTypeToggle(t *testing.T) {
	form := NewCustomCreateForm()

	// Initially "run"
	assert.Equal(t, "run", form.GetRunType())

	// Change to "plan"
	form.SetValue("runtype", "plan")
	assert.Equal(t, "plan", form.GetRunType())

	// Change back to "run"
	form.SetValue("runtype", "run")
	assert.Equal(t, "run", form.GetRunType())
}

func TestCustomCreateForm_FieldIcons(t *testing.T) {
	form := NewCustomCreateForm()

	// Check that all fields have icons
	expectedIcons := map[string]string{
		"title":      "📝",
		"repository": "📦",
		"source":     "🌿",
		"target":     "🎯",
		"prompt":     "💭",
		"context":    "📋",
		"runtype":    "⚙️",
		"submit":     "🚀",
	}

	for _, field := range form.fields {
		expected, exists := expectedIcons[field.Name]
		assert.True(t, exists, "Field %s should have an expected icon", field.Name)
		assert.Equal(t, expected, field.Icon, "Field %s should have icon %s", field.Name, expected)
	}
}
