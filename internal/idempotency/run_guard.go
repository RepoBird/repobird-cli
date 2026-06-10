package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrDuplicate = errors.New("duplicate run submission")

type RunIdentity struct {
	Repository string
	Prompt     string
	RunType    string
}

type RunGuard struct {
	cacheFile string
	window    time.Duration
	now       func() time.Time
}

type runGuardData struct {
	Submissions map[string]time.Time `json:"submissions"`
}

func BuildRunKey(identity RunIdentity) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{
		strings.ToLower(strings.TrimSpace(identity.Repository)),
		strings.TrimSpace(identity.Prompt),
		strings.TrimSpace(identity.RunType),
	}, "\x00")))

	return hex.EncodeToString(hash[:])
}

func NewRunGuard(cacheDir string, window time.Duration, now func() time.Time) *RunGuard {
	if now == nil {
		now = time.Now
	}
	return &RunGuard{
		cacheFile: filepath.Join(cacheDir, "run_submissions.json"),
		window:    window,
		now:       now,
	}
}

func DefaultCacheDir() string {
	baseDir, err := os.UserCacheDir()
	if err != nil {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return filepath.Join(os.TempDir(), "repobird", "run-submissions")
		}
		baseDir = filepath.Join(homeDir, ".cache")
	}
	return filepath.Join(baseDir, "repobird", "run-submissions")
}

func (g *RunGuard) Reserve(key string, force bool) error {
	if key == "" {
		return nil
	}

	data := g.load()
	now := g.now().UTC()
	if submittedAt, ok := data.Submissions[key]; ok && now.Sub(submittedAt) < g.window && !force {
		return fmt.Errorf("%w: identical run submitted %s ago; pass --force to submit again",
			ErrDuplicate,
			now.Sub(submittedAt).Round(time.Second),
		)
	}

	data.Submissions[key] = now
	g.prune(data, now)
	return g.save(data)
}

func (g *RunGuard) load() runGuardData {
	data := runGuardData{Submissions: make(map[string]time.Time)}

	body, err := os.ReadFile(g.cacheFile)
	if err != nil {
		return data
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return runGuardData{Submissions: make(map[string]time.Time)}
	}
	if data.Submissions == nil {
		data.Submissions = make(map[string]time.Time)
	}
	return data
}

func (g *RunGuard) save(data runGuardData) error {
	if err := os.MkdirAll(filepath.Dir(g.cacheFile), 0o755); err != nil {
		return fmt.Errorf("failed to create idempotency cache directory: %w", err)
	}

	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode idempotency cache: %w", err)
	}
	if err := os.WriteFile(g.cacheFile, body, 0o600); err != nil {
		return fmt.Errorf("failed to write idempotency cache: %w", err)
	}
	return nil
}

func (g *RunGuard) prune(data runGuardData, now time.Time) {
	for key, submittedAt := range data.Submissions {
		if now.Sub(submittedAt) > g.window {
			delete(data.Submissions, key)
		}
	}
}
