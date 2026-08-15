package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestAPIRunRepositoryCreateRunTypeDoesNotSelectModelPolicy(t *testing.T) {
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
			require.NotContains(t, body, "opencodeModel")
			require.NotContains(t, body, "opencodeProvider")
		})
	}
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
