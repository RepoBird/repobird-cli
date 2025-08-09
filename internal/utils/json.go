package utils

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON marshals an interface to JSON with standardized error handling
func MarshalJSON(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return data, nil
}

// MarshalJSONIndent marshals an interface to pretty-printed JSON with standardized error handling
func MarshalJSONIndent(v interface{}, prefix, indent string) ([]byte, error) {
	data, err := json.MarshalIndent(v, prefix, indent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return data, nil
}

// UnmarshalJSON unmarshals JSON data into an interface with standardized error handling
func UnmarshalJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	return nil
}

// MarshalJSONToString marshals an interface to a JSON string
func MarshalJSONToString(v interface{}) (string, error) {
	data, err := MarshalJSON(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MarshalJSONIndentToString marshals an interface to a pretty-printed JSON string
func MarshalJSONIndentToString(v interface{}) (string, error) {
	data, err := MarshalJSONIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
