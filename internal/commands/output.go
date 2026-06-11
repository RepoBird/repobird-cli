// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/repobird/repobird-cli/internal/api/dto"
	"github.com/repobird/repobird-cli/internal/bulk"
	configpkg "github.com/repobird/repobird-cli/internal/config"
	"github.com/repobird/repobird-cli/internal/domain"
	"github.com/repobird/repobird-cli/internal/output"
	"github.com/repobird/repobird-cli/internal/utils"
)

func styleFor(out io.Writer) output.Styler {
	mode := output.ColorAuto
	if cfg != nil && cfg.Config != nil {
		mode = cfg.Color
	} else if loaded, err := configpkg.LoadSecureConfig(); err == nil && loaded.Config != nil {
		mode = loaded.Color
	}
	return output.NewStyler(out, output.ModeFromEnv(mode))
}

func stdoutStyle() output.Styler {
	return styleFor(os.Stdout)
}

func stderrStyle() output.Styler {
	return styleFor(os.Stderr)
}

type runCreateJSONOutput struct {
	Schema    string      `json:"schema"`
	Operation string      `json:"operation"`
	Success   bool        `json:"success"`
	Run       runJSON     `json:"run"`
	URL       string      `json:"url,omitempty"`
	Request   *runRequest `json:"request,omitempty"`
}

type runDryRunJSONOutput struct {
	Schema    string     `json:"schema"`
	Operation string     `json:"operation"`
	Valid     bool       `json:"valid"`
	Request   runRequest `json:"request"`
}

type runWaitJSONOutput struct {
	Schema    string      `json:"schema"`
	Operation string      `json:"operation"`
	Success   bool        `json:"success"`
	Run       runJSON     `json:"run,omitempty"`
	URL       string      `json:"url,omitempty"`
	Request   *runRequest `json:"request,omitempty"`
	ExitCode  int         `json:"exitCode"`
	Status    string      `json:"status,omitempty"`
	TimedOut  bool        `json:"timedOut"`
	Error     string      `json:"error,omitempty"`
}

type runJSON struct {
	ID                 string  `json:"id"`
	PublicID           string  `json:"publicId,omitempty"`
	Status             string  `json:"status,omitempty"`
	StatusMessage      string  `json:"statusMessage,omitempty"`
	RepositoryName     string  `json:"repositoryName,omitempty"`
	SourceBranch       string  `json:"sourceBranch,omitempty"`
	TargetBranch       string  `json:"targetBranch,omitempty"`
	BaseBranch         string  `json:"baseBranch,omitempty"`
	OutputMode         string  `json:"outputMode,omitempty"`
	OutputBranch       string  `json:"outputBranch,omitempty"`
	PRTargetBranch     string  `json:"prTargetBranch,omitempty"`
	OutputBranchPolicy string  `json:"outputBranchPolicy,omitempty"`
	PullRequestURL     string  `json:"prUrl,omitempty"`
	RunType            string  `json:"runType,omitempty"`
	Title              string  `json:"title,omitempty"`
	Error              string  `json:"error,omitempty"`
	Cost               float64 `json:"cost,omitempty"`
	InputTokens        int     `json:"inputTokens,omitempty"`
	OutputTokens       int     `json:"outputTokens,omitempty"`
	FileCount          int     `json:"fileCount,omitempty"`
}

type runRequest struct {
	Prompt                string   `json:"prompt"`
	RepositoryName        string   `json:"repositoryName"`
	SourceBranch          string   `json:"sourceBranch,omitempty"`
	TargetBranch          string   `json:"targetBranch,omitempty"`
	BaseBranch            string   `json:"baseBranch,omitempty"`
	OutputMode            string   `json:"outputMode,omitempty"`
	OutputBranch          string   `json:"outputBranch,omitempty"`
	PRTargetBranch        string   `json:"prTargetBranch,omitempty"`
	OutputBranchPolicy    string   `json:"outputBranchPolicy,omitempty"`
	RunType               string   `json:"runType"`
	Agent                 string   `json:"agent,omitempty"`
	OpenCodeModel         string   `json:"opencodeModel,omitempty"`
	OpenCodeProvider      string   `json:"opencodeProvider,omitempty"`
	Title                 string   `json:"title,omitempty"`
	Context               string   `json:"context,omitempty"`
	Files                 []string `json:"files,omitempty"`
	BranchOnly            bool     `json:"branchOnly,omitempty"`
	AcknowledgePromptRisk bool     `json:"acknowledgePromptRisk,omitempty"`
	IdempotencyKey        string   `json:"idempotencyKey,omitempty"`
}

type bulkDryRunJSONOutput struct {
	Schema         string             `json:"schema"`
	Operation      string             `json:"operation"`
	Valid          bool               `json:"valid"`
	RepositoryName string             `json:"repositoryName,omitempty"`
	RepoID         int                `json:"repoId,omitempty"`
	SourceBranch   string             `json:"sourceBranch,omitempty"`
	RunType        string             `json:"runType,omitempty"`
	BatchTitle     string             `json:"batchTitle,omitempty"`
	TotalRuns      int                `json:"totalRuns"`
	Runs           []bulkRunConfigOut `json:"runs"`
}

type bulkRunConfigOut struct {
	Index        int    `json:"index"`
	Title        string `json:"title"`
	Prompt       string `json:"prompt,omitempty"`
	TargetBranch string `json:"targetBranch,omitempty"`
	Context      string `json:"context,omitempty"`
}

type bulkCreateJSONOutput struct {
	Schema          string            `json:"schema"`
	Operation       string            `json:"operation"`
	Success         bool              `json:"success"`
	BatchID         string            `json:"batchId"`
	BatchTitle      string            `json:"batchTitle,omitempty"`
	Runs            []bulkRunJSON     `json:"runs"`
	Failed          []bulkFailureJSON `json:"failed,omitempty"`
	TotalRequested  int               `json:"totalRequested"`
	TotalSuccessful int               `json:"totalSuccessful"`
	TotalFailed     int               `json:"totalFailed"`
}

type bulkRunJSON struct {
	ID             string `json:"id"`
	Status         string `json:"status,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
	Title          string `json:"title,omitempty"`
	Index          int    `json:"index"`
}

type bulkFailureJSON struct {
	Index         int    `json:"index"`
	Prompt        string `json:"prompt,omitempty"`
	Error         string `json:"error,omitempty"`
	Message       string `json:"message,omitempty"`
	ExistingRunID int    `json:"existingRunId,omitempty"`
}

func printRunDryRunJSON(out io.Writer, req domain.CreateRunRequest) error {
	return printJSON(out, runDryRunJSONOutput{
		Schema:    "repobird.run.dry_run.v1",
		Operation: "run.dry_run",
		Valid:     true,
		Request:   makeRunRequestJSON(req),
	})
}

func printRunCreateJSON(out io.Writer, run *domain.Run, req domain.CreateRunRequest) error {
	urlID := createdRunURLID(run)
	output := runCreateJSONOutput{
		Schema:    "repobird.run.create.v1",
		Operation: "run.create",
		Success:   true,
		Run:       makeRunJSON(run),
	}
	if urlID != "" {
		output.URL = utils.GenerateRepoBirdURL(urlID)
	}
	request := makeRunRequestJSON(req)
	output.Request = &request
	return printJSON(out, output)
}

func printRunWaitJSON(out io.Writer, run *domain.Run, req domain.CreateRunRequest, exitCode int, timedOut bool, message string) error {
	urlID := createdRunURLID(run)
	output := runWaitJSONOutput{
		Schema:    "repobird.run.wait.v1",
		Operation: "run.wait",
		Success:   exitCode == ExitCodeSuccess,
		Run:       makeRunJSON(run),
		ExitCode:  exitCode,
		TimedOut:  timedOut,
		Error:     message,
	}
	if run != nil {
		output.Status = run.Status
	}
	if urlID != "" {
		output.URL = utils.GenerateRepoBirdURL(urlID)
	}
	request := makeRunRequestJSON(req)
	output.Request = &request
	return printJSON(out, output)
}

func makeRunJSON(run *domain.Run) runJSON {
	if run == nil {
		return runJSON{}
	}
	return runJSON{
		ID:                 run.ID,
		PublicID:           run.PublicID,
		Status:             run.Status,
		StatusMessage:      run.StatusMessage,
		RepositoryName:     run.RepositoryName,
		SourceBranch:       run.SourceBranch,
		TargetBranch:       run.TargetBranch,
		BaseBranch:         run.BaseBranch,
		OutputMode:         run.OutputMode,
		OutputBranch:       run.OutputBranch,
		PRTargetBranch:     run.PRTargetBranch,
		OutputBranchPolicy: run.OutputBranchPolicy,
		PullRequestURL:     run.PullRequestURL,
		RunType:            run.RunType,
		Title:              run.Title,
		Error:              run.Error,
		Cost:               run.Cost,
		InputTokens:        run.InputTokens,
		OutputTokens:       run.OutputTokens,
		FileCount:          run.FileCount,
	}
}

func makeRunRequestJSON(req domain.CreateRunRequest) runRequest {
	return runRequest{
		Prompt:                req.Prompt,
		RepositoryName:        req.RepositoryName,
		SourceBranch:          req.SourceBranch,
		TargetBranch:          req.TargetBranch,
		BaseBranch:            req.BaseBranch,
		OutputMode:            req.OutputMode,
		OutputBranch:          req.OutputBranch,
		PRTargetBranch:        req.PRTargetBranch,
		OutputBranchPolicy:    req.OutputBranchPolicy,
		RunType:               req.RunType,
		Agent:                 req.Agent,
		OpenCodeModel:         req.OpenCodeModel,
		OpenCodeProvider:      req.OpenCodeProvider,
		Title:                 req.Title,
		Context:               req.Context,
		Files:                 req.Files,
		BranchOnly:            req.BranchOnly,
		AcknowledgePromptRisk: req.AcknowledgePromptRisk,
		IdempotencyKey:        req.IdempotencyKey,
	}
}

func printBulkDryRunJSON(out io.Writer, bulkConfig *bulk.BulkConfig) error {
	runs := make([]bulkRunConfigOut, 0, len(bulkConfig.Runs))
	for i, run := range bulkConfig.Runs {
		title := run.Title
		if title == "" {
			title = fallbackRunTitle(i)
		}
		runs = append(runs, bulkRunConfigOut{
			Index:        i + 1,
			Title:        title,
			Prompt:       run.Prompt,
			TargetBranch: run.Target,
			Context:      run.Context,
		})
	}
	return printJSON(out, bulkDryRunJSONOutput{
		Schema:         "repobird.bulk.dry_run.v1",
		Operation:      "bulk.dry_run",
		Valid:          true,
		RepositoryName: bulkConfig.Repository,
		RepoID:         bulkConfig.RepoID,
		SourceBranch:   bulkConfig.Source,
		RunType:        bulkConfig.RunType,
		BatchTitle:     bulkConfig.BatchTitle,
		TotalRuns:      len(bulkConfig.Runs),
		Runs:           runs,
	})
}

func printBulkCreateJSON(out io.Writer, bulkResp *dto.BulkRunResponse) error {
	runs := make([]bulkRunJSON, 0, len(bulkResp.Data.Successful))
	for _, run := range bulkResp.Data.Successful {
		runs = append(runs, bulkRunJSON{
			ID:             intIDString(run.ID),
			Status:         run.Status,
			RepositoryName: run.RepositoryName,
			Title:          run.Title,
			Index:          run.RequestIndex + 1,
		})
	}
	failed := make([]bulkFailureJSON, 0, len(bulkResp.Data.Failed))
	for _, failure := range bulkResp.Data.Failed {
		failed = append(failed, bulkFailureJSON{
			Index:         failure.RequestIndex + 1,
			Prompt:        failure.Prompt,
			Error:         failure.Error,
			Message:       failure.Message,
			ExistingRunID: failure.ExistingRunId,
		})
	}
	return printJSON(out, bulkCreateJSONOutput{
		Schema:          "repobird.bulk.create.v1",
		Operation:       "bulk.create",
		Success:         len(bulkResp.Data.Failed) == 0,
		BatchID:         bulkResp.Data.BatchID,
		BatchTitle:      bulkResp.Data.BatchTitle,
		Runs:            runs,
		Failed:          failed,
		TotalRequested:  bulkResp.Data.Metadata.TotalRequested,
		TotalSuccessful: bulkResp.Data.Metadata.TotalSuccessful,
		TotalFailed:     bulkResp.Data.Metadata.TotalFailed,
	})
}

func fallbackRunTitle(index int) string {
	return "Run " + intIDString(index+1)
}

func intIDString(id int) string {
	return fmt.Sprintf("%d", id)
}
