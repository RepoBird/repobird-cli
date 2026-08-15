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

func TestRunResponsePolicyFieldsAreIndependentOpaqueValues(t *testing.T) {
	tests := []struct {
		name          string
		run           Run
		wantModel     string
		wantReasoning string
	}{
		{
			name:          "complete unfamiliar policy",
			run:           Run{Model: "future-provider/model-v9", ReasoningEffort: "adaptive-ultra"},
			wantModel:     "future-provider/model-v9",
			wantReasoning: "adaptive-ultra",
		},
		{
			name:      "model only",
			run:       Run{Model: "model-only"},
			wantModel: "model-only",
		},
		{
			name:          "reasoning effort only",
			run:           Run{ReasoningEffort: "reasoning-only"},
			wantReasoning: "reasoning-only",
		},
		{name: "absent legacy policy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.run.Model != tt.wantModel {
				t.Fatalf("Model = %q, want %q", tt.run.Model, tt.wantModel)
			}
			if tt.run.ReasoningEffort != tt.wantReasoning {
				t.Fatalf("ReasoningEffort = %q, want %q", tt.run.ReasoningEffort, tt.wantReasoning)
			}
		})
	}
}
