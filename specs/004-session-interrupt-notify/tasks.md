# Tasks: Interrupt-Driven Session Notifications

**Input**: Design documents from `specs/004-session-interrupt-notify/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅

**Tests**: Per constitution Principle II (TDD), test tasks MUST precede every implementation task
for any logic with behaviour. Tests are written first and verified to fail before implementation.

**Organization**: Tasks are grouped by user story to enable independent implementation and
testing of each story increment.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no blocking dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1–US4 from spec.md)

---

## Phase 1: Setup

**Purpose**: Verify the Go version and applicable stdlib patterns before writing any new Go code.
The constitution (Principle I) gates all implementation on `/use-modern-go` being invoked.

- [x] - [x] T001 Invoke `/use-modern-go` skill in the current session to verify Go 1.26.2 stdlib
  patterns (range-over-int, `cmp.Or`, `slices.*`, `sync/atomic` typed vars) before any Go
  code is written — this is a constitution gate, not optional

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core interface extensions that multiple user stories depend on. Must complete before
US3 can implement session-level monitor-silence, and before US4 can use the tmux client.

**⚠️ CRITICAL**: US3 cannot start until T002–T005 are complete.

- [x] - [x] T002 Write failing table-driven tests for `SetWindowOption` (assert tmux command format
  `"set-option -t <session> <option> <value>"` via the fake runner) in
  `internal/tmux/exec_unit_test.go`
- [x] - [x] T003 Add `SetWindowOption(ctx context.Context, session, option, value string) error` to the
  `TmuxClient` interface in `internal/tmux/client.go`
- [x] - [x] T004 [P] Implement `SetWindowOption` on `ExecClient` in `internal/tmux/exec.go` — shells
  out to `tmux -S <socket> set-option -t <session> <option> <value>`
- [x] - [x] T005 [P] Add `SetWindowOption` no-op stub and call recorder to `FakeClient` in
  `internal/tmux/fake.go` so existing tests continue to compile and new tests can assert calls

**Checkpoint**: `go test -race ./internal/tmux/...` passes. `SetWindowOption` is usable by US3.

---

## Phase 3: User Story 1 — Ghostty-Native Notification Appearance (Priority: P1) 🎯 MVP

**Goal**: Deliver `cli/assets/notify.sh` — the Ghostty-native notify script embedded in the
binary. On invocation with a Claude Code hook JSON payload, it dispatches a macOS notification
that visually matches Ghostty's own alerts (correct icon, correct sender identity,
replace-not-stack grouping, click-to-jump to the correct tmux pane).

**Independent Test**: Install the script manually. Run:

```bash
echo '{"hook_event_name":"Notification","session_id":"abc12345-test","cwd":"/Users/dev/myproject/src","message":"Can I run: rm -rf build/?","notification_type":"permission_prompt"}' \
  | ~/.local/share/cs/notify.sh
```

Verify: notification appears with Ghostty icon and sender, title is "Claude: myproject/src",
clicking focuses Ghostty and activates the correct pane.

### Tests for User Story 1 ⚠️ MANDATORY

- [x] - [x] T006 [P] [US1] Write integration test (`//go:build integration`) in
  `cli/notify_test.go` — places a mock `terminal-notifier` binary on PATH that records its
  arguments to a temp file; asserts: `-sender com.mitchellh.ghostty` present, `-group` uses
  truncated session ID, TMUX_TARGET containing `'` is rejected (no `-execute` arg), invocation
  with `terminal-notifier` absent exits 0 (graceful degradation), log file created with mode
  0600

### Implementation for User Story 1

- [x] - [x] T007 [US1] Create `cli/assets/notify.sh` implementing all plan.md Phase 0 requirements:
  read stdin via `jq -r` (no raw `eval`/interpolation), validate `TMUX_TARGET` against
  `[A-Za-z0-9_\-.:]+` allowlist (drop `-execute` if invalid), resolve pane via
  `tmux -S "${CS_TMUX_SOCKET:-~/.local/share/cs/cs.sock} list-panes -a"`, dispatch via
  `terminal-notifier -sender com.mitchellh.ghostty`, set `-group "cs-${SESSION_ID:0:8}"`,
  set `-execute` only when TMUX_TARGET passes validation, create log at `~/.cs/notification.log`
  with mode 0600 on first write (`install -m 0600`), strip ANSI escape sequences in log lines,
  exit 0 on `Stop` hook (no notification, no file write), exit 0 gracefully when
  `terminal-notifier` is absent
- [x] - [x] T008 [US1] Add `//go:embed assets/notify.sh` declaration and `var notifyShScript []byte`
  to `cli/setup.go` (or a new `cli/assets.go` if keeping setup.go clean)

**Checkpoint**: T006 integration test passes. Manual smoke test produces a Ghostty-icon
notification. `go test -race ./cli/...` passes (non-integration tests unaffected).

---

## Phase 4: User Story 2 — Alert When Claude Needs Input (Priority: P1)

**Goal**: Implement the Go hook-registration functions (`ReadHookSettings`, `MergeHooks`,
`RemoveHooks`, `WriteHookSettings`) in `internal/setup/hooks.go`. These wire Claude Code's
lifecycle events to the notify script and are the plumbing that makes US1 fire automatically
from real Claude sessions (used by US4's setup step).

**Independent Test**: Call `MergeHooks` + `WriteHookSettings` directly (no UI), then verify
`~/.claude/settings.json` contains the expected hook entries. Invoke notify.sh with a simulated
Notification payload and confirm it fires within 5 seconds.

### Tests for User Story 2 ⚠️ MANDATORY

- [x] - [x] T009 [P] [US2] Write table-driven tests for `ReadHookSettings` (missing file returns empty
  map; malformed JSON returns error; valid file round-trips), `MergeHooks` (additive; idempotent
  on duplicate command; rejects non-absolute Command path), `RemoveHooks` (strips matching
  entries; removes empty arrays; removes empty hooks key), and `WriteHookSettings` (atomic:
  temp+rename; existing file replaced; concurrent write does not corrupt) in
  `internal/setup/hooks_test.go`

### Implementation for User Story 2

- [x] - [x] T010 [US2] Define `HookEntry`, `HookGroup`, `HookDef` structs with `json` tags in
  `internal/setup/hooks.go`
- [x] - [x] T011 [US2] Implement `ReadHookSettings(path string) (map[string]any, error)` in
  `internal/setup/hooks.go` — returns empty map (not error) when file is absent; returns error
  on malformed JSON
- [x] - [x] T012 [P] [US2] Implement `MergeHooks(existing map[string]any, entries []HookEntry)
  (map[string]any, bool)` in `internal/setup/hooks.go` — additive, idempotent (no duplicate
  Command entries per event type), validates `HookEntry.Command` is an absolute path before
  accepting
- [x] - [x] T013 [P] [US2] Implement `RemoveHooks(existing map[string]any, scriptPath string)
  (map[string]any, bool)` in `internal/setup/hooks.go` — strips entries whose `command` matches
  `scriptPath`; prunes empty arrays and empty `hooks` key
- [x] - [x] T014 [US2] Implement `WriteHookSettings(path string, settings map[string]any) error` in
  `internal/setup/hooks.go` — atomic write via `os.CreateTemp` in same directory as `path` +
  `os.Rename`; preserves all non-hooks keys verbatim

**Checkpoint**: `go test -race ./internal/setup/...` passes. `MergeHooks` + `WriteHookSettings`
can be called from a test harness to install hooks manually and verify the notify script fires.

---

## Phase 5: User Story 3 — Stall Detection for Idle Sessions (Priority: P2)

**Goal**: Detect sessions that have been silent longer than `CS_STALL_THRESHOLD` (default 180s)
using tmux's built-in `monitor-silence` + `alert-silence` hook — no daemon, no polling. Each
`cs`-created session automatically gets `monitor-silence` set. When silence fires, notify.sh is
invoked in stall mode and dispatches a "Session idle for Nm" notification.

**Independent Test**:

```bash
export CS_STALL_THRESHOLD=30
cs new stall-test
# leave the session idle for 30+ seconds
# verify stall notification fires via ~/.cs/notification.log
```

### Tests for User Story 3 ⚠️ MANDATORY

- [x] - [x] T015 [P] [US3] Write integration test (`//go:build integration`) for notify.sh stall mode
  in `cli/notify_test.go` — invoke `notify.sh stall` with `TMUX_SESSION=test-session` and
  `TMUX_WINDOW=0` set; mock `tmux display-message` via PATH override; verify notification
  dispatched with message containing "idle", JSON constructed via `jq -n --arg` (not string
  interpolation, validated by checking log output)
- [x] - [x] T016 [P] [US3] Write unit test in `internal/session/manager_test.go` verifying that
  `NewSession` calls `SetWindowOption` with option `"monitor-silence"` and the value from
  `CS_STALL_THRESHOLD` (default `"180"`) immediately after session creation

### Implementation for User Story 3

- [x] - [x] T017 [US3] Add stall invocation mode to `cli/assets/notify.sh` — detect `[[ "$1" ==
  "stall" ]]`; read `$TMUX_SESSION`, `$TMUX_WINDOW` from environment; call
  `tmux display-message -p '#{pane_current_path}'` for CWD and
  `tmux display-message -p '#{window_silence_interval}'` for idle seconds; construct JSON
  exclusively via `jq -n --arg id "$TMUX_SESSION" --arg cwd "$CWD" --arg msg "..."`;
  dispatch notification with idle duration in message
- [x] - [x] T018 [US3] Call `client.SetWindowOption(ctx, name, "monitor-silence", threshold)` in
  `internal/session/manager.go` after session creation — read threshold via
  `cmp.Or(os.Getenv("CS_STALL_THRESHOLD"), "180")`
- [x] - [x] T019 [US3] Add `allow-passthrough on` and `set-hook -g alert-silence` entries to
  `cli/assets/tmux.conf` inside `# --- cs begin ---` / `# --- cs end ---` markers (consistent
  with reconciliation contract in plan.md)

**Checkpoint**: `go test -race ./...` passes. Creating a cs session and waiting 30s (with
reduced threshold) produces a stall notification logged to `~/.cs/notification.log`.

---

## Phase 6: User Story 4 — Configure Notifications and Ghostty via cs setup (Priority: P3)

**Goal**: Expand `cs setup` with: (1) a redesigned tmux.conf step that diffs, backs up, and
reconciles existing configs; (2) a Ghostty recommendation step; (3) a notification opt-in step
that installs the script, registers hooks, and fires a live test notification. All steps are
idempotent on re-run.

**Independent Test**: Run `cs setup` in a clean environment (no Ghostty, no notifications
installed). Verify all prompts appear correctly, opting in installs the script and hooks, the
test notification fires with Ghostty icon, and re-running shows the update/remove menu.

### Tests for User Story 4 ⚠️ MANDATORY

- [x] - [x] T020 [P] [US4] Write table-driven tests for `GhosttyInstalled()` (true when app bundle
  present; true when `ghostty` on PATH; false otherwise) and `TerminalNotifierInstalled()` (true
  when `terminal-notifier` on PATH; false otherwise) in `internal/setup/checker_test.go`
- [x] - [x] T021 [P] [US4] Write table-driven tests for `TmuxConfStatus` (Absent / Identical /
  Differs), `AppendCsBlock` (no markers: appends; markers present: replaces inner content;
  atomic write), `RemoveCsBlock` (strips block; no-op if absent), and `UnifiedDiff` (truncates
  to `maxLines`; returns empty string when no diff) in `internal/setup/checker_test.go`
- [x] - [x] T022 [P] [US4] Write tests for the redesigned tmux.conf setup step in `cli/setup_test.go`
  — Case A (file absent → install offered); Case B (file identical → "up to date" printed,
  no prompt); Case C (file differs → diff shown, [a]/[r]/[s] prompt, [r] creates `.bk` backup)
- [x] - [x] T023 [P] [US4] Write tests for Ghostty recommendation step in `cli/setup_test.go` —
  GhosttyInstalled true: prints "Ghostty detected ✓"; false: prints install command and waits
  for Enter before continuing
- [x] - [x] T024 [P] [US4] Write tests for notification opt-in step in `cli/setup_test.go` — first
  run: prompts [Y/n], on Y: checks terminal-notifier, writes script to
  `~/.local/share/cs/notify.sh` (chmod 0700), calls MergeHooks, fires test notification; re-run:
  shows [t]/[u]/[r]/[s] menu; [r]: calls RemoveHooks + RemoveCsBlock + removes script file

### Implementation for User Story 4

- [x] - [x] T025 [P] [US4] Implement `GhosttyInstalled() bool` in `internal/setup/checker.go` —
  checks for `/Applications/Ghostty.app` via `os.Stat` OR `ghostty` in PATH via
  `exec.LookPath`; returns true if either found
- [x] - [x] T026 [P] [US4] Implement `TerminalNotifierInstalled() bool` in `internal/setup/checker.go`
  — `exec.LookPath("terminal-notifier") == nil`
- [x] - [x] T027 [US4] Implement `TmuxConfStatus(embedded []byte) (TmuxConfState, error)`,
  `AppendCsBlock(path string, block []byte) error`, `RemoveCsBlock(path string) error`, and
  `UnifiedDiff(existing, incoming []byte, maxLines int) string` in `internal/setup/checker.go`;
  `AppendCsBlock` and `RemoveCsBlock` use atomic write (temp+rename); markers are
  `# --- cs begin ---` and `# --- cs end ---`
- [x] - [x] T028 [US4] Redesign tmux.conf installation step in `cli/setup.go` — use `TmuxConfStatus`
  to branch: Absent → offer full install; Identical → print "tmux config up to date ✓";
  Differs → print `UnifiedDiff` (max 40 lines) and prompt `[a]ppend / [r]eplace / [s]kip`;
  `[r]` copies existing file to `~/.tmux.conf.bk` before overwriting
- [x] - [x] T029 [US4] Add Ghostty recommendation step to `cli/setup.go` after tmux.conf step —
  call `GhosttyInstalled()`; if true print "Ghostty detected ✓"; if false print
  `brew install --cask ghostty` and prompt Enter to continue
- [x] - [x] T030 [US4] Add notification opt-in step to `cli/setup.go` after Ghostty step — prompt
  `[Y/n]`; on accept: call `TerminalNotifierInstalled()` (offer `brew install terminal-notifier`
  if absent), write `notifyShScript` to `~/.local/share/cs/notify.sh` (mode 0700), call
  `MergeHooks`+`WriteHookSettings` on `~/.claude/settings.json`, fire test notification via
  `jq -n` payload piped to the installed script; on re-run: detect existing script and show
  `[t]est / [u]pdate / [r]emove / [s]kip`; `[r]` calls `RemoveHooks`+`RemoveCsBlock`+
  `os.Remove`
- [x] - [x] T031 [US4] Update setup summary step in `cli/setup.go` to include notification system
  status: "Notifications: installed ✓ / not installed" with tip to run `cs setup` if not
  installed

**Checkpoint**: `go test -race ./...` passes. `cs setup` end-to-end produces Ghostty-native test
notification. Re-running is idempotent. Removal cleans up all artifacts.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation updates and end-to-end validation.

- [x] - [x] T032 [P] Update `CLAUDE.md` architecture table to document the `notify.sh` embedded asset
  and the new `internal/setup/hooks.go` functions
- [x] - [x] T033 [P] Run all `quickstart.md` validation scenarios end-to-end: setup flow, human-in-
  the-loop notification, stall detection (30s threshold), replace-not-stack, and removal
- [x] - [x] T034 Run `make lint` and fix any `golangci-lint` violations introduced by new code
- [x] - [x] T035 Verify binary cross-compiles for `linux/amd64` and `darwin/arm64` per constitution
  Quality Gate 7: `GOOS=linux GOARCH=amd64 go build ./...` and
  `GOOS=darwin GOARCH=arm64 go build ./...`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1. Blocks Phase 5 (US3 needs `SetWindowOption`)
- **Phase 3 (US1)**: Depends on Phase 1 only. Can start in parallel with Phase 2
- **Phase 4 (US2)**: Depends on Phase 1 only. Can start in parallel with Phases 2 and 3
- **Phase 5 (US3)**: Depends on Phase 2 (needs `SetWindowOption`) and Phase 3 (extends
  notify.sh)
- **Phase 6 (US4)**: Depends on Phases 3, 4, 5 (integrates all three into setup UI)
- **Phase 7 (Polish)**: Depends on Phase 6 completion

### User Story Dependencies

- **US1 (P1)**: Independent after Phase 1 — can start immediately
- **US2 (P1)**: Independent after Phase 1 — can run in parallel with US1
- **US3 (P2)**: Depends on Phase 2 (foundational) and US1 (extends notify.sh)
- **US4 (P3)**: Depends on US1 + US2 + US3 (integrates all into setup UI)

### Within Each User Story

- Tests MUST be written first and verified to fail before implementation tasks start
- Within US2: types (T010) → read (T011) → merge/remove (T012, T013) → write (T014)
- Within US4: checkers (T025–T027) → setup steps (T028–T031)

### Parallel Opportunities

- T004 and T005 (Foundational): different files, run in parallel
- T006 (US1 test) can be written in parallel with T009 (US2 test)
- T012 and T013 (MergeHooks, RemoveHooks): different functions, run in parallel
- T020, T021, T022, T023, T024 (US4 tests): all test files, run in parallel
- T025, T026 (US4 checker impls): independent functions, run in parallel
- T032 and T033 (Polish): independent, run in parallel

---

## Parallel Example: User Story 1 + User Story 2

```text
# After Phase 1 completes, these can run simultaneously:

Thread A (US1):
  T006 → write notify_test.go integration test
  T007 → create cli/assets/notify.sh
  T008 → add embed declaration

Thread B (US2):
  T009 → write hooks_test.go
  T010 → define types
  T011 → ReadHookSettings
  T012+T013 → MergeHooks + RemoveHooks (parallel)
  T014 → WriteHookSettings
```

---

## Implementation Strategy

### MVP (User Stories 1 + 2 Only)

1. Phase 1: Setup — invoke `/use-modern-go`
2. Phase 3: US1 — write and embed notify.sh
3. Phase 4: US2 — hook registration Go functions
4. **STOP and VALIDATE**: manually call `WriteHookSettings`, trigger a Claude event, confirm
   Ghostty notification appears within 5 seconds
5. Ship: developers get Ghostty-native notifications from real Claude sessions

### Incremental Delivery

1. Phase 1 + 3 + 4 → MVP: event-based Ghostty notifications
2. Phase 2 + 5 → Add stall detection: silent sessions get alerted
3. Phase 6 → Add `cs setup` UI: zero-config onboarding
4. Phase 7 → Polish: docs, lint, cross-compile gate

### Single-Developer Execution Order (Recommended)

```text
T001 → T002 → T003 → T004+T005 (parallel)
     → T006+T009 (parallel, write tests first)
     → T007 → T008 (US1 impl)
     → T010 → T011 → T012+T013 (parallel) → T014 (US2 impl)
     → T015+T016 (parallel, US3 tests)
     → T017 → T018 → T019 (US3 impl)
     → T020+T021+T022+T023+T024 (parallel, US4 tests)
     → T025+T026 (parallel) → T027 → T028 → T029 → T030 → T031 (US4 impl)
     → T032+T033 (parallel) → T034 → T035 (polish)
```

---

## Notes

- `[P]` tasks operate on different files with no blocking inter-dependencies — safe to parallelise
- `[Story]` label maps each task to its user story for traceability
- Each user story is independently completable and testable before the next begins
- All test tasks must be written first and confirmed failing before implementation starts
- Constitution Quality Gates (go build, go vet, golangci-lint, go test -race, ≥80% coverage,
  doc comments, cross-compile) must pass before any user story phase is considered complete
