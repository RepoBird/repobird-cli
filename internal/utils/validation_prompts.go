package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ValidationPrompt represents a single validation prompt
type ValidationPrompt struct {
	Type        string // "field_suggestion", "unknown_field", "validation_error", "confirmation"
	Message     string // The prompt message to show user
	Field       string // Field name (for field-related prompts)
	Suggestion  string // Suggested field name (for similarity prompts)
	Required    bool   // Whether user must respond (true) or can skip (false)
	DefaultNo   bool   // Whether default response is No (for safety)
}

// ValidationPromptHandler manages multiple validation prompts in sequence
type ValidationPromptHandler struct {
	prompts []ValidationPrompt
	results map[string]string // Field -> user response
}

// NewValidationPromptHandler creates a new prompt handler
func NewValidationPromptHandler() *ValidationPromptHandler {
	return &ValidationPromptHandler{
		prompts: []ValidationPrompt{},
		results: make(map[string]string),
	}
}

// AddFieldSuggestionPrompt adds a field similarity suggestion prompt
func (h *ValidationPromptHandler) AddFieldSuggestionPrompt(field, suggestion string) {
	h.prompts = append(h.prompts, ValidationPrompt{
		Type:       "field_suggestion",
		Message:    fmt.Sprintf("Unknown field '%s' - did you mean '%s'?", field, suggestion),
		Field:      field,
		Suggestion: suggestion,
		Required:   false,
		DefaultNo:  true, // Default to No for safety
	})
}

// AddUnknownFieldWarning adds a generic unknown field warning
func (h *ValidationPromptHandler) AddUnknownFieldWarning(fields []string) {
	if len(fields) > 0 {
		h.prompts = append(h.prompts, ValidationPrompt{
			Type:      "unknown_field",
			Message:   fmt.Sprintf("Configuration contains unsupported fields: %s. Continue anyway?", strings.Join(fields, ", ")),
			Required:  false,
			DefaultNo: false, // Allow continuing by default
		})
	}
}

// AddValidationError adds a validation error prompt
func (h *ValidationPromptHandler) AddValidationError(message string) {
	h.prompts = append(h.prompts, ValidationPrompt{
		Type:      "validation_error",
		Message:   fmt.Sprintf("Validation error: %s", message),
		Required:  true, // User must acknowledge/fix validation errors
		DefaultNo: true, // Don't continue with validation errors by default
	})
}

// AddConfirmationPrompt adds a final confirmation prompt
func (h *ValidationPromptHandler) AddConfirmationPrompt(message string) {
	h.prompts = append(h.prompts, ValidationPrompt{
		Type:      "confirmation",
		Message:   message,
		Required:  false,
		DefaultNo: false,
	})
}

// HasPrompts returns true if there are prompts to show
func (h *ValidationPromptHandler) HasPrompts() bool {
	return len(h.prompts) > 0
}

// ProcessPrompts shows all prompts in sequence and collects responses
// Returns true if user wants to proceed, false if they want to cancel
func (h *ValidationPromptHandler) ProcessPrompts() (bool, error) {
	if len(h.prompts) == 0 {
		return true, nil
	}

	reader := bufio.NewReader(os.Stdin)
	
	for i, prompt := range h.prompts {
		// Show prompt number if multiple prompts
		if len(h.prompts) > 1 {
			fmt.Printf("\n[%d/%d] ", i+1, len(h.prompts))
		}
		
		response, err := h.showPrompt(prompt, reader)
		if err != nil {
			return false, err
		}
		
		// Store response
		if prompt.Field != "" {
			h.results[prompt.Field] = response
		}
		
		// Handle responses that should stop the process
		if prompt.Required && (response == "n" || response == "no") {
			fmt.Println("Operation cancelled.")
			return false, nil
		}
	}
	
	return true, nil
}

// showPrompt displays a single prompt and gets user response
func (h *ValidationPromptHandler) showPrompt(prompt ValidationPrompt, reader *bufio.Reader) (string, error) {
	var defaultResponse string
	var promptSuffix string
	
	switch prompt.Type {
	case "field_suggestion":
		if prompt.DefaultNo {
			promptSuffix = " [y/N]: "
			defaultResponse = "n"
		} else {
			promptSuffix = " [Y/n]: "
			defaultResponse = "y"
		}
	case "unknown_field":
		if prompt.DefaultNo {
			promptSuffix = " [y/N]: "
			defaultResponse = "n"
		} else {
			promptSuffix = " [Y/n]: "
			defaultResponse = "y"
		}
	case "validation_error":
		if prompt.Required {
			promptSuffix = " (Press Enter to continue or Ctrl+C to cancel): "
			defaultResponse = "ok"
		} else {
			promptSuffix = " [y/N]: "
			defaultResponse = "n"
		}
	case "confirmation":
		if prompt.DefaultNo {
			promptSuffix = " [y/N]: "
			defaultResponse = "n"
		} else {
			promptSuffix = " [Y/n]: "
			defaultResponse = "y"
		}
	default:
		promptSuffix = " [Y/n]: "
		defaultResponse = "y"
	}
	
	fmt.Printf("%s%s", prompt.Message, promptSuffix)
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		input = defaultResponse
	}
	
	return input, nil
}

// GetFieldResponse returns the user's response for a specific field
func (h *ValidationPromptHandler) GetFieldResponse(field string) string {
	return h.results[field]
}

// ShouldContinue returns whether user chose to continue after all prompts
func (h *ValidationPromptHandler) ShouldContinue() bool {
	// Check if any required prompts were answered with "no"
	for _, prompt := range h.prompts {
		if prompt.Required {
			if prompt.Field != "" {
				response := h.results[prompt.Field]
				if response == "n" || response == "no" {
					return false
				}
			}
		}
	}
	return true
}