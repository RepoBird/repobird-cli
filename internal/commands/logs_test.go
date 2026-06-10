// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/repobird/repobird-cli/internal/models"
)

func TestLogsCommandRequiresRunID(t *testing.T) {
	err := logsCmd.Args(logsCmd, []string{})
	if err == nil {
		t.Fatal("expected missing run ID error")
	}
}

func TestRenderRunLogsHumanOutput(t *testing.T) {
	var out bytes.Buffer
	messages := []models.RunLogMessage{
		{Type: "user", Content: "Fix the bug"},
		{Type: "assistant", Content: "I'll inspect the failure."},
		{Type: "tool_call", ToolName: "Bash", ToolParams: "go test ./...", ToolResult: "ok"},
		{Type: "error", Content: "Agent session not found.", IsError: true},
	}

	if err := renderRunLogs(&out, messages, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"[user] Fix the bug",
		"[assistant] I'll inspect the failure.",
		"[tool] Bash",
		"go test ./...",
		"ok",
		"[error] Agent session not found.",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestRenderRunLogsJSONOutput(t *testing.T) {
	var out bytes.Buffer
	messages := []models.RunLogMessage{
		{
			ID:      "msg-1",
			Type:    "assistant",
			Content: "hello",
			Raw: map[string]any{
				"content": "hello",
				"custom":  "preserved",
				"id":      "msg-1",
				"type":    "assistant",
			},
		},
	}

	if err := renderRunLogs(&out, messages, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded []models.RunLogMessage
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", out.String(), err)
	}
	if len(decoded) != 1 || decoded[0].Content != "hello" {
		t.Fatalf("unexpected decoded output: %#v", decoded)
	}
	if decoded[0].Raw["custom"] != "preserved" {
		t.Fatalf("expected raw custom field to be preserved, got %#v", decoded[0].Raw)
	}
}

func TestWriteFollowLogLineAdvancesCursor(t *testing.T) {
	var out bytes.Buffer
	cursor := 2

	next, wrote, err := writeFollowLogLine(&out, []byte(`{"id":"live-000003.jsonl-0","type":"assistant","content":"new"}`), cursor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !wrote {
		t.Fatal("expected line to be written")
	}
	if next != 3 {
		t.Fatalf("expected cursor 3, got %d", next)
	}
	if got := strings.TrimSpace(out.String()); got != `{"id":"live-000003.jsonl-0","type":"assistant","content":"new"}` {
		t.Fatalf("unexpected output line: %q", got)
	}
}
