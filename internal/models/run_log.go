// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package models

import "encoding/json"

// RunLogMessage is one NDJSON record from the agent conversation log endpoint.
type RunLogMessage struct {
	ID         string         `json:"id,omitempty"`
	Type       string         `json:"type"`
	Content    string         `json:"content"`
	IsError    bool           `json:"isError,omitempty"`
	ToolName   string         `json:"toolName,omitempty"`
	ToolParams string         `json:"toolParams,omitempty"`
	ToolResult string         `json:"toolResult,omitempty"`
	Cost       *float64       `json:"cost,omitempty"`
	Duration   *float64       `json:"duration,omitempty"`
	Tokens     map[string]any `json:"tokens,omitempty"`
	Raw        map[string]any `json:"-"`
}

func (m *RunLogMessage) UnmarshalJSON(data []byte) error {
	type alias RunLogMessage
	var parsed alias
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	*m = RunLogMessage(parsed)
	m.Raw = raw
	return nil
}

func (m RunLogMessage) MarshalJSON() ([]byte, error) {
	if len(m.Raw) == 0 {
		type alias RunLogMessage
		return json.Marshal(alias(m))
	}

	raw := make(map[string]any, len(m.Raw)+9)
	for key, value := range m.Raw {
		raw[key] = value
	}
	if m.ID != "" {
		raw["id"] = m.ID
	}
	if m.Type != "" {
		raw["type"] = m.Type
	}
	if m.Content != "" {
		raw["content"] = m.Content
	}
	if m.IsError {
		raw["isError"] = m.IsError
	}
	if m.ToolName != "" {
		raw["toolName"] = m.ToolName
	}
	if m.ToolParams != "" {
		raw["toolParams"] = m.ToolParams
	}
	if m.ToolResult != "" {
		raw["toolResult"] = m.ToolResult
	}
	if m.Cost != nil {
		raw["cost"] = *m.Cost
	}
	if m.Duration != nil {
		raw["duration"] = *m.Duration
	}
	if m.Tokens != nil {
		raw["tokens"] = m.Tokens
	}

	return json.Marshal(raw)
}
