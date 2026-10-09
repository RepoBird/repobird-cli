package dto

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicModelsResponsePreservesSafeAdditiveFieldsAndRedactsInternalData(t *testing.T) {
	const fixture = `{
  "data": {
    "catalogRevision": "catalog_test",
    "futureCatalogField": "preserve-me",
    "models": [
      {
        "id": "future-provider/future-model-v9",
        "label": "Future Model",
        "provider": "Future Provider",
        "capabilities": ["future-flow"],
        "isDefault": true,
        "futureDescriptorField": {"displayHint": "safe"},
        "policyHash": "must-not-survive",
        "workerProfile": "must-not-survive",
        "credentialId": "must-not-survive",
        "apiKey": "must-not-survive",
        "email": "must-not-survive",
        "password": "must-not-survive",
        "tenantAvailability": "must-not-survive",
        "metadata": {"safe": true, "workerProfile": "must-not-survive"}
      }
    ]
  }
}`

	var response PublicModelsResponse
	require.NoError(t, json.Unmarshal([]byte(fixture), &response))

	encoded, err := json.Marshal(response)
	require.NoError(t, err)

	var output map[string]any
	require.NoError(t, json.Unmarshal(encoded, &output))
	data := output["data"].(map[string]any)
	model := data["models"].([]any)[0].(map[string]any)

	require.Equal(t, "preserve-me", data["futureCatalogField"])
	require.Equal(t, "Future Provider", model["provider"])
	require.Equal(t, []any{"future-flow"}, model["capabilities"])
	require.Equal(t, "safe", model["futureDescriptorField"].(map[string]any)["displayHint"])
	require.NotContains(t, data, "policyHash")
	require.NotContains(t, model, "policyHash")
	require.NotContains(t, model, "workerProfile")
	require.NotContains(t, model, "credentialId")
	require.NotContains(t, model, "apiKey")
	require.NotContains(t, model, "email")
	require.NotContains(t, model, "password")
	require.NotContains(t, model, "tenantAvailability")
	require.NotContains(t, model["metadata"], "workerProfile")
}
