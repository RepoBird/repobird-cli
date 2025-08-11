package api

import "fmt"

// API endpoints constants
const (
	// EndpointRuns is the base endpoint for runs operations
	EndpointRuns = "/api/v1/runs"

	// EndpointRunDetails is the endpoint template for getting a specific run
	EndpointRunDetailsTemplate = "/api/v1/runs/%s"

	// EndpointRunsList is the endpoint template for listing runs with pagination
	EndpointRunsListTemplate = "/api/v1/runs?limit=%d&offset=%d"

	// EndpointAuthVerify is the endpoint for verifying authentication
	EndpointAuthVerify = "/api/v1/auth/verify"

	// EndpointRepositories is the endpoint for listing repositories
	EndpointRepositories = "/api/v1/repositories"

	// EndpointUser is the endpoint for getting user information
	EndpointUser = "/api/v1/user"

	// EndpointRunsHashes is the endpoint for getting all file hashes
	EndpointRunsHashes = "/api/v1/runs/hashes"
)

// RunDetailsURL builds the URL for getting a specific run by ID
func RunDetailsURL(id string) string {
	return fmt.Sprintf(EndpointRunDetailsTemplate, id)
}

// RunsListURL builds the URL for listing runs with pagination
func RunsListURL(limit, offset int) string {
	return fmt.Sprintf(EndpointRunsListTemplate, limit, offset)
}
