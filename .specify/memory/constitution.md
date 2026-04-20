<!--
SYNC IMPACT REPORT
==================
Version change: [unversioned template] → 1.0.0
Type of bump: MAJOR (initial ratification — all content is new)

Added sections:
  - I. Project Identity
  - II. Core Principles (7 principles)
  - III. Technology Invariants
  - IV. Quality Gates
  - V. Scope Constraints
  - VI. Amendment Governance

Modified principles: N/A (initial ratification)
Removed sections: N/A (initial ratification)

Template consistency updates:
  ✅ .specify/templates/tasks-template.md — updated test task note to reflect mandatory TDD
  ✅ .specify/templates/plan-template.md — Constitution Check section already generic; no change needed
  ✅ .specify/templates/spec-template.md — no changes required

Deferred TODOs: None
-->

# cs (claude-sessions) Constitution

## I. Project Identity

**Name:** `cs` — claude-sessions

**Purpose:** A developer-focused CLI binary that manages multiple Claude Code instances using tmux
as the session backend and fzf as the interactive selection surface. The tool allows a developer to
create, list, attach to, and destroy Claude Code sessions from anywhere in their terminal, with
fuzzy-search navigation across all active sessions.

**Primary User:** The tool's author — a single developer using it daily as part of their local
development workflow. There is no multi-user, networked, or production deployment concern.

**Guiding metaphor:** `cs` should feel like a well-worn shell alias that happens to be a binary —
fast, opinionated, and invisible when it works.

## II. Core Principles

### I. Modern Go, Version-Locked

**Rule:** All generated code MUST use modern Go idioms as defined by the project's `go.mod`
version. Before writing any Go code, the agent MUST invoke the `/use-modern-go` skill
(JetBrains go-modern-guidelines) to detect the active Go version and apply version-appropriate
patterns.

**This means:**

- Use `slices.Contains`, `maps.Keys`, `cmp.Or`, `errors.AsType[T]` and other stdlib additions
  where applicable — never hand-roll equivalents that already exist in stdlib.
- Use `new(val)` to create pointers; never `x := val; &x`.
- Use `wg.Go(fn)` with `sync.WaitGroup`; never the
  `wg.Add(1) + go func() { defer wg.Done() }()` pattern.
- Use `range N` for integer iteration (Go 1.22+); never `for i := 0; i < N; i++`.
- Use `atomic.Bool`, `atomic.Pointer[T]` from `sync/atomic`; never raw `int32` with
  `atomic.LoadInt32`.
- Outdated patterns in code review or existing code are bugs, not style issues — fix them.

**Enforcement gate:** Every `/speckit.plan` and `/speckit.implement` phase MUST verify the
`/use-modern-go` skill has been invoked in the current session before generating Go code.

### II. Test-Driven Development, Red-Green-Refactor

**Rule:** No production code may be written before a failing test exists for it. The TDD cycle —
write a failing test, write the minimum code to pass, refactor — is mandatory for all
non-trivial logic.

**This means:**

- The task list from `/speckit.tasks` MUST place the test file creation task immediately before
  each corresponding implementation task. There is never an implementation task without a
  preceding test task.
- Trivial boilerplate (struct declarations, `main()` wiring, constant definitions) does not
  require a prior test. Everything with behavior does.
- The minimum coverage threshold is **80% per package**. A `/speckit.implement` run is not
  complete until `go test -coverprofile=coverage.out ./...` followed by
  `go tool cover -func=coverage.out` shows ≥80% across all packages with meaningful logic.
- Tests MUST be table-driven where multiple input variants exist. Single-case tests are a smell
  that the test is underspecified.
- Test files live in the same package as the code under test (`_test.go` suffix, same package
  name) for white-box unit tests. Black-box integration tests use a `_test` package suffix.

### III. Interface at the Boundary

**Rule:** Every interaction with an external system (tmux, fzf, the filesystem, `os.Exec`) MUST
be expressed as an interface in the production code. Concrete implementations are injected;
never instantiated inline in logic functions.

**This means:**

- The tmux driver, fzf selector, and any OS-level process executor each have a corresponding Go
  interface (e.g., `TmuxClient`, `FuzzySelector`, `Executor`).
- Unit tests use in-process fake/stub implementations of these interfaces. No test spawns a real
  tmux server or real fzf process — those belong only in integration or end-to-end tests, which
  are gated behind a `//go:build integration` build tag.
- Functions that contain logic MUST accept their dependencies as interface parameters (or via a
  receiver that holds injected interfaces), never call global singletons.
- If a function cannot be unit tested without spawning a subprocess, it violates this principle
  and MUST be refactored before the task is considered done.

### IV. Structured Contextual Logging, Never Global

**Rule:** All operational logging uses `log/slog` with a `*slog.Logger` passed explicitly
through function signatures. There is no global logger, no `log.Println`, no
`fmt.Fprintf(os.Stderr, ...)` for operational output.

**This means:**

- The root `*slog.Logger` is created once in `main()` and passed as the first parameter (or as
  part of a context/config struct) to all functions that perform I/O or make decisions.
- Log levels are used semantically: `Debug` for tracing execution paths (disabled in normal
  use), `Info` for meaningful lifecycle events (session created, attached, destroyed), `Warn`
  for recoverable anomalies, `Error` for failures.
- Every `slog.Log` call MUST include at least one structured attribute that identifies the
  entity being acted on (e.g., `slog.String("session", name)`). Bare string messages without
  attributes are not permitted in non-trivial log statements.
- The logger is NEVER stored in a `context.Context`. It is passed as an explicit function
  parameter or as a field on the relevant struct.
- `fmt.Println` and friends are reserved for user-facing stdout output only (e.g., the fzf
  selection prompt, the session list). Log output goes to stderr.

### V. Explicit, Contextual Error Handling

**Rule:** Every error is either handled (with a recovery strategy) or returned with context
added. Errors are never silently discarded. `panic` is never used for conditions that could
occur in normal operation.

**This means:**

- Use `fmt.Errorf("operation description: %w", err)` to wrap errors at every layer boundary,
  creating a traceable chain.
- Never assign an error to `_` unless the error is provably impossible (e.g., writing to a
  `strings.Builder`). Any suppressed error requires a comment explaining why.
- Error messages use lowercase, no trailing punctuation, and describe the operation that failed
  — not the result (e.g., `"attach to session"` not `"failed to attach"`).
- `panic` is reserved for programmer errors: nil pointer dereferences in invariant-guaranteed
  code, impossible type assertions. Never `panic` on user input, missing environment variables,
  or subprocess failures.
- The `main()` function handles top-level errors by printing a user-friendly message to stderr
  and exiting with a non-zero code. It does NOT `log.Fatal`.

### VI. Minimal, Justified Dependency Footprint

**Rule:** Each external dependency MUST be justified in the plan with a rationale. The standard
library is preferred. A new dependency is only acceptable when the stdlib alternative would
require significant hand-rolled complexity that a battle-tested library solves correctly.

**This means:**

- The expected external dependencies for this project are: a CLI framework (e.g., `cobra` or
  `urfave/cli`), and nothing else at the core logic layer. The fzf integration MUST use
  `os/exec` to invoke the fzf binary — not an fzf Go library — because the user already has
  fzf installed and shell integration is not required.
- Any proposed dependency not in the approved list above requires an explicit justification
  added to `plan.md` before the task is accepted.
- Dependencies are pinned in `go.mod` with exact versions. The `go.sum` file is always
  committed.
- Indirect dependencies are periodically reviewed; `go mod tidy` is run as part of every
  implementation phase.

### VII. CLI Composability and Unix Conventions

**Rule:** `cs` behaves as a first-class Unix citizen. Each subcommand does exactly one thing.
Machine-readable output is available on all list/query commands.

**This means:**

- Exit codes follow convention: `0` for success, `1` for user errors (invalid args, session not
  found), `2` for internal/unexpected errors.
- All subcommands accept `--help` and produce usage output that is accurate and complete.
- List/query subcommands accept a `--json` flag that emits newline-delimited JSON for piping to
  other tools.
- Interactive modes (fzf selection) are only triggered when stdout is a TTY. When piped, the
  command falls back to non-interactive behavior and outputs the full list.
- Environment variables follow the pattern `CS_<NAME>` (e.g., `CS_TMUX_SOCKET`) and are
  documented in `--help` output.

## III. Technology Invariants

The following choices are fixed for the lifetime of this project and are not re-evaluated per
feature:

- **Language:** Go, minimum version as declared in `go.mod` (target Go 1.24+).
- **Session backend:** tmux, invoked via `os/exec`. No alternatives are evaluated.
- **Selection UI:** fzf binary, invoked via `os/exec`. No alternatives are evaluated.
- **Logging library:** `log/slog` (stdlib). No third-party logging libraries.
- **CLI framework:** A single CLI framework chosen during the first `/speckit.plan`. Once
  chosen, it is not replaced mid-project.
- **Build output:** A single statically-linked binary named `cs`. No dynamic library
  dependencies. `CGO_ENABLED=0` for all release builds.
- **Test runner:** `go test` only. No external test framework (Ginkgo, testify/suite, etc.).
  `testify/assert` is acceptable for assertions; `require` is preferred over `assert` for
  fatal assertions.
- **Configuration:** No configuration file in v1. All behavior is controlled by CLI flags and
  `CS_*` environment variables.

## IV. Quality Gates

A feature is not considered complete unless all of the following are true:

1. `go build ./...` produces no errors or warnings.
2. `go vet ./...` produces no output.
3. `golangci-lint run` (with the project's `.golangci.yml`) produces no new linter violations.
4. `go test -race -coverprofile=coverage.out ./...` passes with zero failures.
5. Per-package coverage is ≥80% across all packages containing business logic (excludes `main`
   package wiring and generated code).
6. All public functions, types, and constants have a doc comment.
7. The binary cross-compiles for `linux/amd64` and `darwin/arm64` without errors
   (`GOOS=linux GOARCH=amd64 go build` and `GOOS=darwin GOARCH=arm64 go build`).
8. The feature's acceptance criteria in `spec.md` have been manually verified against the
   running binary.

## V. Scope Constraints

`cs` v1 explicitly does NOT:

- Manage tmux sessions unrelated to Claude Code (it filters by a naming convention or marker).
- Provide a TUI (terminal UI) beyond what fzf offers natively.
- Sync session state across machines or persist session metadata to a remote store.
- Require network access.
- Manage Claude Code configuration, API keys, or per-project Claude settings.
- Support Windows (tmux is not natively available on Windows).

Any spec that introduces functionality outside these constraints requires a constitutional
amendment before it is accepted into a plan.

## Governance

**Amendment process:**

1. Run `/speckit.constitution <description of proposed change>` to draft the amendment.
2. The version is bumped according to semantic versioning:
   - **MAJOR** — a principle is removed, fundamentally reworded, or a technology invariant is
     changed.
   - **MINOR** — a new principle or quality gate is added; existing principles are clarified
     without changing their decision impact.
   - **PATCH** — wording improvements, typo fixes, example updates with no behavioral change.
3. After amendment, `/speckit.analyze` MUST be run to verify downstream artifacts
   (`plan-template.md`, `spec-template.md`, `tasks-template.md`) are consistent with the
   updated principles.
4. Amendments are committed as a standalone commit with the message format:
   `docs: amend constitution to vX.Y.Z — <one-line rationale>`.

**When to amend:**

- A new external system dependency is introduced (triggers Principle VI review).
- The Go version in `go.mod` is upgraded (triggers Principle I review and `/use-modern-go`
  re-invocation).
- A recurring implementation decision reveals a gap in the constitution — add a clarifying
  principle rather than making the same decision twice.
- A principle is found to be routinely ignored or impossible to enforce — remove or reword it
  rather than keeping dead text.

**Version**: 1.0.0 | **Ratified**: 2026-04-20 | **Last Amended**: 2026-04-20
