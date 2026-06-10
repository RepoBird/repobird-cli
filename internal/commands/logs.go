// Copyright (C) 2025 Ariel Frischer
// SPDX-License-Identifier: AGPL-3.0-or-later

package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/repobird/repobird-cli/internal/api"
	"github.com/repobird/repobird-cli/internal/errors"
	"github.com/repobird/repobird-cli/internal/models"
	"github.com/repobird/repobird-cli/internal/utils"
)

var (
	logsJSON   bool
	logsFollow bool
)

var logsCmd = &cobra.Command{
	Use:   "logs <run-id>",
	Short: "Inspect run agent logs",
	Long: `Inspect the agent conversation logs/messages for a run.

The current RepoBird API exposes logs as NDJSON through the agent-logs endpoint.
Without --follow, the CLI fetches the current snapshot once. With --follow, the
CLI polls for newer messages and writes NDJSON lines as they arrive.`,
	Args: cobra.ExactArgs(1),
	RunE: logsCommand,
}

//nolint:gochecknoinits // Required for CLI command registration
func init() {
	logsCmd.Flags().BoolVar(&logsJSON, "json", false, "output the current log snapshot as JSON")
	logsCmd.Flags().BoolVar(&logsFollow, "follow", false, "poll for new log messages and output NDJSON")
}

func logsCommand(_ *cobra.Command, args []string) error {
	if cfg.APIKey == "" {
		return errors.NoAPIKeyError()
	}

	client := api.NewClient(cfg.APIKey, utils.GetAPIURL(cfg.APIURL), cfg.Debug)
	runID := args[0]
	if logsFollow {
		return followRunLogs(context.Background(), client, runID, os.Stdout)
	}

	messages, err := client.GetRunLogs(context.Background(), runID, 0)
	if err != nil {
		return fmt.Errorf("failed to get run logs: %s", errors.FormatUserError(err))
	}
	return renderRunLogs(os.Stdout, messages, logsJSON)
}

func renderRunLogs(out io.Writer, messages []models.RunLogMessage, asJSON bool) error {
	if asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(messages)
	}

	for _, message := range messages {
		writeHumanLogMessage(out, message)
	}
	return nil
}

func writeHumanLogMessage(out io.Writer, message models.RunLogMessage) {
	label := message.Type
	if label == "tool_call" {
		label = "tool"
	}
	if label == "" {
		label = "log"
	}

	content := message.Content
	if message.ToolName != "" {
		content = message.ToolName
	}
	if content != "" {
		_, _ = fmt.Fprintf(out, "[%s] %s\n", label, content)
	}
	if message.ToolParams != "" {
		_, _ = fmt.Fprintf(out, "  params: %s\n", message.ToolParams)
	}
	if message.ToolResult != "" {
		_, _ = fmt.Fprintf(out, "  result: %s\n", message.ToolResult)
	}
}

type runLogClient interface {
	OpenRunLogs(ctx context.Context, id string, afterSeq int) (io.ReadCloser, error)
	GetRunWithRetry(ctx context.Context, id string) (*models.RunResponse, error)
}

func followRunLogs(ctx context.Context, client runLogClient, runID string, out io.Writer) error {
	ticker := time.NewTicker(utils.DefaultPollInterval)
	defer ticker.Stop()

	afterSeq := 0
	seen := make(map[string]struct{})
	for {
		nextSeq, wrote, err := fetchAndWriteFollowLogs(ctx, client, runID, afterSeq, seen, out)
		if err != nil {
			return err
		}
		afterSeq = nextSeq

		run, err := client.GetRunWithRetry(ctx, runID)
		if err == nil && utils.IsTerminalStatus(run.Status) {
			return nil
		}

		if !wrote && err != nil && ctx.Err() != nil {
			return ctx.Err()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func fetchAndWriteFollowLogs(
	ctx context.Context,
	client runLogClient,
	runID string,
	afterSeq int,
	seen map[string]struct{},
	out io.Writer,
) (int, bool, error) {
	body, err := client.OpenRunLogs(ctx, runID, afterSeq)
	if err != nil {
		return afterSeq, false, fmt.Errorf("failed to follow run logs: %s", errors.FormatUserError(err))
	}
	defer func() { _ = body.Close() }()

	wrote := false
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		nextSeq, lineWrote, err := writeFollowLogLineWithSeen(out, scanner.Bytes(), afterSeq, seen)
		if err != nil {
			return afterSeq, wrote, err
		}
		if lineWrote {
			wrote = true
			afterSeq = nextSeq
		}
	}
	if err := scanner.Err(); err != nil {
		return afterSeq, wrote, fmt.Errorf("failed to read run logs: %w", err)
	}

	return afterSeq, wrote, nil
}

var liveLogIDPattern = regexp.MustCompile(`live-[^"]*?(\d{6})\.jsonl`)

func writeFollowLogLine(out io.Writer, line []byte, currentSeq int) (int, bool, error) {
	return writeFollowLogLineWithSeen(out, line, currentSeq, nil)
}

func writeFollowLogLineWithSeen(out io.Writer, line []byte, currentSeq int, seen map[string]struct{}) (int, bool, error) {
	trimmed := strings.TrimSpace(string(line))
	if trimmed == "" {
		return currentSeq, false, nil
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
		return currentSeq, false, fmt.Errorf("failed to decode log message: %w", err)
	}

	nextSeq := nextLogSequence(raw, currentSeq)
	if nextSeq <= currentSeq {
		nextSeq = currentSeq + 1
	}

	if seen != nil {
		key := followLogDedupeKey(raw, trimmed)
		if _, ok := seen[key]; ok {
			if nextSeq > currentSeq {
				return nextSeq, false, nil
			}
			return currentSeq, false, nil
		}
		seen[key] = struct{}{}
	}

	_, err := fmt.Fprintln(out, trimmed)
	return nextSeq, true, err
}

func followLogDedupeKey(raw map[string]any, fallback string) string {
	if id, ok := raw["id"].(string); ok && id != "" {
		return "id:" + id
	}
	return "line:" + fallback
}

func nextLogSequence(raw map[string]any, currentSeq int) int {
	for _, key := range []string{"seq", "sequence", "latest_seq"} {
		if value, ok := raw[key].(float64); ok {
			return int(value)
		}
	}

	if id, ok := raw["id"].(string); ok {
		if matches := liveLogIDPattern.FindStringSubmatch(id); len(matches) == 2 {
			var seq int
			if _, err := fmt.Sscanf(matches[1], "%d", &seq); err == nil {
				return seq
			}
		}
	}

	return currentSeq
}
