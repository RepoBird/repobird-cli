package messages

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNavigationMessages(t *testing.T) {
	tests := []struct {
		name     string
		msg      NavigationMsg
		expected bool
	}{
		{
			name:     "NavigateToCreateMsg implements NavigationMsg",
			msg:      NavigateToCreateMsg{},
			expected: true,
		},
		{
			name:     "NavigateToCreateMsg with repository",
			msg:      NavigateToCreateMsg{SelectedRepository: "test/repo"},
			expected: true,
		},
		{
			name:     "NavigateToDetailsMsg implements NavigationMsg",
			msg:      NavigateToDetailsMsg{RunID: "123"},
			expected: true,
		},
		{
			name:     "NavigateToDetailsMsg with FromCreate flag",
			msg:      NavigateToDetailsMsg{RunID: "456", FromCreate: true},
			expected: true,
		},
		{
			name:     "NavigateToDashboardMsg implements NavigationMsg",
			msg:      NavigateToDashboardMsg{},
			expected: true,
		},
		{
			name:     "NavigateToListMsg implements NavigationMsg",
			msg:      NavigateToListMsg{},
			expected: true,
		},
		{
			name:     "NavigateToListMsg with selected index",
			msg:      NavigateToListMsg{SelectedIndex: 5},
			expected: true,
		},
		{
			name:     "NavigateToBulkMsg implements NavigationMsg",
			msg:      NavigateToBulkMsg{},
			expected: true,
		},
		{
			name:     "NavigateBackMsg implements NavigationMsg",
			msg:      NavigateBackMsg{},
			expected: true,
		},
		{
			name: "NavigateToErrorMsg recoverable",
			msg: NavigateToErrorMsg{
				Error:       errors.New("test error"),
				Message:     "Something went wrong",
				Recoverable: true,
				ReturnTo:    "dashboard",
			},
			expected: true,
		},
		{
			name: "NavigateToErrorMsg non-recoverable",
			msg: NavigateToErrorMsg{
				Error:       errors.New("fatal error"),
				Message:     "Fatal error occurred",
				Recoverable: false,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that all messages implement the NavigationMsg interface
			assert.Equal(t, tt.expected, tt.msg.IsNavigation())

			// Verify the message is actually a NavigationMsg
			_, ok := tt.msg.(NavigationMsg)
			assert.True(t, ok, "Message should implement NavigationMsg interface")
		})
	}
}

func TestNavigationMessageFields(t *testing.T) {
	t.Run("NavigateToCreateMsg fields", func(t *testing.T) {
		msg := NavigateToCreateMsg{
			SelectedRepository: "org/repo",
		}
		assert.Equal(t, "org/repo", msg.SelectedRepository)
	})

	t.Run("NavigateToDetailsMsg fields", func(t *testing.T) {
		msg := NavigateToDetailsMsg{
			RunID:      "run-123",
			FromCreate: true,
		}
		assert.Equal(t, "run-123", msg.RunID)
		assert.True(t, msg.FromCreate)
	})

	t.Run("NavigateToListMsg fields", func(t *testing.T) {
		msg := NavigateToListMsg{
			SelectedIndex: 10,
		}
		assert.Equal(t, 10, msg.SelectedIndex)
	})

	t.Run("NavigateToErrorMsg fields", func(t *testing.T) {
		err := errors.New("test error")
		msg := NavigateToErrorMsg{
			Error:       err,
			Message:     "Error message",
			Recoverable: true,
			ReturnTo:    "list",
		}
		assert.Equal(t, err, msg.Error)
		assert.Equal(t, "Error message", msg.Message)
		assert.True(t, msg.Recoverable)
		assert.Equal(t, "list", msg.ReturnTo)
	})
}

func TestNavigationMessageUsage(t *testing.T) {
	// Test that messages can be used in type switches
	t.Run("Type switch compatibility", func(t *testing.T) {
		messages := []NavigationMsg{
			NavigateToCreateMsg{},
			NavigateToDetailsMsg{RunID: "123"},
			NavigateToDashboardMsg{},
			NavigateToListMsg{SelectedIndex: 1},
			NavigateToBulkMsg{},
			NavigateBackMsg{},
			NavigateToErrorMsg{Error: errors.New("test")},
		}

		for _, msg := range messages {
			var result string
			switch m := msg.(type) {
			case NavigateToCreateMsg:
				result = "create"
			case NavigateToDetailsMsg:
				result = "details:" + m.RunID
			case NavigateToDashboardMsg:
				result = "dashboard"
			case NavigateToListMsg:
				result = "list"
			case NavigateToBulkMsg:
				result = "bulk"
			case NavigateBackMsg:
				result = "back"
			case NavigateToErrorMsg:
				result = "error"
			default:
				t.Errorf("Unexpected message type: %T", msg)
			}
			assert.NotEmpty(t, result, "Should handle all navigation message types")
		}
	})
}
