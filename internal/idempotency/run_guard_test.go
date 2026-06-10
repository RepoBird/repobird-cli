package idempotency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildRunKeyIsStableForEquivalentRequests(t *testing.T) {
	first := BuildRunKey(RunIdentity{
		Repository: "Owner/Repo",
		Prompt:     "  Fix auth bug\n",
		RunType:    "run",
	})
	second := BuildRunKey(RunIdentity{
		Repository: "owner/repo",
		Prompt:     "Fix auth bug",
		RunType:    "run",
	})

	require.Equal(t, first, second)
	require.NotEmpty(t, first)
}

func TestBuildRunKeyChangesForDifferentPrompt(t *testing.T) {
	first := BuildRunKey(RunIdentity{
		Repository: "owner/repo",
		Prompt:     "Fix auth bug",
		RunType:    "run",
	})
	second := BuildRunKey(RunIdentity{
		Repository: "owner/repo",
		Prompt:     "Add auth tests",
		RunType:    "run",
	})

	require.NotEqual(t, first, second)
}

func TestGuardBlocksDuplicateWithinWindow(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	guard := NewRunGuard(t.TempDir(), 30*time.Second, func() time.Time { return now })

	require.NoError(t, guard.Reserve("key-1", false))
	err := guard.Reserve("key-1", false)

	require.Error(t, err)
	require.ErrorIs(t, err, ErrDuplicate)
	require.Contains(t, err.Error(), "identical run submitted")
	require.Contains(t, err.Error(), "--force")
}

func TestGuardAllowsDuplicateWithForce(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	guard := NewRunGuard(t.TempDir(), 30*time.Second, func() time.Time { return now })

	require.NoError(t, guard.Reserve("key-1", false))
	require.NoError(t, guard.Reserve("key-1", true))
}

func TestGuardAllowsExpiredDuplicate(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	guard := NewRunGuard(t.TempDir(), 30*time.Second, func() time.Time { return now })

	require.NoError(t, guard.Reserve("key-1", false))
	now = now.Add(31 * time.Second)

	require.NoError(t, guard.Reserve("key-1", false))
}
