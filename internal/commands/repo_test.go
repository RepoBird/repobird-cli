// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bytes"
	"testing"

	"github.com/repobird/repobird-cli/internal/models"
)

func TestPrintRepositoryDetailsShowsBranchDefaults(t *testing.T) {
	base := "develop"
	prTarget := "release"
	repo := &models.APIRepository{
		ID:                    42,
		RepoOwner:             "acme",
		RepoName:              "webapp",
		DefaultBranch:         "main",
		DefaultBaseBranch:     &base,
		DefaultPRTargetBranch: &prTarget,
		DefaultOutputBranch:   nil,
	}

	var out bytes.Buffer
	printRepositoryDetails(&out, repo)

	got := out.String()
	for _, want := range []string{
		"Repository: acme/webapp",
		"Default Branch: main",
		"Default Base Branch: develop",
		"Default PR Target Branch: release",
		"Default Output Branch: (generated)",
	} {
		if !bytes.Contains([]byte(got), []byte(want)) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, got)
		}
	}
}

func TestBuildRepositoryDefaultsUpdateTreatsBlankOutputAsClear(t *testing.T) {
	base := "develop"
	blank := ""

	update := buildRepositoryDefaultsUpdate(repositoryDefaultsOptions{
		baseBranch:   &base,
		outputBranch: &blank,
	})
	payload := update.Payload()

	if got := payload["defaultBaseBranch"]; got != "develop" {
		t.Fatalf("expected defaultBaseBranch develop, got %#v", got)
	}
	if got, ok := payload["defaultOutputBranch"]; !ok || got != nil {
		t.Fatalf("expected blank output branch to clear defaultOutputBranch, got %#v (present=%v)", got, ok)
	}
}
