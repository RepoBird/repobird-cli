// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import "fmt"

// API endpoints constants
const (
	// EndpointRuns is the base endpoint for runs operations
	EndpointRuns = "/api/v1/runs"

	// EndpointRunDetails is the endpoint template for getting a specific run
	EndpointRunDetailsTemplate = "/api/v1/runs/%s"

	// EndpointRunsList is the endpoint template for listing runs with pagination
	EndpointRunsListTemplate = "/api/v1/runs?page=%d&limit=%d"

	// EndpointAuthVerify is the endpoint for verifying authentication
	EndpointAuthVerify = "/api/v1/auth/verify"

	// EndpointRepositories is the endpoint for listing repositories
	EndpointRepositories = "/api/v1/repositories"

	// EndpointRepoDetailsTemplate is the endpoint template for repository details and settings updates.
	EndpointRepoDetailsTemplate = "/api/repos/%s"

	// EndpointUser is the endpoint for getting user information
	EndpointUser = "/api/v1/user"

	// EndpointRunsHashes is the endpoint for getting all file hashes
	EndpointRunsHashes = "/api/v1/runs/hashes"

	// EndpointBulkRuns is the endpoint for bulk run operations
	EndpointBulkRuns = "/api/v1/runs/bulk"
)

// RunDetailsURL builds the URL for getting a specific run by ID
func RunDetailsURL(id string) string {
	return fmt.Sprintf(EndpointRunDetailsTemplate, id)
}

// RunsListURL builds the URL for listing runs with pagination.
// The legacy offset argument is converted to the page-based API contract.
func RunsListURL(limit, offset int) string {
	page := 1
	if limit > 0 && offset > 0 {
		page = (offset / limit) + 1
	}
	return RunsPageURL(page, limit)
}

// RunsPageURL builds the URL for the current page-based API contract.
func RunsPageURL(page, limit int) string {
	if page < 1 {
		page = 1
	}
	return fmt.Sprintf(EndpointRunsListTemplate, page, limit)
}

// RepositoryDetailsURL builds the URL for repository details and updates.
func RepositoryDetailsURL(id string) string {
	return fmt.Sprintf(EndpointRepoDetailsTemplate, id)
}
