package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/repobird/repobird-cli/internal/models"
)

// CalculateConfigHash calculates the hash of a RunConfig
func CalculateConfigHash(config *models.RunConfig) (string, error) {
	if config == nil {
		return "", fmt.Errorf("config is nil")
	}

	// Create a normalized version for hashing
	normalized := struct {
		Prompt     string `json:"prompt"`
		Repository string `json:"repository"`
		Source     string `json:"source"`
		Target     string `json:"target"`
		Context    string `json:"context,omitempty"`
		Title      string `json:"title,omitempty"`
	}{
		Prompt:     config.Prompt,
		Repository: config.Repository,
		Source:     config.Source,
		Target:     config.Target,
		Context:    config.Context,
		Title:      config.Title,
	}

	jsonData, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(jsonData)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CalculateFileHashFromPath calculates the hash of a file at the given path
func CalculateFileHashFromPath(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	h := sha256.New()
	h.Write(content)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// NewFileHashCache creates a simple file hash cache (compatibility function)
func NewFileHashCache() map[string]string {
	return make(map[string]string)
}
