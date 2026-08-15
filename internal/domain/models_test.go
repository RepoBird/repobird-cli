// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package domain

import "testing"

func TestCreateRunRequestReasoningEffortContract(t *testing.T) {
	t.Run("preserves opaque reasoning effort without deriving policy", func(t *testing.T) {
		req := CreateRunRequest{
			Prompt:          "Fix auth",
			RepositoryName:  "acme/webapp",
			RunType:         RunTypeBasic,
			ReasoningEffort: "future-adaptive-effort",
		}

		if req.ReasoningEffort != "future-adaptive-effort" {
			t.Fatalf("ReasoningEffort = %q, want exact opaque value", req.ReasoningEffort)
		}
		if req.OpenCodeModel != "" {
			t.Fatalf("OpenCodeModel = %q, want zero value without derived policy", req.OpenCodeModel)
		}
		if req.OpenCodeProvider != "" {
			t.Fatalf("OpenCodeProvider = %q, want zero value without derived policy", req.OpenCodeProvider)
		}
		if req.RunType != RunTypeBasic {
			t.Fatalf("RunType = %q, want %q", req.RunType, RunTypeBasic)
		}
	})

	t.Run("keeps optional reasoning effort at its zero value when omitted", func(t *testing.T) {
		req := CreateRunRequest{}

		if req.ReasoningEffort != "" {
			t.Fatalf("ReasoningEffort = %q, want empty zero value", req.ReasoningEffort)
		}
	})
}
