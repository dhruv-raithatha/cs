# Tasks: cs — Claude Session Manager

**Input**: Design documents from `specs/001-cs-tmux-session-manager/`  
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, contracts/cli-schema.md ✓

**Tests**: Per Constitution Principle II (TDD), a test task immediately precedes every implementation task containing behavioral logic. Trivial boilerplate (interface declarations, struct definitions, constant enums, main() wiring) does not require a prior test.

**Format**: `[ID] [P?] [Story?] Description — file path`

---

## Phase 1: Setup (Project Initialization)

**Purpose**: Create a compilable Go project skeleton with all tooling in place.

- [x] T001 Invoke `/use-modern-go` skill (JetBrains go-modern-guidelines) to detect the active Go version and apply version-appropriate patterns for this session — required by Constitution Principle I before any Go code is written
- [x] T002 Create directory structure: `cmd/cs/`, `internal/tmux/`, `internal/fzf/`, `internal/session/`, `internal/setup/`, `cli/`
- [x] T003 Initialize `go.mod` with module path `github.com/dhruv/cs`, Go 1.24+; add `github.com/urfave/cli/v3` and `github.com/stretchr/testify`; run `go mod tidy` — `go.mod`, `go.sum`
- [x] T004 [P] Create `Makefile` with targets: `build` (CGO_ENABLED=0), `test`, `test-integration` (-tags integration), `lint`, `cross-compile` (linux/amd64 + darwin/arm64), `coverage` — `Makefile`
- [x] T005 [P] Create `.golangci.yml` enabling: `errcheck`, `govet`, `staticcheck`, `gosimple`, `exhaustive`, `godot` (doc comments); configure exclusions for `main` package and `_test.go` files — `.golangci.yml`

---

## Phase 2: Foundation (Blocking Prerequisites)

**Purpose**: Interfaces, types, fakes, real exec implementations, and CLI skeleton. No user story work starts until this phase is complete.

**⚠️ CRITICAL**: All user story phases depend on this phase.

- [x] T006 [P] Define `TmuxClient` interface with methods: `ListSessions`, `NewSession`, `AttachSession`, `KillSession`, `HasSession` — `internal/tmux/client.go`
- [x] T007 [P] Define `FuzzySelector` interface with method `Select(items []string, prompt, header string) (string, error)` — `internal/fzf/selector.go`
- [x] T008 [P] Define `Session` struct and `SessionStatus` enum (`Active`, `Dead`) — `internal/session/session.go`
- [x] T009 [P] Implement `FakeTmuxClient` (all methods configurable via fields; returns pre-set data or errors) for use in unit tests — `internal/tmux/fake.go`
- [x] T010 [P] Implement `FakeFuzzySelector` (returns a pre-set selection string or error) for use in unit tests — `internal/fzf/fake.go`
- [x] T011 Wire urfave/cli app skeleton in `cmd/cs/main.go` and implement `cs version` (prints `cs version v0.1.0`) in `cli/version.go`; verify `go build ./cmd/cs` succeeds — `cmd/cs/main.go`, `cli/version.go`
- [x] T012 Write integration tests (build tag `//go:build integration`) for `execTmuxClient` covering `ListSessions`, `NewSession`, `AttachSession`, `KillSession`, `HasSession` against a real tmux server on the cs socket — `internal/tmux/exec_test.go`
- [x] T013 Implement `execTmuxClient` using `tmux -S <socketPath> <subcommand>` via `os/exec`; derive `SessionStatus` from `#{pane_current_command}` format string — `internal/tmux/exec.go`
- [x] T014 Write integration tests (build tag `//go:build integration`) for `execFuzzySelector` verifying it pipes items to fzf and returns the selected line — `internal/fzf/exec_test.go`
- [x] T015 Implement `execFuzzySelector` piping item list to fzf stdin via `strings.NewReader`; capture selected line from stdout after `cmd.Wait()`; use flags `--height 40%`, `--no-sort` — `internal/fzf/exec.go`

**Checkpoint**: `go build ./...` succeeds, `cs version` runs, integration tests pass with `make test-integration`

---

## Phase 3: User Story 0 — Initial Setup (Priority: P0)

**Goal**: `cs setup` checks and optionally installs all required dependencies; creates the cs data directory.

**Independent Test**: Run `cs setup` on a machine with at least one dep missing; verify it reports the missing dep and offers install. Then run again with all deps present; verify all-pass output and `~/.local/share/cs/` exists.

- [x] T016 [US0] Write unit tests for `DependencyChecker` in `internal/setup/checker_test.go` using table-driven cases: dep on PATH (found + version), dep missing (not found), directory creation when dir absent, directory skip when dir present — `internal/setup/checker_test.go`
- [x] T017 [US0] Implement `DependencyChecker` in `internal/setup/checker.go`: check `tmux`, `fzf`, `claude` on PATH via `exec.LookPath`; capture version strings; prompt Homebrew (`brew install <dep>`) or npm (`npm i -g @anthropic-ai/claude-code`) install per dep; create `~/.local/share/cs/` using `os.MkdirAll` — `internal/setup/checker.go`
- [x] T018 [US0] Implement `cs setup` subcommand in `cli/setup.go` wiring `DependencyChecker`; print ✓/✗ status per dep; exit 0 on all-pass, exit 1 if any dep remains missing — `cli/setup.go`
- [x] T019 [US0] Register `cs setup` in `cmd/cs/main.go`; run `cs setup` manually end-to-end per acceptance scenarios in spec.md US0 — `cmd/cs/main.go`

**Checkpoint**: `cs setup` reports all deps and creates `~/.local/share/cs/` on first run

---

## Phase 4: User Story 1 — Launch or Resume a Session (Priority: P1) 🎯 MVP

**Goal**: `cs` with no args opens an fzf picker showing all cs sessions; user can attach to an existing one.

**Independent Test**: Run `cs` with zero sessions → confirm create-new prompt. Run `cs` with one or more sessions → confirm fzf picker appears and selecting one attaches. Run `cs` while already inside tmux → confirm error message and exit 1.

- [x] T020 [P] [US1] Write unit tests for `SessionManager.List` in `internal/session/manager_test.go` using `FakeTmuxClient`: verifies `Active` status when pane command is `claude`; verifies `Dead` status when pane command is a shell; verifies empty list returns nil error — `internal/session/manager_test.go`
- [x] T021 [US1] Implement `SessionManager` with `List(socketPath string) ([]Session, error)` in `internal/session/manager.go`; call `TmuxClient.ListSessions`; derive `SessionStatus` from `PaneCommand` — `internal/session/manager.go`
- [x] T022 [P] [US1] Write unit tests for `run()` picker orchestration in `cli/root_test.go` using `FakeTmuxClient` and `FakeFuzzySelector`: covers no-sessions path (creates new), has-sessions attach path, already-inside-tmux guard (TMUX env var set → returns error before any tmux call) — `cli/root_test.go`
- [x] T023 [US1] Implement root command picker flow in `cli/root.go`: (1) guard — if `os.Getenv("TMUX") != ""` print error and exit 1; (2) call `SessionManager.List`; (3) if 0 sessions go to create path; (4) otherwise build fzf item list with format `<name>   <dir>   [dead]?`, add `[ + new session ]` entry, call `FuzzySelector.Select`; (5) on existing session selected call `TmuxClient.AttachSession` — `cli/root.go`
- [x] T024 [US1] Inject `execTmuxClient` and `execFuzzySelector` into root command in `cmd/cs/main.go`; manually verify US1 acceptance scenarios 1–4 from spec.md — `cmd/cs/main.go`

**Checkpoint**: `cs` opens fzf picker; selecting an existing session attaches; error shown if already in tmux

---

## Phase 5: User Story 2 — Create a Named Session (Priority: P2)

**Goal**: From the picker, user selects "new session", provides a mandatory name, and is attached to a new Claude session.

**Independent Test**: Select "new session" in picker, provide a name → verify new tmux session is created and Claude starts. Provide empty name → verify re-prompt. Provide a name that already exists → verify redirect to attach.

- [x] T025 [P] [US2] Write unit tests for `SessionManager.NewSession` in `internal/session/manager_test.go`: successful create on cs socket; empty name returns error; name already exists (`HasSession` returns true) triggers attach flow — `internal/session/manager_test.go`
- [x] T026 [US2] Implement `SessionManager.NewSession(socketPath, name, workingDir string) error` in `internal/session/manager.go`: call `TmuxClient.HasSession`; if exists call `AttachSession` (redirect per FR-012); otherwise call `TmuxClient.NewSession` then `AttachSession`; reject empty name with error — `internal/session/manager.go`
- [x] T027 [P] [US2] Write unit tests for name-prompt loop in `cli/root_test.go`: empty input re-prompts (simulated via `FakeFuzzySelector` sequence); collision triggers attach path — `cli/root_test.go`
- [x] T028 [US2] Implement mandatory name-prompt in create path of `cli/root.go`: when user selects `[ + new session ]`, use `bufio.Scanner` on stdin to read name; re-prompt if empty; call `SessionManager.NewSession`; `claude` started as initial tmux command via `TmuxClient.NewSession` with arg `'claude'` — `cli/root.go`
- [x] T029 [US2] Manual verification: create two sessions with distinct names, detach from each, run `cs` and confirm both appear with correct names and working directories (US2 acceptance scenarios 1–3) — manual test step

**Checkpoint**: Create flow requires a non-empty name; collision redirects to attach; new session starts Claude

---

## Phase 6: User Story 3 — Delete a Stale Session (Priority: P3)

**Goal**: From the picker, user presses the delete keybind on any session, confirms, and the session is removed.

**Independent Test**: Create a session, run `cs`, trigger delete keybind on it, confirm → verify session gone. Trigger delete and cancel → verify session still present. Also verify `cs delete <name>` works non-interactively.

- [x] T030 [P] [US3] Write unit tests for `SessionManager.Kill` in `internal/session/manager_test.go` using `FakeTmuxClient`: kill existing session succeeds; kill non-existent session returns error — `internal/session/manager_test.go`
- [x] T031 [US3] Implement `SessionManager.Kill(socketPath, name string) error` in `internal/session/manager.go` wrapping `TmuxClient.KillSession` — `internal/session/manager.go`
- [x] T032 [P] [US3] Write unit tests for delete confirmation flow in `cli/root_test.go`: confirm (`y`) calls `SessionManager.Kill` and returns picker; cancel (`n`) leaves session intact — `cli/root_test.go`
- [x] T033 [US3] Implement fzf `--bind ctrl-d:execute(...)` delete keybinding in `cli/root.go` picker invocation: on ctrl-d prompt `"Delete session '<name>'? [y/N]: "` to stderr; on `y` call `SessionManager.Kill`; refresh picker; on `n` return to picker unchanged — `cli/root.go`
- [x] T034 [US3] Implement `cs delete <name>` non-interactive subcommand in `cli/delete.go`: call `SessionManager.Kill`; print success or "session not found"; exit 0/1 accordingly — `cli/delete.go`
- [x] T035 [US3] Register `cs delete` in `cmd/cs/main.go`; manually verify US3 acceptance scenarios 1–2 from spec.md — `cmd/cs/main.go`

**Checkpoint**: Delete keybind removes confirmed sessions; `cs delete <name>` works non-interactively

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Non-interactive subcommands, env var wiring, coverage gate, lint, and cross-compile.

- [x] T036 [P] Implement `cs list [--json]` in `cli/list.go`: default tabular output (`NAME  WORKING_DIR  STATUS`); `--json` flag emits newline-delimited JSON `{"name":…,"working_dir":…,"status":…}`; TTY detection — `cli/list.go`
- [x] T037 [P] Implement `cs attach <name>` non-interactive subcommand in `cli/attach.go`: call `SessionManager.Attach`; guard against already-in-tmux (same as root command); exit 0/1 — `cli/attach.go`
- [x] T038 Wire `CS_TMUX_SOCKET` env var as default for `--socket` global flag in `cmd/cs/main.go`; verify flag override takes precedence over env var — `cmd/cs/main.go`
- [x] T039 [P] Add `--help` doc to all subcommands documenting all flags and `CS_TMUX_SOCKET` env var; verify `cs --help`, `cs setup --help`, `cs list --help`, `cs attach --help`, `cs delete --help` all produce accurate output per Constitution Principle VII — `cli/*.go`
- [x] T040 Run `go test -race -coverprofile=coverage.out ./...` and `go tool cover -func=coverage.out`; fix until ≥80% per package with business logic (Quality Gate 5) — all `internal/` packages
- [x] T041 Run `golangci-lint run` and resolve all violations per `.golangci.yml` until output is clean (Quality Gate 3) — all packages
- [x] T042 [P] Cross-compile: `GOOS=linux GOARCH=amd64 go build ./cmd/cs` and `GOOS=darwin GOARCH=arm64 go build ./cmd/cs`; both must succeed with zero errors (Quality Gate 7) — `cmd/cs/main.go`
- [x] T043 Run `quickstart.md` end-to-end validation on the built binary: `cs setup`, create two sessions, list, attach, delete; confirm all acceptance scenarios from spec.md US0–US3 pass — manual validation step

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundation)**: Depends on Phase 1 — **BLOCKS all user story phases**
- **Phase 3 (US0 — cs setup)**: Depends on Phase 2
- **Phase 4 (US1 — picker/attach)**: Depends on Phase 2; Phase 3 recommended first (setup needed to run the binary)
- **Phase 5 (US2 — create)**: Depends on Phase 4 (create flow extends the picker)
- **Phase 6 (US3 — delete)**: Depends on Phase 4 (delete flow extends the picker)
- **Phase 7 (Polish)**: Depends on Phases 3–6 complete

### User Story Dependencies

- **US0 (P0)**: Can start after Foundation — independent of US1/US2/US3
- **US1 (P1)**: Can start after Foundation — is the picker core that US2 and US3 extend
- **US2 (P2)**: Depends on US1 picker existing (create is a path within the picker)
- **US3 (P3)**: Depends on US1 picker existing (delete keybind is in the picker)

### Within Each Phase

- Tests MUST be written first and MUST FAIL before implementation begins (Constitution Principle II)
- `FakeTmuxClient` and `FakeFuzzySelector` are used in all unit tests — no real tmux/fzf in unit tests
- Integration tests (real tmux/fzf) use `//go:build integration` and run via `make test-integration`
- Interfaces before implementations within Foundation
- `SessionManager` before CLI commands

### Parallel Opportunities

- T004 and T005 (Makefile, golangci.yml) can run in parallel within Phase 1
- T006, T007, T008 (interfaces + Session type) can all run in parallel within Phase 2
- T009 and T010 (fakes) can run in parallel within Phase 2
- T012 and T014 (integration tests for exec impls) can run in parallel within Phase 2
- T013 and T015 (exec implementations) can run in parallel within Phase 2
- T020 and T022 (test tasks for US1) can run in parallel within Phase 4
- T025 and T027 (test tasks for US2) can run in parallel within Phase 5
- T030 and T032 (test tasks for US3) can run in parallel within Phase 6
- T036, T037, T039 (polish subcommands + help) can run in parallel within Phase 7

---

## Parallel Example: Phase 2 Foundation

```
# Run in parallel (separate files, no dependencies):
T006: internal/tmux/client.go         — TmuxClient interface
T007: internal/fzf/selector.go        — FuzzySelector interface
T008: internal/session/session.go     — Session type + SessionStatus

# Then in parallel:
T009: internal/tmux/fake.go           — FakeTmuxClient
T010: internal/fzf/fake.go            — FakeFuzzySelector

# Then sequentially (exec impl after test):
T012: internal/tmux/exec_test.go      — integration test (write + fail)
T013: internal/tmux/exec.go           — implementation (make pass)

T014: internal/fzf/exec_test.go       — integration test (write + fail)
T015: internal/fzf/exec.go            — implementation (make pass)
```

## Parallel Example: Phase 4 (US1)

```
# Run in parallel (separate files):
T020: internal/session/manager_test.go  — SessionManager.List tests
T022: cli/root_test.go                  — picker orchestration tests

# Then:
T021: internal/session/manager.go       — SessionManager.List impl
T023: cli/root.go                       — picker flow impl
T024: cmd/cs/main.go                    — wire and verify
```

---

## Implementation Strategy

### MVP First (US0 + US1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundation (CRITICAL — blocks all stories)
3. Complete Phase 3: US0 (`cs setup`)
4. Complete Phase 4: US1 (picker + attach)
5. **STOP and VALIDATE**: Run `cs setup` + `cs` picker manually — attach to an existing session
6. Ship MVP: zero-config session management works

### Incremental Delivery

1. Setup + Foundation → compilable skeleton with `cs version`
2. + US0 → `cs setup` handles all dependency onboarding
3. + US1 → `cs` picker attach flow (MVP!)
4. + US2 → create named sessions from picker
5. + US3 → delete sessions from picker
6. + Polish → `cs list`, `cs attach`, `cs delete`, coverage gate, lint clean

---

## Notes

- `[P]` tasks operate on different files with no blocking dependencies — safe to run in parallel
- `[USN]` label maps each task to its user story for traceability
- All unit tests use `FakeTmuxClient` / `FakeFuzzySelector` — never spawn real tmux/fzf in unit tests
- Integration tests tagged `//go:build integration` run separately via `make test-integration`
- `go test -race ./...` must pass with zero failures before any phase is considered done
- Commit after each phase checkpoint; tag MVP after Phase 4 completes
