// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/pkg/version"
)

const MaxRunLogMessageBytes = 10 * 1024 * 1024

// OpenRunLogs opens the agent log NDJSON response for a run.
func (c *Client) OpenRunLogs(ctx context.Context, id string, afterSeq int) (io.ReadCloser, error) {
	if id == "" {
		return nil, fmt.Errorf("run ID cannot be empty")
	}

	path := RunLogsURL(id, afterSeq)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/x-ndjson")
	req.Header.Set("User-Agent", fmt.Sprintf("repobird-cli/%s", version.GetVersion()))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &errors.NetworkError{
			Err:       err,
			Operation: fmt.Sprintf("GET %s", path),
			URL:       c.baseURL + path,
		}
	}

	if err := ValidateResponseOK(resp); err != nil {
		_ = resp.Body.Close()
		return nil, err
	}

	return resp.Body, nil
}

// GetRunLogs fetches and decodes the current agent log snapshot.
func (c *Client) GetRunLogs(ctx context.Context, id string, afterSeq int) ([]models.RunLogMessage, error) {
	body, err := c.OpenRunLogs(ctx, id, afterSeq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = body.Close() }()

	var messages []models.RunLogMessage
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), MaxRunLogMessageBytes)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var message models.RunLogMessage
		if err := json.Unmarshal(line, &message); err != nil {
			return nil, fmt.Errorf("failed to decode log message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read log messages: %w", err)
	}

	return messages, nil
}
