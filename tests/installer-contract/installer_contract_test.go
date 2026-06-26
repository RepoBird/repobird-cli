package installercontract

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallersUseCanonicalGitHubReleaseAssets(t *testing.T) {
	root := repoRoot(t)

	goreleaser := readFile(t, root, ".goreleaser.yml")
	assertContains(t, goreleaser, `name_template: "{{ .ProjectName }}_{{ .Os }}_{{ .Arch }}"`)
	assertContains(t, goreleaser, `name_template: "checksums.txt"`)

	shell := readFile(t, root, "scripts/install.sh")
	assertContains(t, shell, `GITHUB_REPO="RepoBird/repobird-cli"`)
	assertContains(t, shell, `PROJECT_NAME="repobird-cli"`)
	assertContains(t, shell, `https://github.com/${GITHUB_REPO}/releases/latest/download/${filename}`)
	assertContains(t, shell, `"${PROJECT_NAME}_${platform}.tar.gz"`)
	assertContains(t, shell, `"${PROJECT_NAME}_${platform}.zip"`)
	assertContains(t, shell, `checksums.txt`)

	powershell := readFile(t, root, "scripts/install.ps1")
	assertContains(t, powershell, `$GITHUB_REPO = "RepoBird/repobird-cli"`)
	assertContains(t, powershell, `$PROJECT_NAME = "repobird-cli"`)
	assertContains(t, powershell, `https://github.com/$GITHUB_REPO/releases/latest/download/$filename`)
	assertContains(t, powershell, `"${PROJECT_NAME}_${platform}.zip"`)
	assertContains(t, powershell, `checksums.txt`)
}

func TestInstallationDocsAdvertiseFirstPartyEntrypointsAndFollowUps(t *testing.T) {
	root := repoRoot(t)

	docs := readFile(t, root, "docs/INSTALLATION.md")
	assertContains(t, docs, `https://repobird.ai/install`)
	assertContains(t, docs, `https://repobird.ai/install.sh`)
	assertContains(t, docs, `https://repobird.ai/install.ps1`)
	assertContains(t, docs, `repobird-cli_linux_amd64.tar.gz`)
	assertContains(t, docs, `repobird-cli_windows_amd64.zip`)
	assertContains(t, docs, `Package manager follow-ups`)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

func readFile(t *testing.T, root, path string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}

func assertContains(t *testing.T, content, want string) {
	t.Helper()

	if !strings.Contains(content, want) {
		t.Fatalf("expected content to contain %q", want)
	}
}
