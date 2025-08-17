// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "RepoBird-CLI"
	keyringAccount = "api-key"
)

// Security Strategy:
// 1. Environment variable (REPOBIRD_API_KEY) - Best for CI/CD, containers
// 2. System keyring - Only on macOS/Windows or Linux with desktop environment
// 3. Encrypted file (AES-256-GCM) - Universal fallback, works everywhere
//
// On Linux servers/containers, we default to encrypted file storage
// which is more reliable than trying to use desktop keyrings that
// may not be available or properly configured.

// SecureStorage handles secure storage of sensitive data
type SecureStorage struct {
	useKeyring bool
	configDir  string
}

// NewSecureStorage creates a new secure storage instance
func NewSecureStorage() *SecureStorage {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".repobird")

	// Check if keyring is available
	useKeyring := isKeyringAvailable()

	return &SecureStorage{
		useKeyring: useKeyring,
		configDir:  configDir,
	}
}

// SaveAPIKey securely stores the API key
func (s *SecureStorage) SaveAPIKey(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Try to use system keyring first
	if s.useKeyring {
		err := keyring.Set(keyringService, keyringAccount, apiKey)
		if err == nil {
			// Successfully stored in keyring, remove from config file if exists
			s.removeAPIKeyFromConfig()
			return nil
		}
		// Fall back to encrypted file if keyring fails
	}

	// Fall back to encrypted file storage
	return s.saveEncryptedAPIKey(apiKey)
}

// GetAPIKey retrieves the API key from secure storage
func (s *SecureStorage) GetAPIKey() (string, error) {
	// Check environment variable first (highest priority)
	if envKey := os.Getenv(EnvAPIKey); envKey != "" {
		return envKey, nil
	}

	// Try system keyring
	if s.useKeyring {
		apiKey, err := keyring.Get(keyringService, keyringAccount)
		if err == nil && apiKey != "" {
			return apiKey, nil
		}
	}

	// Try encrypted file
	apiKey, err := s.getEncryptedAPIKey()
	if err == nil && apiKey != "" {
		return apiKey, nil
	}

	// Check plain text config as last resort (for backward compatibility)
	apiKey = s.getPlainTextAPIKey()
	if apiKey != "" {
		// Migrate to secure storage
		if err := s.SaveAPIKey(apiKey); err != nil {
			// Log migration failure but continue - API key is still available
			fmt.Fprintf(os.Stderr, "Warning: failed to migrate API key to secure storage: %v\n", err)
		}
		return apiKey, nil
	}

	return "", fmt.Errorf("API key not found. Please run 'rb config set api-key YOUR_KEY'")
}

// DeleteAPIKey removes the API key from all storage locations
func (s *SecureStorage) DeleteAPIKey() error {
	var errors []string
	var removedAny bool

	// Remove from keyring
	if s.useKeyring {
		if err := keyring.Delete(keyringService, keyringAccount); err != nil {
			// Only log keyring errors if it's not a "not found" error
			// The "name is not activatable" error means keyring service isn't available
			if err != keyring.ErrNotFound && !isKeyringServiceError(err) {
				errors = append(errors, fmt.Sprintf("keyring: %v", err))
			}
		} else {
			removedAny = true
		}
	}

	// Remove encrypted file
	encryptedFile := filepath.Join(s.configDir, ".api_key.enc")
	if err := os.Remove(encryptedFile); err != nil {
		if !os.IsNotExist(err) {
			errors = append(errors, fmt.Sprintf("encrypted file: %v", err))
		}
	} else {
		removedAny = true
	}

	// Remove from plain text config
	s.removeAPIKeyFromConfig()

	// Check if we actually had an API key stored anywhere
	configFile := filepath.Join(s.configDir, "config.yaml")
	if _, err := os.Stat(configFile); err == nil {
		removedAny = true
	}

	// Only return error if we had real failures and didn't remove anything
	if len(errors) > 0 && !removedAny {
		return fmt.Errorf("failed to remove API key: %s", errors[0])
	}

	return nil
}

// isKeyringServiceError checks if the error is due to keyring service not being available
func isKeyringServiceError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return errStr == "The name is not activatable" ||
		errStr == "Cannot autolaunch D-Bus without X11 $DISPLAY" ||
		errStr == "The name org.freedesktop.secrets was not provided by any .service files"
}

// saveEncryptedAPIKey saves the API key in an encrypted file
func (s *SecureStorage) saveEncryptedAPIKey(apiKey string) error {
	// Ensure config directory exists
	if err := os.MkdirAll(s.configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate encryption key from machine ID
	key := s.getEncryptionKey()

	// Encrypt the API key
	encrypted, err := encrypt([]byte(apiKey), key)
	if err != nil {
		return fmt.Errorf("failed to encrypt API key: %w", err)
	}

	// Save to file with restricted permissions
	encryptedFile := filepath.Join(s.configDir, ".api_key.enc")
	if err := os.WriteFile(encryptedFile, []byte(encrypted), 0600); err != nil {
		return fmt.Errorf("failed to save encrypted API key: %w", err)
	}

	// Remove from plain text config if exists
	s.removeAPIKeyFromConfig()

	return nil
}

// getEncryptedAPIKey retrieves the API key from encrypted file
func (s *SecureStorage) getEncryptedAPIKey() (string, error) {
	encryptedFile := filepath.Join(s.configDir, ".api_key.enc")

	data, err := os.ReadFile(encryptedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read encrypted API key: %w", err)
	}

	key := s.getEncryptionKey()

	decrypted, err := decrypt(string(data), key)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt API key: %w", err)
	}

	return string(decrypted), nil
}

// getEncryptionKey generates a machine-specific encryption key
func (s *SecureStorage) getEncryptionKey() []byte {
	// Use multiple machine-specific identifiers for better entropy
	var parts []string

	// Hostname
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		parts = append(parts, hostname)
	}

	// Username
	if username := os.Getenv("USER"); username != "" {
		parts = append(parts, username)
	} else if username := os.Getenv("USERNAME"); username != "" {
		parts = append(parts, username)
	}

	// Home directory path (unique per user)
	if home, err := os.UserHomeDir(); err == nil {
		parts = append(parts, home)
	}

	// Machine ID on Linux (if available)
	if runtime.GOOS == "linux" {
		if machineID, err := os.ReadFile("/etc/machine-id"); err == nil {
			parts = append(parts, string(machineID))
		} else if machineID, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			parts = append(parts, string(machineID))
		}
	}

	// Add application-specific salt
	parts = append(parts, "RepoBird-CLI-2024-Secure")

	// Combine all parts
	combined := fmt.Sprintf("%s", parts)

	// Use SHA-256 to generate a consistent 32-byte key
	hash := sha256.Sum256([]byte(combined))
	return hash[:]
}

// getPlainTextAPIKey reads API key from plain text config (backward compatibility)
func (s *SecureStorage) getPlainTextAPIKey() string {
	configFile := filepath.Join(s.configDir, "config.yaml")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return ""
	}

	// Simple extraction (avoiding full YAML parsing for security)
	lines := string(data)
	for _, line := range splitLines(lines) {
		if len(line) > 9 && line[:9] == "api_key: " {
			return line[9:]
		}
	}
	return ""
}

// removeAPIKeyFromConfig removes API key from plain text config
func (s *SecureStorage) removeAPIKeyFromConfig() {
	configFile := filepath.Join(s.configDir, "config.yaml")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return
	}

	var newLines []string
	for _, line := range splitLines(string(data)) {
		if len(line) < 9 || line[:9] != "api_key: " {
			newLines = append(newLines, line)
		}
	}

	// Rewrite config without API key
	if err := os.WriteFile(configFile, []byte(joinLines(newLines)), 0644); err != nil {
		// Log error but don't fail - API key might already be removed from memory
		fmt.Fprintf(os.Stderr, "Warning: failed to update config file %s: %v\n", configFile, err)
	}
}

// isKeyringAvailable checks if system keyring is available
func isKeyringAvailable() bool {
	// Only use keyring on systems where it's reliably available:
	// - macOS (always has Keychain)
	// - Windows (always has Credential Manager)
	// - Linux only if explicitly in a desktop environment

	switch runtime.GOOS {
	case "darwin", "windows":
		return true
	case "linux":
		// Only use keyring if we're definitely in a desktop environment
		// This avoids issues on headless servers, containers, WSL, etc.

		// Check if running in container or SSH session (likely headless)
		if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("CONTAINER") != "" {
			return false
		}

		// Check for Docker/Kubernetes
		if _, err := os.Stat("/.dockerenv"); err == nil {
			return false
		}
		if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
			return false
		}

		// Only enable if we have a desktop session
		hasDesktop := os.Getenv("DESKTOP_SESSION") != "" ||
			os.Getenv("GNOME_DESKTOP_SESSION_ID") != "" ||
			os.Getenv("KDE_FULL_SESSION") != "" ||
			os.Getenv("XDG_CURRENT_DESKTOP") != ""

		// Also need a display
		hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""

		return hasDesktop && hasDisplay
	default:
		return false
	}
}

// encrypt encrypts data using AES-GCM
func encrypt(data []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts data using AES-GCM
func decrypt(encrypted string, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// Helper functions
func splitLines(s string) []string {
	var lines []string
	line := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, line)
			line = ""
		} else {
			line += string(r)
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		if i > 0 {
			result += "\n"
		}
		result += line
	}
	return result
}

// SecureConfig wraps Config with secure API key handling
type SecureConfig struct {
	*Config
	storage *SecureStorage
}

// LoadSecureConfig loads config with secure API key handling
func LoadSecureConfig() (*SecureConfig, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	storage := NewSecureStorage()

	// Override API key with secure storage
	if secureKey, err := storage.GetAPIKey(); err == nil && secureKey != "" {
		config.APIKey = secureKey
	}

	return &SecureConfig{
		Config:  config,
		storage: storage,
	}, nil
}

// SaveAPIKey saves the API key securely
func (sc *SecureConfig) SaveAPIKey(apiKey string) error {
	return sc.storage.SaveAPIKey(apiKey)
}

// GetStorageInfo returns information about where the API key is stored
func (sc *SecureConfig) GetStorageInfo() map[string]interface{} {
	info := make(map[string]interface{})

	// Check environment variable
	if os.Getenv(EnvAPIKey) != "" {
		info["source"] = "environment"
		info["secure"] = true
		return info
	}

	// Check keyring
	if sc.storage.useKeyring {
		if _, err := keyring.Get(keyringService, keyringAccount); err == nil {
			info["source"] = "system_keyring"
			info["secure"] = true
			info["keyring_type"] = getKeyringType()
			return info
		}
	}

	// Check encrypted file
	encryptedFile := filepath.Join(sc.storage.configDir, ".api_key.enc")
	if _, err := os.Stat(encryptedFile); err == nil {
		info["source"] = "encrypted_file"
		info["secure"] = true
		info["location"] = encryptedFile
		return info
	}

	// Check plain text
	if sc.storage.getPlainTextAPIKey() != "" {
		configFile := filepath.Join(sc.storage.configDir, "config.yaml")
		info["source"] = "plain_text_config"
		info["secure"] = false
		info["location"] = configFile
		info["warning"] = "API key stored in plain text. Run 'rb config set api-key' to migrate to secure storage."
		return info
	}

	info["source"] = "not_found"
	info["secure"] = false
	return info
}

func getKeyringType() string {
	switch runtime.GOOS {
	case "darwin":
		return "macOS Keychain"
	case "windows":
		return "Windows Credential Manager"
	case "linux":
		if os.Getenv("GNOME_DESKTOP_SESSION_ID") != "" {
			return "GNOME Keyring"
		}
		if os.Getenv("KDE_FULL_SESSION") != "" {
			return "KWallet"
		}
		return "Linux Secret Service"
	default:
		return "Unknown"
	}
}
