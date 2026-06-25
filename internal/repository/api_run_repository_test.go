package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
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
