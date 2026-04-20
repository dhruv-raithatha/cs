# Implementation Plan: cs — Claude Session Manager

**Branch**: `001-cs-tmux-session-manager` | **Date**: 2026-04-20 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `specs/001-cs-tmux-session-manager/spec.md`

## Summary

`cs` is a single statically-linked Go binary that wraps tmux and fzf to give developers a
frictionless way to create, resume, and delete Claude Code sessions without any tmux knowledge.
The tool isolates all its sessions on a dedicated tmux socket (`~/.local/share/cs/cs.sock`),
presents them through an fzf picker, and requires a session name on creation (no defaults).
A `cs setup` subcommand handles first-run dependency verification.

## Technical Context

**Language/Version**: Go 1.24+ (version pinned in `go.mod`)  
**Primary Dependencies**: `github.com/urfave/cli/v3` (CLI framework), `github.com/stretchr/testify` (assertions only)  
**Storage**: None — all state derived at runtime from the cs tmux socket server  
**Testing**: `go test`, `testify/assert` + `testify/require`  
**Target Platform**: macOS (darwin/arm64) primary; Linux (linux/amd64) cross-compile target  
**Project Type**: CLI binary  
**Performance Goals**: fzf picker interactive in <1 second with up to 50 sessions (SC-004)  
**Constraints**: No config file, no network, no CGO (`CGO_ENABLED=0`), single static binary  
**Scale/Scope**: Single-user local tool; up to 50 sessions; zero remote state

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Modern Go, Version-Locked | ✅ Pass | go.mod targets Go 1.24+; `/use-modern-go` MUST be invoked before any Go code generation |
| II. TDD Red-Green-Refactor | ✅ Pass | test tasks precede every impl task in tasks.md; 80% coverage gate enforced |
| III. Interface at the Boundary | ✅ Pass | `TmuxClient`, `FuzzySelector`, `Executor` interfaces defined in data-model.md; unit tests use fakes only |
| IV. Structured Contextual Logging | ✅ Pass | `log/slog` passed explicitly; no global logger; `fmt.Println` reserved for user-facing stdout |
| V. Explicit Error Handling | ✅ Pass | `fmt.Errorf("…: %w", err)` at every layer; no panic on user input |
| VI. Minimal Dependency Footprint | ✅ Pass | urfave/cli (zero external deps) + testify only; fzf invoked via `os/exec` |
| VII. CLI Composability | ✅ Pass | exit codes 0/1/2; `--help` on all commands; `--json` on list; `CS_*` env vars; TTY detection |

**No violations. Proceeding to design.**

**Post-Phase-1 re-check**: All principles upheld. Interface boundaries defined in data-model.md.
Project structure uses internal packages with injected dependencies. No new violations introduced.

## Project Structure

### Documentation (this feature)

```text
specs/001-cs-tmux-session-manager/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── cli-schema.md    # Phase 1 output
└── tasks.md             # Phase 2 output (from /speckit-tasks)
```

### Source Code (repository root)

```text
cmd/
└── cs/
    └── main.go               # Entry point — wires CLI app and injects dependencies

internal/
├── tmux/
│   ├── client.go             # TmuxClient interface definition
│   ├── exec.go               # execTmuxClient: real implementation via os/exec
│   └── exec_test.go          # Unit tests using FakeTmuxClient
├── fzf/
│   ├── selector.go           # FuzzySelector interface definition
│   ├── exec.go               # execFuzzySelector: real implementation via os/exec
│   └── exec_test.go
├── session/
│   ├── session.go            # Session type, SessionStatus enum
│   ├── manager.go            # SessionManager — business logic (depends on TmuxClient)
│   └── manager_test.go       # Table-driven tests with FakeTmuxClient
└── setup/
    ├── checker.go            # DependencyChecker — checks PATH, offers install
    └── checker_test.go

cli/
├── root.go                   # cs (default command — picker flow)
├── setup.go                  # cs setup
├── list.go                   # cs list [--json]
├── attach.go                 # cs attach <name>
├── delete.go                 # cs delete <name>
└── version.go                # cs version

go.mod
go.sum
Makefile
.golangci.yml
```

**Structure Decision**: Single-project layout with `internal/` for all business logic packages.
`cli/` package wires urfave/cli commands to the internal layer. `cmd/cs/main.go` creates concrete
implementations and injects them into the CLI layer. No monorepo, no submodules.

## Complexity Tracking

No constitution violations requiring justification.
