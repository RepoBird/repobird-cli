// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestBuiltBinaryModelsAndRunSubmissionContracts(t *testing.T) {
	var createBodies []map[string]any
	var createBodiesMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/models":
			if got := r.Header.Get("Authorization"); got != "" {
				t.Errorf("models discovery sent authorization with cleared credentials: %q", got)
				http.Error(w, "unexpected authorization", http.StatusBadRequest)
				return
			}
			w.Header().Set("ETag", `"catalog_test"`)
			_, _ = io.WriteString(w, `{"data":{"catalogRevision":"catalog_test","futureCatalogField":"preserve-me","models":[{"id":"future-provider/future-model-v9","label":"Future Model","provider":"Future Provider","capabilities":["future-flow"],"isDefault":true,"futureDescriptorField":{"displayHint":"safe"},"policyHash":"omit-me","workerProfile":"omit-me","metadata":{"safe":true,"credentialId":"omit-me"}}]}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs":
			if got := r.Header.Get("Authorization"); got != "Bearer TEST_KEY" {
				t.Errorf("run submission authorization = %q, want fake local key", got)
				http.Error(w, "unexpected authorization", http.StatusUnauthorized)
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode run request: %v", err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			createBodiesMu.Lock()
			createBodies = append(createBodies, body)
			createBodiesMu.Unlock()
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"data":{"id":123,"status":"QUEUED","repositoryName":"test/repo","runType":"run"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	baseEnv := func(apiKey string) map[string]string {
		homeDir := SetupTestConfig(t)
		return map[string]string{
			"HOME":               homeDir,
			"XDG_CONFIG_HOME":    filepath.Join(homeDir, ".config"),
			"REPOBIRD_API_URL":   server.URL,
			"REPOBIRD_API_KEY":   apiKey,
			"REPOBIRD_TEST_MODE": "true",
			"NO_COLOR":           "true",
		}
	}

	human := RunCommandWithEnv(t, baseEnv(""), "models")
	AssertSuccess(t, human)
	AssertContains(t, human.Stdout, "future-provider/future-model-v9")
	AssertContains(t, human.Stdout, "Future Provider")
	AssertContains(t, human.Stdout, "futureCatalogField")
	AssertContains(t, human.Stdout, "preserve-me")
	AssertContains(t, human.Stdout, "futureDescriptorField")
	AssertContains(t, human.Stdout, "displayHint")
	AssertNotContains(t, human.Stdout, "policyHash")
	AssertNotContains(t, human.Stdout, "workerProfile")
	AssertNotContains(t, human.Stdout, "credentialId")
	AssertNotContains(t, human.Stdout, "omit-me")

	machine := RunCommandWithEnv(t, baseEnv(""), "models", "--json")
	AssertSuccess(t, machine)
	assertStdoutIsSingleJSONValue(t, machine)
	var catalog map[string]any
	if err := json.Unmarshal([]byte(machine.Stdout), &catalog); err != nil {
		t.Fatalf("decode models JSON: %v", err)
	}
	encodedCatalog, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("re-encode models JSON: %v", err)
	}
	AssertContains(t, string(encodedCatalog), "futureCatalogField")
	AssertContains(t, string(encodedCatalog), "futureDescriptorField")
	AssertNotContains(t, string(encodedCatalog), "policyHash")
	AssertNotContains(t, string(encodedCatalog), "workerProfile")
	AssertNotContains(t, string(encodedCatalog), "credentialId")

	for _, preset := range []string{"basic", "pro"} {
		result := RunCommandWithEnv(t, baseEnv("TEST_KEY"), preset, "Preset submission", "--repo", "test/repo", "--acknowledge-prompt-risk", "--force")
		AssertSuccess(t, result)
	}
	explicit := RunCommandWithEnv(t, baseEnv("TEST_KEY"), "run", "--repo", "test/repo", "--prompt", "Explicit submission", "--opencode-model", "future-provider/future-model-v9", "--opencode-provider", "future-provider", "--acknowledge-prompt-risk", "--force")
	AssertSuccess(t, explicit)

	createBodiesMu.Lock()
	capturedBodies := append([]map[string]any(nil), createBodies...)
	createBodiesMu.Unlock()
	if len(capturedBodies) != 3 {
		t.Fatalf("captured %d create requests, want 3", len(capturedBodies))
	}
	for _, body := range capturedBodies[:2] {
		if _, ok := body["opencodeModel"]; ok {
			t.Fatalf("preset request unexpectedly selected a model: %#v", body)
		}
		if _, ok := body["opencodeProvider"]; ok {
			t.Fatalf("preset request unexpectedly selected a provider: %#v", body)
		}
	}
	if got := capturedBodies[2]["opencodeModel"]; got != "future-provider/future-model-v9" {
		t.Fatalf("explicit opencodeModel = %#v", got)
	}
	if got := capturedBodies[2]["opencodeProvider"]; got != "future-provider" {
		t.Fatalf("explicit opencodeProvider = %#v", got)
	}
	if strings.Contains(explicit.Stdout, "fallback") || strings.Contains(explicit.Stderr, "fallback") {
		t.Fatal("explicit submission reported a client fallback")
	}
}
