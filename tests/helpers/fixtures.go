package helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// LoadFixture loads test data from the fixtures directory
func LoadFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to load fixture: %s", name)

	return data
}

// LoadJSONFixture loads and unmarshals JSON test data
func LoadJSONFixture(t *testing.T, name string, target interface{}) {
	t.Helper()

	data := LoadFixture(t, name)
	err := json.Unmarshal(data, target)
	require.NoError(t, err, "failed to unmarshal JSON fixture: %s", name)
}

// WriteFixture creates a test fixture file
func WriteFixture(t *testing.T, name string, data []byte) {
	t.Helper()

	dir := filepath.Join("testdata")
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err)

	path := filepath.Join(dir, name)
	err = os.WriteFile(path, data, 0644)
	require.NoError(t, err, "failed to write fixture: %s", name)
}

// CreateTempFile creates a temporary file with given content
func CreateTempFile(t *testing.T, pattern, content string) string {
	t.Helper()

	tmpfile, err := os.CreateTemp("", pattern)
	require.NoError(t, err)

	_, err = tmpfile.WriteString(content)
	require.NoError(t, err)

	err = tmpfile.Close()
	require.NoError(t, err)

	// Register cleanup
	t.Cleanup(func() {
		os.Remove(tmpfile.Name())
	})

	return tmpfile.Name()
}
