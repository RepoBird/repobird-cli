package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/config"
)

func TestModelsCommandIsRegistered(t *testing.T) {
	command, _, err := rootCmd.Find([]string{"models"})
	if err != nil {
		t.Fatalf("expected models command to be registered: %v", err)
	}
	if command.Name() != "models" {
		t.Fatalf("command name = %q, want models", command.Name())
	}
}

func TestModelsCommandOutputsHumanAndStableJSONFromPublicCatalog(t *testing.T) {
	previousJSONOutput := jsonOutput
	t.Cleanup(func() { jsonOutput = previousJSONOutput })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/models", r.URL.Path)
		require.Empty(t, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"data":{"catalogRevision":"catalog_test","futureCatalogField":"keep-me","models":[{"id":"future-provider/future-model-v9","label":"Future Model","provider":"Future Provider","capabilities":["future-flow"],"isDefault":true,"futureDescriptorField":{"displayHint":"safe"},"policyHash":"redact-me","metadata":{"safe":true,"workerProfile":"redact-me"}}]}}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	previousConfig := cfg
	cfg = &config.SecureConfig{Config: &config.Config{APIURL: server.URL}}
	t.Cleanup(func() { cfg = previousConfig })

	t.Run("human", func(t *testing.T) {
		jsonOutput = false
		cmd := newModelsCommand()
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		require.NoError(t, cmd.Execute())
		require.Contains(t, output.String(), "Catalog revision: catalog_test")
		require.Contains(t, output.String(), "future-provider/future-model-v9")
		require.Contains(t, output.String(), "Future Provider")
		require.Contains(t, output.String(), "futureCatalogField")
		require.Contains(t, output.String(), "keep-me")
		require.Contains(t, output.String(), "futureDescriptorField")
		require.Contains(t, output.String(), "displayHint")
		require.NotContains(t, output.String(), "policyHash")
		require.NotContains(t, output.String(), "workerProfile")
	})

	t.Run("json", func(t *testing.T) {
		jsonOutput = false
		cmd := newModelsCommand()
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		cmd.SetArgs([]string{"--json"})
		require.NoError(t, cmd.Execute())

		var decoded map[string]any
		require.NoError(t, json.Unmarshal(output.Bytes(), &decoded))
		encoded, err := json.Marshal(decoded)
		require.NoError(t, err)
		require.Contains(t, string(encoded), "futureCatalogField")
		require.Contains(t, string(encoded), "futureDescriptorField")
		require.NotContains(t, string(encoded), "policyHash")
		require.NotContains(t, string(encoded), "workerProfile")
	})
}
