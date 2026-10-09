package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigLoaderPreservesReasoningEffort(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "run.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{
		"prompt":"Fix the issue",
		"repository":"owner/repo",
		"runType":"run",
		"reasoningEffort":"future-adaptive"
	}`), 0o600))

	runRequest, err := NewConfigLoader().LoadConfig(configPath)
	require.NoError(t, err)
	require.Equal(t, "future-adaptive", runRequest.ReasoningEffort)
}
