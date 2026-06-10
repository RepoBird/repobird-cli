// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRunLogsDecodesNDJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/issue-runs/run_123/agent-logs" {
			t.Fatalf("expected logs path, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("afterSeq") != "" {
			t.Fatalf("expected no afterSeq query, got %s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("expected authorization header, got %q", r.Header.Get("Authorization"))
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"id":"msg-1","type":"assistant","content":"hello"}` + "\n"))
		_, _ = w.Write([]byte(`{"id":"tool-1","type":"tool_call","content":"Bash","toolName":"Bash","toolResult":"ok"}` + "\n"))
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)
	messages, err := client.GetRunLogs(context.Background(), "run_123", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	if messages[0].Type != "assistant" || messages[0].Content != "hello" {
		t.Fatalf("unexpected first message: %#v", messages[0])
	}
	if messages[1].ToolName != "Bash" || messages[1].ToolResult != "ok" {
		t.Fatalf("unexpected tool message: %#v", messages[1])
	}
}

func TestOpenRunLogsUsesAfterSeq(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/issue-runs/123/agent-logs" {
			t.Fatalf("expected logs path, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("afterSeq") != "37" {
			t.Fatalf("expected afterSeq=37, got %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"type":    "assistant",
			"content": "new chunk",
		})
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL, false)
	body, err := client.OpenRunLogs(context.Background(), "123", 37)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = body.Close() }()

	var message map[string]string
	if err := json.NewDecoder(body).Decode(&message); err != nil {
		t.Fatalf("failed to decode streamed message: %v", err)
	}
	if message["content"] != "new chunk" {
		t.Fatalf("unexpected message: %#v", message)
	}
}

func TestGetRunLogsRequiresRunID(t *testing.T) {
	client := NewClient("test-key", "https://example.test", false)
	_, err := client.GetRunLogs(context.Background(), "", 0)
	if err == nil {
		t.Fatal("expected missing run ID error")
	}
}
