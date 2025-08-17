// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package version

import (
	"strings"
	"testing"
)

const testVersion = "1.0.0"

func TestGetVersion(t *testing.T) {
	Version = testVersion
	if v := GetVersion(); v != testVersion {
		t.Errorf("GetVersion() = %s, want %s", v, testVersion)
	}
}

func TestGetBuildInfo(t *testing.T) {
	Version = testVersion
	GitCommit = "abc123"
	BuildDate = "2024-01-01"

	info := GetBuildInfo()

	if !strings.Contains(info, "Version: "+testVersion) {
		t.Error("Build info should contain version")
	}

	if !strings.Contains(info, "Git Commit: abc123") {
		t.Error("Build info should contain git commit")
	}

	if !strings.Contains(info, "Build Date: 2024-01-01") {
		t.Error("Build info should contain build date")
	}

	if !strings.Contains(info, "Go Version:") {
		t.Error("Build info should contain Go version")
	}

	if !strings.Contains(info, "OS/Arch:") {
		t.Error("Build info should contain OS/Arch")
	}
}
