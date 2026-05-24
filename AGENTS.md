# AGENTS.md - RepoBird CLI Project Guidelines

## Project Context
RepoBird CLI is a Go-based command-line tool for interacting with the RepoBird AI platform. It lets users submit AI-powered code generation tasks, track progress, and manage runs through CLI commands and a Bubble Tea TUI.

RepoBird is currently in a large `repobird-next` migration from Claude-oriented workflows to OpenCode-oriented workflows. Treat OpenCode as the forward-looking agent workflow unless existing CLI/API compatibility requires otherwise.

Product behavior should reflect credits-based runs. Do not model usage, quotas, pricing, or run availability around Basic/Pro run-count amounts unless the current product/API contract explicitly requires it for compatibility.

## Source Of Truth
- Keep this file and `CLAUDE.md` aligned when project-critical agent guidance changes.
- Core project docs live in `docs/`.
- Existing architecture and TUI patterns in `CLAUDE.md` remain applicable to agents working in this repo.

## Development Guidelines
- Follow Go conventions and project-local patterns.
- Read existing code before changing behavior.
- Keep functions short, focused, and explicit about errors.
- Preserve CLI backward compatibility unless the task explicitly changes it.
- Use message-based navigation in the TUI; route transitions through `internal/tui/app.go`.
- Never log sensitive data such as API keys or tokens.

## Validation
- Run focused tests for changed code first.
- For Go changes, run `make test` before handoff when feasible.
- For CLI command changes under `internal/commands/`, also run `make test-integration` when feasible.
- For significant changes, run `make check` before finalizing when practical.
