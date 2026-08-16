package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/repobird/repobird-cli/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestAPIRunRepositoryCreateSendsIdempotencyKey(t *testing.T) {
	repo := NewAPIRunRepository(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "run-key-123", req.Header.Get("Idempotency-Key"))

		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, "run-key-123", body["idempotencyKey"])

		return &http.Response{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": 123,
					"status": "QUEUED"
				}
			}`)),
			Header: make(http.Header),
		}, nil
	}), "https://example.test", "test-key", false)

	_, err := repo.Create(context.Background(), domain.CreateRunRequest{
		Prompt:         "Fix auth",
		RepositoryName: "acme/webapp",
		RunType:        domain.RunTypeRun,
		IdempotencyKey: "run-key-123",
	})
	require.NoError(t, err)
}

func TestAPIRunRepositoryCreateSendsGitLabFields(t *testing.T) {
	repo := NewAPIRunRepository(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var body map[string]any
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, "cred_123", body["providerCredentialId"])
		require.Equal(t, "byok-user", body["providerMode"])

		gitlabCredential, ok := body["gitlabCredential"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "stored_token_reference", gitlabCredential["mode"])
		require.Equal(t, "glref_123", gitlabCredential["tokenReferenceId"])

		return &http.Response{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": 123,
					"status": "QUEUED"
				}
			}`)),
			Header: make(http.Header),
		}, nil
	}), "https://example.test", "test-key", false)

	_, err := repo.Create(context.Background(), domain.CreateRunRequest{
		Prompt:               "Fix GitLab task",
		RepositoryName:       "acme/webapp",
		RunType:              domain.RunTypeRun,
		ProviderCredentialID: "cred_123",
		ProviderMode:         "byok-user",
		GitLabCredential: &domain.GitLabCredentialRequest{
			Mode:             "stored_token_reference",
			TokenReferenceID: "glref_123",
		},
	})
	require.NoError(t, err)
}

func TestAPIRunRepositoryCreateReasoningEffortRequestContract(t *testing.T) {
	tests := []struct {
		name            string
		reasoningEffort string
		wantPresent     bool
	}{
		{name: "explicit unfamiliar value", reasoningEffort: "future-adaptive", wantPresent: true},
		{name: "omitted", reasoningEffort: "", wantPresent: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "/api/v1/runs", req.URL.Path)

				var body map[string]any
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
				value, present := body["reasoningEffort"]
				require.Equal(t, tt.wantPresent, present)
				if tt.wantPresent {
					require.Equal(t, tt.reasoningEffort, value)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(w, `{"data":{"id":123,"status":"QUEUED"}}`)
			}))
			defer server.Close()

			repo := NewAPIRunRepository(server.Client(), server.URL, "test-key", false)
			_, err := repo.Create(context.Background(), domain.CreateRunRequest{
				Prompt:          "Fix auth",
				RepositoryName:  "acme/webapp",
				RunType:         domain.RunTypeRun,
				ReasoningEffort: tt.reasoningEffort,
			})
			require.NoError(t, err)
		})
	}
}

func TestAPIRunRepositoryCreateRunTypePreservesExplicitModelSelection(t *testing.T) {
	for _, runType := range []string{domain.RunTypeBasic, domain.RunTypePro} {
		t.Run(runType, func(t *testing.T) {
			var body map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				require.NoError(t, json.NewDecoder(req.Body).Decode(&body))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = io.WriteString(w, `{"data":{"id":123,"status":"QUEUED"}}`)
			}))
			defer server.Close()

			repo := NewAPIRunRepository(server.Client(), server.URL, "test-key", false)
			_, err := repo.Create(context.Background(), domain.CreateRunRequest{
				Prompt:           "Fix auth",
				RepositoryName:   "acme/webapp",
				RunType:          runType,
				OpenCodeModel:    "client-selected-model",
				OpenCodeProvider: "client-selected-provider",
			})
			require.NoError(t, err)
			require.Equal(t, runType, body["runType"])
			require.Equal(t, "client-selected-model", body["opencodeModel"])
			require.Equal(t, "client-selected-provider", body["opencodeProvider"])
		})
	}
}

func TestAPIRunRepositoryCreatePassesExplicitOpenCodeSelection(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"data":{"id":123,"status":"QUEUED"}}`)
	}))
	defer server.Close()

	repo := NewAPIRunRepository(server.Client(), server.URL, "test-key", false)
	_, err := repo.Create(context.Background(), domain.CreateRunRequest{
		Prompt:           "Fix auth",
		RepositoryName:   "acme/webapp",
		RunType:          domain.RunTypePro,
		OpenCodeModel:    "future-provider/future-model-v9",
		OpenCodeProvider: "future-provider",
	})
	require.NoError(t, err)
	require.Equal(t, "future-provider/future-model-v9", body["opencodeModel"])
	require.Equal(t, "future-provider", body["opencodeProvider"])
}

func TestAPIRunRepositoryCreateReturnsLocal400WithoutRetryOrFallback(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"VALIDATION_ERROR","message":"reasoning effort rejected","details":{"field":"reasoningEffort"}}`)
	}))
	defer server.Close()

	repo := NewAPIRunRepository(server.Client(), server.URL, "test-key", false)
	run, err := repo.Create(context.Background(), domain.CreateRunRequest{
		Prompt:          "Fix auth",
		RepositoryName:  "acme/webapp",
		RunType:         domain.RunTypeRun,
		ReasoningEffort: "service-rejected-value",
	})

	require.Nil(t, run)
	require.EqualError(t, err, "validation error for field 'reasoningEffort': reasoning effort rejected")
	require.Equal(t, 1, attempts)
}

func TestAPIRunRepositoryCreateDoesNotFallbackAfterModelRejection(t *testing.T) {
	attempts := 0
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		attempts++
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":"MODEL_NOT_SUPPORTED","message":"model rejected"}`)
	}))
	defer server.Close()

	repo := NewAPIRunRepository(server.Client(), server.URL, "test-key", false)
	run, err := repo.Create(context.Background(), domain.CreateRunRequest{
		Prompt:           "Fix auth",
		RepositoryName:   "acme/webapp",
		RunType:          domain.RunTypePro,
		OpenCodeModel:    "future-provider/future-model-v9",
		OpenCodeProvider: "future-provider",
	})

	require.Nil(t, run)
	require.ErrorContains(t, err, "model rejected")
	require.Equal(t, 1, attempts)
	require.Equal(t, "future-provider/future-model-v9", body["opencodeModel"])
	require.Equal(t, "future-provider", body["opencodeProvider"])
}

func TestAPIRunRepositoryListUsesPageBasedPagination(t *testing.T) {
	var requestedURL string
	repo := NewAPIRunRepository(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requestedURL = req.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"data":[],"metadata":{"currentPage":3,"total":0,"totalPages":0}}`)),
			Header:     make(http.Header),
		}, nil
	}), "https://example.test", "test-key", false)

	_, err := repo.List(context.Background(), domain.ListOptions{Limit: 25, Offset: 50})
	require.NoError(t, err)
	require.Equal(t, "https://example.test/api/v1/runs?limit=25&page=3", requestedURL)
}

func TestAPIRunRepositoryCreatePreservesCanonicalBranchFields(t *testing.T) {
	repo := NewAPIRunRepository(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/api/v1/runs", req.URL.Path)

		return &http.Response{
			StatusCode: http.StatusCreated,
			Body: io.NopCloser(strings.NewReader(`{
				"data": {
					"id": 123,
					"publicId": "run_123e4567-e89b-12d3-a456-426614174000",
					"status": "QUEUED",
					"baseBranch": "main",
					"outputMode": "pull_request",
					"outputBranch": "repobird/fix-auth",
					"prTargetBranch": "release",
					"outputBranchPolicy": "create"
				}
			}`)),
			Header: make(http.Header),
		}, nil
	}), "https://example.test", "test-key", false)

	run, err := repo.Create(context.Background(), domain.CreateRunRequest{
		Prompt:         "Fix auth",
		RepositoryName: "acme/webapp",
		RunType:        domain.RunTypeRun,
	})
	require.NoError(t, err)
	require.NotNil(t, run)

	require.Equal(t, "123", run.ID)
	require.Equal(t, domain.StatusQueued, run.Status)
	require.Equal(t, "acme/webapp", run.RepositoryName)
	require.Equal(t, "main", run.BaseBranch)
	require.Equal(t, "pull_request", run.OutputMode)
	require.Equal(t, "repobird/fix-auth", run.OutputBranch)
	require.Equal(t, "release", run.PRTargetBranch)
	require.Equal(t, "create", run.OutputBranchPolicy)
	require.Equal(t, "run_123e4567-e89b-12d3-a456-426614174000", run.PublicID)
}

func TestAPIRunRepositoryGetPropagatesResponsePolicy(t *testing.T) {
	fixtures := loadReasoningEffortRunFixtures(t)
	tests := []struct {
		name                string
		fixture             string
		wantModel           string
		wantReasoningEffort string
	}{
		{
			name:                "complete",
			fixture:             "complete",
			wantModel:           "openrouter/z-ai/glm-5.2",
			wantReasoningEffort: "high",
		},
		{
			name:      "partial model only",
			fixture:   "partialModelOnly",
			wantModel: "openrouter/deepseek/deepseek-v4-flash",
		},
		{
			name:                "partial reasoning effort only",
			fixture:             "partialReasoningEffortOnly",
			wantReasoningEffort: "max",
		},
		{
			name:    "absent legacy response does not reconstruct policy from run type",
			fixture: "absentLegacy",
		},
		{
			name:                "unfamiliar future values",
			fixture:             "unfamiliarFutureValues",
			wantModel:           "future-provider/future-model-v9",
			wantReasoningEffort: "adaptive-ultra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "/api/v1/runs/run-under-test", req.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(fixtures[tt.fixture])
			}))
			defer server.Close()

			repo := NewAPIRunRepository(server.Client(), server.URL, "test-key", false)
			run, err := repo.Get(context.Background(), "run-under-test")
			require.NoError(t, err)
			require.Equal(t, tt.wantModel, run.Model)
			require.Equal(t, tt.wantReasoningEffort, run.ReasoningEffort)
		})
	}
}

func TestAPIRunRepositoryListPropagatesResponsePolicy(t *testing.T) {
	fixtures := loadReasoningEffortRunFixtures(t)
	fixtureNames := []string{
		"complete",
		"partialModelOnly",
		"partialReasoningEffortOnly",
		"absentLegacy",
		"unfamiliarFutureValues",
	}

	data := make([]json.RawMessage, 0, len(fixtureNames))
	for _, name := range fixtureNames {
		var fixture struct {
			Data json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(fixtures[name], &fixture))
		data = append(data, fixture.Data)
	}

	body, err := json.Marshal(map[string]any{
		"data": data,
		"metadata": map[string]int{
			"currentPage": 1,
			"total":       len(data),
			"totalPages":  1,
		},
	})
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, http.MethodGet, req.Method)
		require.Equal(t, "/api/v1/runs", req.URL.Path)
		require.Equal(t, "5", req.URL.Query().Get("limit"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	repo := NewAPIRunRepository(server.Client(), server.URL, "test-key", false)
	runs, err := repo.List(context.Background(), domain.ListOptions{Limit: len(data)})
	require.NoError(t, err)
	require.Len(t, runs, len(data))

	want := []struct {
		model           string
		reasoningEffort string
	}{
		{model: "openrouter/z-ai/glm-5.2", reasoningEffort: "high"},
		{model: "openrouter/deepseek/deepseek-v4-flash"},
		{reasoningEffort: "max"},
		{},
		{model: "future-provider/future-model-v9", reasoningEffort: "adaptive-ultra"},
	}
	for i, expected := range want {
		require.Equal(t, expected.model, runs[i].Model, "run %d model", i)
		require.Equal(t, expected.reasoningEffort, runs[i].ReasoningEffort, "run %d reasoning effort", i)
	}
}

func loadReasoningEffortRunFixtures(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	body, err := os.ReadFile("../api/dto/testdata/reasoning_effort_runs.json")
	require.NoError(t, err)

	var fixtures map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &fixtures))
	return fixtures
}
