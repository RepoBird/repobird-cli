package api

import (
	"io"
	"net/http"

	"github.com/repobird/repobird-cli/internal/errors"
)

// ValidateResponse validates HTTP response status codes and returns an error if not valid
func ValidateResponse(resp *http.Response, allowedCodes ...int) error {
	if len(allowedCodes) == 0 {
		allowedCodes = []int{http.StatusOK}
	}

	for _, code := range allowedCodes {
		if resp.StatusCode == code {
			return nil
		}
	}

	body, _ := io.ReadAll(resp.Body)
	return errors.ParseAPIError(resp.StatusCode, body)
}

// ValidateResponseOK validates that the response status is 200 OK
func ValidateResponseOK(resp *http.Response) error {
	return ValidateResponse(resp, http.StatusOK)
}

// ValidateResponseOKOrCreated validates that the response status is either 200 OK or 201 Created
func ValidateResponseOKOrCreated(resp *http.Response) error {
	return ValidateResponse(resp, http.StatusOK, http.StatusCreated)
}
