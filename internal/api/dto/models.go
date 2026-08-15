// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// PublicModelsResponse is the public, policy-sanitized model catalog response.
// AdditionalFields retains safe additive fields from newer servers without
// making internal policy data part of the CLI's public output.
type PublicModelsResponse struct {
	Data             *PublicModelsData          `json:"data"`
	AdditionalFields map[string]json.RawMessage `json:"-"`
}

// PublicModelsData contains the public catalog revision and descriptors.
type PublicModelsData struct {
	CatalogRevision  string                     `json:"catalogRevision"`
	Models           []PublicModelDescriptor    `json:"models"`
	AdditionalFields map[string]json.RawMessage `json:"-"`
}

// PublicModelDescriptor describes one selectable model using only public
// catalog fields. Provider and capability values remain opaque strings so the
// CLI stays forward-compatible with server additions.
type PublicModelDescriptor struct {
	ID               string                     `json:"id"`
	Label            string                     `json:"label"`
	Provider         string                     `json:"provider"`
	Capabilities     []string                   `json:"capabilities"`
	IsDefault        bool                       `json:"isDefault"`
	AdditionalFields map[string]json.RawMessage `json:"-"`
}

// Validate checks the required public envelope without applying model
// admission or inventing a local catalog policy.
func (r *PublicModelsResponse) Validate() error {
	if r == nil || r.Data == nil {
		return fmt.Errorf("models response is missing data")
	}
	if r.Data.CatalogRevision == "" {
		return fmt.Errorf("models response is missing catalogRevision")
	}
	for i, model := range r.Data.Models {
		if model.ID == "" {
			return fmt.Errorf("models response model %d is missing id", i)
		}
		if model.Label == "" {
			return fmt.Errorf("models response model %d is missing label", i)
		}
		if model.Provider == "" {
			return fmt.Errorf("models response model %d is missing provider", i)
		}
	}
	return nil
}

func (r *PublicModelsResponse) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	r.Data = nil
	if raw, ok := fields["data"]; ok && !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		var catalog PublicModelsData
		if err := json.Unmarshal(raw, &catalog); err != nil {
			return fmt.Errorf("decode models data: %w", err)
		}
		r.Data = &catalog
	}
	r.AdditionalFields = safeAdditionalFields(fields, "data")
	return nil
}

func (r PublicModelsResponse) MarshalJSON() ([]byte, error) {
	fields := cloneAdditionalFields(r.AdditionalFields)
	data, err := json.Marshal(r.Data)
	if err != nil {
		return nil, err
	}
	fields["data"] = data
	return json.Marshal(fields)
}

func (d *PublicModelsData) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	if raw, ok := fields["catalogRevision"]; ok {
		if err := json.Unmarshal(raw, &d.CatalogRevision); err != nil {
			return fmt.Errorf("decode catalogRevision: %w", err)
		}
	}
	if raw, ok := fields["models"]; ok {
		if err := json.Unmarshal(raw, &d.Models); err != nil {
			return fmt.Errorf("decode models: %w", err)
		}
	}
	d.AdditionalFields = safeAdditionalFields(fields, "catalogRevision", "models")
	return nil
}

func (d PublicModelsData) MarshalJSON() ([]byte, error) {
	fields := cloneAdditionalFields(d.AdditionalFields)
	fields["catalogRevision"] = json.RawMessage(mustMarshal(d.CatalogRevision))
	fields["models"] = json.RawMessage(mustMarshal(d.Models))
	return json.Marshal(fields)
}

func (m *PublicModelDescriptor) UnmarshalJSON(data []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}

	if raw, ok := fields["id"]; ok {
		if err := json.Unmarshal(raw, &m.ID); err != nil {
			return fmt.Errorf("decode model id: %w", err)
		}
	}
	if raw, ok := fields["label"]; ok {
		if err := json.Unmarshal(raw, &m.Label); err != nil {
			return fmt.Errorf("decode model label: %w", err)
		}
	}
	if raw, ok := fields["provider"]; ok {
		if err := json.Unmarshal(raw, &m.Provider); err != nil {
			return fmt.Errorf("decode model provider: %w", err)
		}
	}
	if raw, ok := fields["capabilities"]; ok {
		if err := json.Unmarshal(raw, &m.Capabilities); err != nil {
			return fmt.Errorf("decode model capabilities: %w", err)
		}
	}
	if raw, ok := fields["isDefault"]; ok {
		if err := json.Unmarshal(raw, &m.IsDefault); err != nil {
			return fmt.Errorf("decode model isDefault: %w", err)
		}
	}
	m.AdditionalFields = safeAdditionalFields(fields, "id", "label", "provider", "capabilities", "isDefault")
	return nil
}

func (m PublicModelDescriptor) MarshalJSON() ([]byte, error) {
	fields := cloneAdditionalFields(m.AdditionalFields)
	fields["id"] = json.RawMessage(mustMarshal(m.ID))
	fields["label"] = json.RawMessage(mustMarshal(m.Label))
	fields["provider"] = json.RawMessage(mustMarshal(m.Provider))
	fields["capabilities"] = json.RawMessage(mustMarshal(m.Capabilities))
	fields["isDefault"] = json.RawMessage(mustMarshal(m.IsDefault))
	return json.Marshal(fields)
}

func safeAdditionalFields(fields map[string]json.RawMessage, known ...string) map[string]json.RawMessage {
	knownFields := make(map[string]struct{}, len(known))
	for _, field := range known {
		knownFields[field] = struct{}{}
	}

	result := make(map[string]json.RawMessage)
	for name, raw := range fields {
		if _, ok := knownFields[name]; ok || !safePublicFieldName(name) {
			continue
		}
		if sanitized, ok := sanitizePublicValue(raw); ok {
			result[name] = sanitized
		}
	}
	return result
}

func cloneAdditionalFields(fields map[string]json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage, len(fields))
	for name, raw := range fields {
		if !safePublicFieldName(name) {
			continue
		}
		if sanitized, ok := sanitizePublicValue(raw); ok {
			result[name] = sanitized
		}
	}
	return result
}

func safePublicFieldName(name string) bool {
	normalized := strings.ToLower(name)
	for _, fragment := range []string{
		"credential", "secret", "token", "tenant", "worker", "policy", "availability", "dispatch", "hash", "internal",
	} {
		if strings.Contains(normalized, fragment) {
			return false
		}
	}
	return true
}

func sanitizePublicValue(raw json.RawMessage) (json.RawMessage, bool) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false
	}
	sanitized, ok := sanitizePublicValueAny(value)
	if !ok {
		return nil, false
	}
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return nil, false
	}
	return encoded, true
}

func sanitizePublicValueAny(value any) (any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(typed))
		for name, child := range typed {
			if !safePublicFieldName(name) {
				continue
			}
			sanitized, ok := sanitizePublicValueAny(child)
			if ok {
				result[name] = sanitized
			}
		}
		return result, true
	case []any:
		result := make([]any, 0, len(typed))
		for _, child := range typed {
			sanitized, ok := sanitizePublicValueAny(child)
			if ok {
				result = append(result, sanitized)
			}
		}
		return result, true
	default:
		return value, true
	}
}

func mustMarshal(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}
