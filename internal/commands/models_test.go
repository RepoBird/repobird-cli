package commands

import "testing"

func TestModelsCommandIsRegistered(t *testing.T) {
	command, _, err := rootCmd.Find([]string{"models"})
	if err != nil {
		t.Fatalf("expected models command to be registered: %v", err)
	}
	if command.Name() != "models" {
		t.Fatalf("command name = %q, want models", command.Name())
	}
}
