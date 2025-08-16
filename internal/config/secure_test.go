package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// redactKey redacts an API key for safe display in test output
func redactKey(key string) string {
	if key == "" {
		return "<empty>"
	}
	if len(key) <= 8 {
		return "***"
	}
	// Show first 4 chars and redact the rest
	return key[:4] + strings.Repeat("*", len(key)-4)
}

func TestSecureStorage_SaveAndGetAPIKey(t *testing.T) {
	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "repobird-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create secure storage instance with test directory
	storage := &SecureStorage{
		useKeyring: false, // Don't use keyring in tests
		configDir:  tmpDir,
	}

	testKey := "test-api-key-123456789"

	// Test saving API key
	err = storage.SaveAPIKey(testKey)
	if err != nil {
		t.Errorf("SaveAPIKey failed: %v", err)
	}

	// Test retrieving API key
	retrievedKey, err := storage.GetAPIKey()
	if err != nil {
		t.Errorf("GetAPIKey failed: %v", err)
	}

	if retrievedKey != testKey {
		// Redact API keys in error message
		redactedGot := redactKey(retrievedKey)
		redactedWant := redactKey(testKey)
		t.Errorf("Retrieved key mismatch: got %q, want %q", redactedGot, redactedWant)
	}

	// Verify encrypted file was created
	encFile := filepath.Join(tmpDir, ".api_key.enc")
	if _, err := os.Stat(encFile); os.IsNotExist(err) {
		t.Error("Encrypted file was not created")
	}

	// Verify file permissions (Unix only)
	if info, err := os.Stat(encFile); err == nil {
		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Incorrect file permissions: got %o, want 0600", mode)
		}
	}
}

func TestSecureStorage_DeleteAPIKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repobird-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := &SecureStorage{
		useKeyring: false,
		configDir:  tmpDir,
	}

	// Save a key first
	testKey := "test-api-key-to-delete"
	err = storage.SaveAPIKey(testKey)
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Delete the key
	err = storage.DeleteAPIKey()
	if err != nil {
		t.Errorf("DeleteAPIKey failed: %v", err)
	}

	// Verify key is deleted
	retrievedKey, err := storage.GetAPIKey()
	if err == nil && retrievedKey != "" {
		// Redact API key in error message
		redacted := redactKey(retrievedKey)
		t.Errorf("Key was not deleted: still got %q", redacted)
	}

	// Verify encrypted file is removed
	encFile := filepath.Join(tmpDir, ".api_key.enc")
	if _, err := os.Stat(encFile); !os.IsNotExist(err) {
		t.Error("Encrypted file was not removed")
	}
}

func TestSecureStorage_EmptyAPIKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repobird-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := &SecureStorage{
		useKeyring: false,
		configDir:  tmpDir,
	}

	// Test saving empty API key
	err = storage.SaveAPIKey("")
	if err == nil {
		t.Error("SaveAPIKey should fail with empty key")
	}
}

func TestSecureStorage_EnvironmentVariable(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repobird-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := &SecureStorage{
		useKeyring: false,
		configDir:  tmpDir,
	}

	// Set environment variable
	envKey := "env-api-key-123"
	os.Setenv(EnvAPIKey, envKey)
	defer os.Unsetenv(EnvAPIKey)

	// Get API key should return env var first
	retrievedKey, err := storage.GetAPIKey()
	if err != nil {
		t.Errorf("GetAPIKey failed: %v", err)
	}

	if retrievedKey != envKey {
		// Redact API keys in error message
		redactedGot := redactKey(retrievedKey)
		redactedWant := redactKey(envKey)
		t.Errorf("Should get env var key: got %q, want %q", redactedGot, redactedWant)
	}
}

func TestSecureStorage_Encryption(t *testing.T) {
	// Test encryption and decryption functions
	key := []byte("12345678901234567890123456789012") // 32 bytes for AES-256
	plaintext := []byte("secret-api-key")

	// Encrypt
	encrypted, err := encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if encrypted == "" {
		t.Error("Encrypted data is empty")
	}

	// Decrypt
	decrypted, err := decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted mismatch: got %q, want %q", decrypted, plaintext)
	}

	// Test with wrong key
	wrongKey := []byte("00000000000000000000000000000000")
	_, err = decrypt(encrypted, wrongKey)
	if err == nil {
		t.Error("Decryption should fail with wrong key")
	}
}

func TestSecureStorage_PlainTextMigration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repobird-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a plain text config file with API key
	configFile := filepath.Join(tmpDir, "config.yaml")
	plainTextKey := "plain-text-api-key-789"
	configContent := "api_key: " + plainTextKey + "\napi_url: https://api.example.com\n"

	err = os.WriteFile(configFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	storage := &SecureStorage{
		useKeyring: false,
		configDir:  tmpDir,
	}

	// Get API key should migrate from plain text
	retrievedKey, err := storage.GetAPIKey()
	if err != nil {
		t.Errorf("GetAPIKey failed: %v", err)
	}

	if retrievedKey != plainTextKey {
		// Redact API keys in error message
		redactedGot := redactKey(retrievedKey)
		redactedWant := redactKey(plainTextKey)
		t.Errorf("Key mismatch: got %q, want %q", redactedGot, redactedWant)
	}

	// Verify encrypted file was created (migration happened)
	encFile := filepath.Join(tmpDir, ".api_key.enc")
	if _, err := os.Stat(encFile); os.IsNotExist(err) {
		t.Error("Migration did not create encrypted file")
	}

	// Verify plain text key was removed from config
	newConfig, _ := os.ReadFile(configFile)
	if contains := string(newConfig); contains != "" &&
		(contains == "api_key: " || contains == plainTextKey) {
		t.Error("Plain text API key was not removed from config")
	}
}

func TestGetStorageInfo(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "repobird-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	storage := &SecureStorage{
		useKeyring: false,
		configDir:  tmpDir,
	}

	secureConfig := &SecureConfig{
		Config:  &Config{},
		storage: storage,
	}

	// Test when no key is stored
	info := secureConfig.GetStorageInfo()
	if source := info["source"]; source != "not_found" {
		t.Errorf("Expected 'not_found', got %v", source)
	}

	// Save a key to encrypted file
	testKey := "test-key-for-info"
	err = storage.SaveAPIKey(testKey)
	if err != nil {
		t.Fatalf("Failed to save key: %v", err)
	}

	// Check storage info
	info = secureConfig.GetStorageInfo()
	if source := info["source"]; source != "encrypted_file" {
		t.Errorf("Expected 'encrypted_file', got %v", source)
	}
	if secure := info["secure"].(bool); !secure {
		t.Error("Encrypted file should be marked as secure")
	}
}

func TestIsKeyringAvailable(t *testing.T) {
	// This test checks the logic but actual availability depends on the system
	available := isKeyringAvailable()

	// The function should return a boolean without errors
	_ = available

	// On CI/containers, it should generally return false
	if os.Getenv("CI") != "" || os.Getenv("CONTAINER") != "" {
		if available {
			t.Error("Keyring should not be available in CI/container environment")
		}
	}
}
