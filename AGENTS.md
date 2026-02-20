# AGENTS.md

Guidance for coding agents working in this repository.

## Project Overview

- Language: Go
- Module: `github.com/viktor/asb-tui`
- App type: terminal UI (Bubble Tea + Bubbles + Lip Gloss)
- Entrypoint: `cmd/asb-tui/main.go`
- Main packages:
  - `internal/config` for env parsing and validation
  - `internal/asb` for Azure Service Bus client interactions
  - `internal/ui` for Bubble Tea model, update loop, and rendering
  - `internal/ui/style` for style definitions

## Prerequisites

- Go toolchain (see `go.mod` for version baseline)
- Azure CLI installed (`az`) when running against real Azure resources
- Authenticated Azure session (`az login`) for live queue data access
- `ASB_NAMESPACE` set when launching the application

## Build, Run, Lint, and Test Commands

### Run the app

- `go run ./cmd/asb-tui`

### Build

- Build all packages: `go build ./...`
- Build binary: `go build -o bin/asb-tui ./cmd/asb-tui`

### Format (required before finalizing changes)

- Format entire repo: `gofmt -w .`
- Quick check for unformatted files: `gofmt -l .`

### Lint/static checks

There is no repo-local `golangci` or Makefile task configured. Use Go-native checks:

- `go vet ./...`
- `go test ./...`

Optional (only if globally installed in your environment):

- `golangci-lint run`

### Tests

- Run all tests: `go test ./...`
- Run tests in one package: `go test ./internal/ui`
- Run one test by exact name:
  - `go test ./internal/ui -run '^TestNewModelInitializesState$'`
- Run one test with verbose output:
  - `go test -v ./internal/ui -run '^TestStatusBarIsSingleLine$'`
- Run multiple related tests by regex:
  - `go test ./internal/ui -run 'TestKey.*Refreshes.*'`

## Environment and Runtime Notes

- Required env var: `ASB_NAMESPACE`
- Optional env vars:
  - `ASB_REFRESH_SECONDS` (default 10)
  - `ASB_ACTIVE_WARN_THRESHOLD` (default disabled via `-1`)
  - `ASB_DLQ_WARN_THRESHOLD` (default disabled via `-1`)
- For local live runs, the common sequence is:
  1. `az login`
  2. export `ASB_NAMESPACE=...`
  3. `go run ./cmd/asb-tui`

## Code Style and Conventions

### Formatting

- Always use `gofmt` formatting.
- Avoid manual alignment changes that fight `gofmt`.
- Keep files ASCII unless the file already relies on Unicode.

### Imports

- Follow Go import grouping/order as produced by `gofmt`:
  1. standard library
  2. third-party modules
  3. this module's internal imports
- Prefer minimal imports; remove unused imports immediately.

### Types and data modeling

- Use explicit structs for domain state (`Config`, `QueueSnapshot`, `QueueMetrics`).
- Keep field names descriptive and consistent across translation layers.
- Use `int64` where queue metrics/counters are involved.
- Favor zero-value-safe struct behavior when practical.

### Naming

- Exported identifiers: `PascalCase`.
- Unexported identifiers: `camelCase`.
- Package names: short, lowercase, no underscores.
- Test names: `TestXxxBehavior` style, focusing on observable behavior.

### Functions and control flow

- Prefer early returns for validation and error branches.
- Keep functions focused on one responsibility.
- For Bubble Tea update logic, mutate model state predictably and return `tea.Cmd` clearly.
- Keep helper functions small (`clip`, `max`, `thresholdExceeded`) and side-effect-free.

### Error handling

- Return errors instead of panicking for expected failure paths.
- Wrap errors with context using `%w` when propagating (`fmt.Errorf("context: %w", err)`).
- In CLI entrypoints, print user-actionable errors to stderr and exit non-zero.
- Preserve existing state on refresh failures when possible (current UI pattern).

### Context and timeouts

- Pass `context.Context` into network/SDK calls.
- Use bounded timeouts for external calls (current pattern uses 8 seconds).
- Ensure cancellations are deferred (`defer cancel()`).

### Testing expectations

- Prefer standard library `testing` package.
- Keep tests deterministic and isolated; avoid real Azure calls in unit tests.
- Use focused assertions with clear failure messages.
- Validate both success and error paths.
- For stateful UI, assert state transitions and command presence where relevant.

## Repository-Specific Implementation Guidance

- Keep package boundaries intact:
  - config parsing in `internal/config`
  - Azure access in `internal/asb`
  - presentation/state in `internal/ui`
- Preserve keybinding behavior unless intentionally changing UX.
- When adding new config, thread it through `config.Config` and `NewModel` explicitly.
- Prefer extending existing UI state types over introducing parallel state models.

## Agent Workflow Checklist

Before finishing a change:

1. Run `gofmt -w .`
2. Run targeted tests for changed package(s)
3. Run `go test ./...`
4. Run `go vet ./...`
5. Verify app still starts: `go run ./cmd/asb-tui` (when applicable)

## Cursor and Copilot Rules

No Cursor rules or Copilot instruction files were found at the time this file was created:

- `.cursorrules` (not found)
- `.cursor/rules/` (not found)
- `.github/copilot-instructions.md` (not found)

If any of these files are added later, treat them as higher-priority repository guidance and update this document accordingly.
