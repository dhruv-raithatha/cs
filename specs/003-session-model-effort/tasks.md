# Tasks: Session Model & Effort Selection

**Input**: Design documents from `specs/003-session-model-effort/`
**Prerequisites**: plan.md ✅ spec.md ✅ research.md ✅ data-model.md ✅ contracts/ ✅ quickstart.md ✅

**Tests**: Per constitution Principle II (TDD), every test task MUST be written first and confirmed
to FAIL before its corresponding implementation task is started.

**Organization**: Tasks are grouped by phase. Foundational phase unblocks all user story phases.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no conflicting dependencies)
- **[US1]**: Creation-time model/effort prompts + env-var pre-selection (spec US1 + US3)
- **[US2]**: Picker and list display of model/effort (spec US2)

---

## Phase 1: Setup

**Purpose**: Confirm modern-Go idiom baseline before writing any code.

- [X] T001 Invoke `/use-modern-go` skill to confirm Go 1.26.2 patterns (slices package, range-over-int, etc.) — no code written until this passes

---

## Phase 2: Foundational — Session Struct & Interface Updates

**Purpose**: Extend the shared `Session` type and update every interface that carries or creates sessions. Nothing in Phase 3 or beyond can compile until these are done.

**⚠️ CRITICAL**: All user story phases depend on this phase completing fully.

- [X] T002 Write failing table-driven tests asserting `Session.Model` and `Session.Effort` fields exist and are zero-value for unset sessions in `internal/session/session_test.go`
- [X] T003 Add `Model string` and `Effort string` fields to `Session` struct in `internal/session/session.go` (makes T002 pass)
- [X] T004 [P] Update `TmuxClient.NewSession` signature to `(socketPath, name, workingDir, model, effort string) error` in `internal/tmux/client.go`
- [X] T005 [P] Update `FakeTmuxClient.NewSession` to match new signature; add `CreatedModel string` and `CreatedEffort string` fields to capture test assertions in `internal/tmux/fake.go`
- [X] T006 Write failing table-driven tests for `Manager.NewSession` asserting model and effort are forwarded to the underlying client (use `FakeTmuxClient.CreatedModel`/`CreatedEffort`) in `internal/session/manager_test.go`
- [X] T007 Update `session.Client` interface `NewSession` signature to `(socketPath, name, workingDir, model, effort string) error`; update `Manager.NewSession` to accept and forward both params in `internal/session/manager.go` (makes T006 pass)

**Checkpoint**: `go build ./...` compiles with zero errors after T007. All existing tests pass.

---

## Phase 3: User Story 1 + User Story 3 — Creation-Time Prompts (Priority: P1)

**Goal**: Two fzf prompts (model then effort) are added after the session name input. The default
item in each prompt is the env-var value (`ANTHROPIC_MODEL` / `CLAUDE_CODE_EFFORT_LEVEL`) or the
built-in fallback (`sonnet` / `medium`). Chosen values are set on the tmux session as both `-e`
environment flags and `@cs-*` session options.

**Independent Test**: Create a new session with no existing sessions, select `opus` + `high` in
the two prompts, and verify `FakeTmuxClient.CreatedModel == "opus"` and `CreatedEffort == "high"`.
Also verify env-var pre-selection by setting `ANTHROPIC_MODEL=haiku` and asserting `haiku` is the
first item passed to the model `Select()` call.

### Tests for US1 + US3 ⚠️ Write first — confirm FAIL before implementing

- [X] T008 [P] [US1] Write failing tests for `ListSessions` parsing 5-part tmux format: assert `session.Model` and `session.Effort` are populated from `#{@cs-model}` / `#{@cs-effort}`, empty-string for sessions without options in `internal/tmux/exec_test.go`
- [X] T012 [P] [US1] Write failing table-driven tests for `createNewSession` covering: (a) fzf returns model + effort → passed to `FakeTmuxClient`; (b) fzf returns `""` for both → falls back to env-var or built-in default; (c) `ANTHROPIC_MODEL=haiku` set → `haiku` is first item in model `Select()` call; (d) `CLAUDE_CODE_EFFORT_LEVEL=xhigh` set → `xhigh` is first effort item in `Select()` call in `cli/root_test.go`

### Implementation for US1 + US3

- [X] T009 [US1] Update `ListSessions` format string to `#{session_name}:#{session_path}:#{pane_current_command}:#{@cs-model}:#{@cs-effort}`; change `strings.SplitN` to 5 parts; populate `Session.Model` and `Session.Effort` (empty string when option unset) in `internal/tmux/exec.go` (makes T008 pass)
- [X] T010 [US1] Write failing tests for `execTmuxClient.NewSession` asserting the tmux command includes `-e ANTHROPIC_MODEL=<model>` and `-e CLAUDE_CODE_EFFORT_LEVEL=<effort>` flags, and that two `set-option` calls are made for `@cs-model` and `@cs-effort` in `internal/tmux/exec_test.go`
- [X] T011 [US1] Update `execTmuxClient.NewSession` to: (1) append `-e ANTHROPIC_MODEL=<model> -e CLAUDE_CODE_EFFORT_LEVEL=<effort>` to `new-session` args; (2) after successful create, run `tmux set-option -t <name> @cs-model <model>` and `tmux set-option -t <name> @cs-effort <effort>` in `internal/tmux/exec.go` (makes T010 pass)
- [X] T013 [US1] Add `knownModels []string` and `knownEfforts []string` package-level vars; add `orderedWithDefault(list []string, def string) []string` helper (moves matching item to front, no-op if not found); add `pickModel(selector fzf.FuzzySelector) (string, error)` and `pickEffort(selector fzf.FuzzySelector) (string, error)` helpers reading env vars for default in `cli/root.go`
- [X] T014 [US1] Update `createNewSession` signature to `(socketPath string, client tmux.TmuxClient, selector fzf.FuzzySelector, stdin io.Reader) error`; insert `pickModel` then `pickEffort` calls after name confirmation; pass results to `mgr.NewSession` in `cli/root.go` (makes T012 pass)
- [X] T015 [US1] Update the two `createNewSession` call sites — in `runWithConfirmReader` (no-sessions path) and `runPicker` (new-session branch) — to pass `selector` as the third argument in `cli/root.go`

**Checkpoint**: `go test ./...` passes. `FakeTmuxClient.CreatedModel` and `CreatedEffort` are populated on every new-session path.

---

## Phase 4: User Story 2 — Picker & List Display (Priority: P1)

**Goal**: Every session entry in the interactive picker shows model and effort alongside name and
working directory. Pre-existing sessions (empty model/effort) display `"unknown"` rather than
blank fields.

**Independent Test**: Construct a session with `Model: "opus"` and `Effort: "high"` and one with
`Model: ""` and assert the picker entry contains `opus` + `high` for the first and `unknown` +
`unknown` for the second.

### Tests for US2 ⚠️ Write first — confirm FAIL before implementing

- [X] T016 [US2] Write failing table-driven tests for `runPicker` entry format: sessions with model/effort show them in correct columns; sessions with empty model/effort display `"unknown"` in both columns; `[dead]` tag still appears for dead sessions in `cli/root_test.go`
- [X] T018 [P] [US2] Write failing tests for `printTable` asserting MODEL and EFFORT columns appear in header and rows; write failing tests for `printJSON` asserting `"model"` and `"effort"` keys are present (empty string, not `"unknown"`, for sessions without values) in `cli/commands_test.go`

### Implementation for US2

- [X] T017 [US2] Update `runPicker` to format each session line as `%-20s %-28s %-12s %-7s` (name, workingDir, model, effort) using `"unknown"` when `Session.Model` or `Session.Effort` is empty; preserve `[dead]` suffix and `strings.TrimRight` trim in `cli/root.go` (makes T016 pass)
- [X] T019 [US2] Update `printTable`: add `MODEL` and `EFFORT` columns to header and row format; update `printJSON`: add `"model"` and `"effort"` keys to the marshalled map (empty string value for pre-existing sessions) in `cli/list.go` (makes T018 pass)

**Checkpoint**: `go test ./...` passes. Picker entries and `cs list` output both include model and effort.

---

## Phase 5: Polish & Verification

**Purpose**: Quality gates required by the constitution before the feature is considered complete.

- [X] T020 Run `go build ./...` — confirm zero errors or warnings
- [X] T021 [P] Run `go vet ./...` — confirm zero output
- [X] T022 Run `go test -race -coverprofile=coverage.out ./...` — confirm zero failures; run `go tool cover -func=coverage.out` and verify ≥80% coverage for every package with business logic (`internal/session`, `internal/tmux`, `cli`)
- [X] T023 [P] Cross-compile `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build ./...` — confirm zero errors
- [X] T024 Manual acceptance test per `specs/003-session-model-effort/quickstart.md`: create session selecting `opus[1m]` + `high`; verify picker displays model/effort; verify `cs list --json` emits `"model":"opus[1m]"` and `"effort":"high"`; verify tmux session option via `tmux show-options -t <name> @cs-model`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 — **blocks all user story phases**
- **Phase 3 (US1+US3)**: Depends on Phase 2 completion — can start as soon as T007 passes
- **Phase 4 (US2)**: Depends on Phase 2 completion — can start in parallel with Phase 3 after T007
- **Phase 5 (Polish)**: Depends on Phase 3 AND Phase 4 both complete

### Within Phase 2

```
T002 → T003 → T004 [P]
                   → T005 [P]
                        → T006 → T007
```

### Within Phase 3

```
T008 [P] → T009 → T010 → T011
T012 [P]          → T013 → T014 → T015
```
(T008/T012 start in parallel; T012 unblocks T013 only after T012 tests are written and confirmed failing)

### Within Phase 4

```
T016 → T017
T018 [P] → T019
```
(T016/T018 can start in parallel)

### Within Phase 5

```
T020 [P] T021
     → T022
     → T023 [P]
T022 passes → T024
```

---

## Parallel Example: Phase 3

```
# Start simultaneously after Phase 2 complete:
Task T008: Write ListSessions tests in internal/tmux/exec_test.go
Task T012: Write createNewSession tests in cli/root_test.go

# After T008 passes (tests confirmed failing):
Task T009: Implement ListSessions format update in internal/tmux/exec.go

# After T009:
Task T010: Write NewSession exec tests in internal/tmux/exec_test.go
# → T011: Implement NewSession exec update

# After T012 tests confirmed failing:
Task T013: Add helpers (knownModels, orderedWithDefault, pickModel, pickEffort) in cli/root.go
# → T014 → T015
```

---

## Implementation Strategy

### MVP (US1 + US3 only — creation flow)

1. Complete Phase 1 (Setup)
2. Complete Phase 2 (Foundational) — required
3. Complete Phase 3 (US1+US3) — creation prompts work
4. **STOP AND VALIDATE**: create a session, confirm model/effort are set in the tmux environment
5. Proceed to Phase 4 (US2) for display

### Incremental Delivery

1. Setup + Foundational → interfaces aligned
2. Phase 3 → model/effort selected and persisted at creation (**functional MVP**)
3. Phase 4 → model/effort visible in picker and `cs list`
4. Phase 5 → constitution quality gates pass

---

## Notes

- `[P]` tasks touch different files; verify no race before running truly in parallel
- Every test task must be run (`go test`) and confirmed **FAIL** before its impl task starts (TDD)
- `sonnet[1m]` / `opus[1m]` brackets are Go string literals — no shell quoting issue in the binary; only shell aliases need quoting (documented in `quickstart.md`)
- `mgr.NewSession` already handles the "session exists → attach" path; model/effort are only set on truly new sessions (the `set-option` calls live inside `execTmuxClient.NewSession`, after `new-session -d` succeeds)
