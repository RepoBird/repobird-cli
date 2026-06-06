package repository

import (
	"context"
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
