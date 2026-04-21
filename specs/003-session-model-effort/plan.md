# Implementation Plan: Session Model & Effort Selection

**Branch**: `003-session-model-effort` | **Date**: 2026-04-21 | **Spec**: [spec.md](spec.md)  
**Input**: Feature specification from `specs/003-session-model-effort/spec.md`

## Summary

Extend `cs` session creation with two new fzf prompts (model and effort level) and surface both
values in the session picker and `cs list` output. Model and effort are persisted as tmux session
options (`@cs-model`, `@cs-effort`) — zero overhead at list time, no external file, and graceful
degradation for pre-existing sessions. Claude inherits the chosen values via `-e` flags on
`tmux new-session`.

## Technical Context

**Language/Version**: Go 1.26.2 (go.mod)  
**Primary Dependencies**: `github.com/urfave/cli/v3`, `github.com/stretchr/testify` (assertions only)  
**Storage**: tmux session options (`@cs-model`, `@cs-effort`) — no new files  
**Testing**: `go test`, `testify/assert` + `testify/require`, table-driven  
**Target Platform**: macOS (darwin/arm64) primary; Linux (linux/amd64) cross-compile target  
**Project Type**: CLI binary (incremental feature to existing binary)  
**Performance Goals**: Session creation with two extra fzf prompts stays under 15 seconds total (SC-001)  
**Constraints**: No new external dependencies; no config file; `CGO_ENABLED=0`  
**Scale/Scope**: Same as existing — single-user local tool, up to 50 sessions

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|---|---|---|
| I. Modern Go, Version-Locked | ✅ Pass | go.mod = Go 1.26.2; `/use-modern-go` MUST be invoked before generating any Go code in this feature |
| II. TDD Red-Green-Refactor | ✅ Pass | All test tasks precede corresponding impl tasks in tasks.md; 80% coverage gate maintained |
| III. Interface at the Boundary | ✅ Pass | `FuzzySelector` used for model/effort prompts; no new subprocess calls without interfaces |
| IV. Structured Contextual Logging | ✅ Pass | No new logging paths; existing slog patterns maintained |
| V. Explicit Error Handling | ✅ Pass | `fmt.Errorf("…: %w", err)` wrapping at every boundary; cancelled fzf returns nil |
| VI. Minimal Dependency Footprint | ✅ Pass | Zero new external dependencies; tmux session options need no new library |
| VII. CLI Composability | ✅ Pass | `cs list --json` updated with model/effort keys; `CS_*` env var convention unchanged |

**No violations. Proceeding to design.**

**Post-Phase-1 re-check**: All principles upheld. tmux session options as metadata are a zero-dependency persistence mechanism. No new interfaces required — `FuzzySelector` reused for model/effort prompts.

## Project Structure

### Documentation (this feature)

```text
specs/003-session-model-effort/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── cli-schema.md    # Phase 1 output
└── tasks.md             # Phase 2 output (from /speckit-tasks)
```

### Source Code (files modified by this feature)

```text
internal/
├── session/
│   ├── session.go            # +Model, +Effort fields on Session struct
│   ├── session_test.go       # update String/status tests if affected
│   ├── manager.go            # +model, +effort params on Client interface + Manager.NewSession
│   └── manager_test.go       # update NewSession call sites; add model/effort assertion tests
└── tmux/
    ├── client.go             # +model, +effort params on TmuxClient.NewSession
    ├── exec.go               # NewSession: -e flags + set-option calls; ListSessions: new format string
    ├── exec_test.go          # update/add tests for new exec behaviour
    └── fake.go               # +CreatedModel, +CreatedEffort; update NewSession signature

cli/
├── root.go                   # createNewSession: +selector param, model/effort prompts; runPicker: new entry format
├── root_test.go              # update existing tests; add model/effort selection tests
├── list.go                   # printTable: +MODEL/EFFORT columns; printJSON: +model/effort keys
└── commands_test.go          # update JSON/table test expectations
```

## Implementation Phases

### Phase 1: Session struct & tmux layer

**Goal**: Extend the data layer — `Session` carries model/effort, `TmuxClient` interface accepts them, and the real tmux exec implementation reads/writes them.

**Tasks** (test before impl per TDD):

1. **Test** `internal/session/session.go` — add table-driven tests asserting `Session.Model` and `Session.Effort` fields exist and are accessible.
2. **Impl** `internal/session/session.go` — add `Model string` and `Effort string` fields to `Session`.
3. **Test** `internal/tmux/exec.go` — update/add tests for `ListSessions` parsing 5-part format; add tests for `NewSession` recording model/effort (use `FakeTmuxClient` or tmux integration test stub).
4. **Impl** `internal/tmux/client.go` — update `TmuxClient.NewSession` signature to `(socketPath, name, workingDir, model, effort string)`.
5. **Impl** `internal/tmux/exec.go`:
   - `ListSessions`: change format to `#{session_name}:#{session_path}:#{pane_current_command}:#{@cs-model}:#{@cs-effort}`; parse 5 parts with `strings.SplitN(line, ":", 5)`; populate `session.Model` and `session.Effort`.
   - `NewSession`: add `-e ANTHROPIC_MODEL=<model> -e CLAUDE_CODE_EFFORT_LEVEL=<effort>` to `new-session` command; after `new-session` succeeds, run two `set-option` calls.
6. **Impl** `internal/tmux/fake.go` — add `CreatedModel`, `CreatedEffort` fields; update `NewSession` signature.
7. **Test** `internal/session/manager.go` — update all `NewSession` call sites in tests; add table-driven tests asserting model/effort are passed through.
8. **Impl** `internal/session/manager.go` — update `Client` interface and `Manager.NewSession` to accept and forward `model, effort string`.

### Phase 2: CLI layer — creation prompts

**Goal**: Wire model/effort selection into `createNewSession` and thread the `FuzzySelector` through the call chain.

**Tasks**:

9. **Test** `cli/root_test.go` — add tests for `createNewSession` scenarios:
   - Model/effort selected via fzf → correct values passed to `FakeTmuxClient.NewSession`.
   - Empty fzf response (no `Selections` in fake) → falls back to env var or built-in default.
   - `ANTHROPIC_MODEL` and `CLAUDE_CODE_EFFORT_LEVEL` env vars pre-populate default (first item in list).
   - Update existing `TestRun_NoSessions_CreatesNew` and `TestRun_HasSessions_SelectNew` to supply model/effort selector responses.
10. **Impl** `cli/root.go`:
    - Define `knownModels` and `knownEfforts` slices.
    - Add `orderedWithDefault(list []string, defaultVal string) []string` helper (moves default to front).
    - Add `pickModel(selector fzf.FuzzySelector) (string, error)` and `pickEffort(selector fzf.FuzzySelector) (string, error)` helpers.
    - Update `createNewSession` signature to `(socketPath string, client tmux.TmuxClient, selector fzf.FuzzySelector, stdin io.Reader) error`.
    - Insert `pickModel` then `pickEffort` calls after name confirmation; pass results to `mgr.NewSession`.
    - Update `runWithConfirmReader` and `runPicker` call sites to pass `selector` to `createNewSession`.

### Phase 3: CLI layer — display

**Goal**: Show model/effort in the interactive picker and `cs list`.

**Tasks**:

11. **Test** `cli/root_test.go` — add tests for `runPicker` entry format: sessions with model/effort show them; sessions without show `"unknown"`.
12. **Impl** `cli/root.go` — update `runPicker` entry format to `%-20s %-28s %-12s %-7s` (name, workingDir, model, effort) with `"unknown"` fallback.
13. **Test** `cli/commands_test.go` — update `TestListCommand_WithSessions_JSON` and add table test for human format including model/effort columns.
14. **Impl** `cli/list.go` — update `printTable` header/rows and `printJSON` map to include `"model"` and `"effort"` keys.

### Phase 4: Verification

15. Run `go build ./...` — zero errors.
16. Run `go vet ./...` — zero output.
17. Run `go test -race -coverprofile=coverage.out ./...` — zero failures; check per-package coverage ≥ 80%.
18. Cross-compile: `GOOS=linux GOARCH=amd64 go build ./...` — zero errors.
19. Manual acceptance test: create a session, verify model/effort in picker, verify `cs list --json`, verify tmux session options with `tmux show-options`.

## Model Alias Validation

Both `sonnet[1m]` and `opus[1m]` were verified against the live `claude` binary before task planning:

```
claude --model 'sonnet[1m]' --print "say: ok"  → ok  (exit 0)
claude --model 'opus[1m]'   --print "say: ok"  → ok  (exit 0)
```

**Zsh shell-globbing caveat**: In zsh, bare `sonnet[1m]` is interpreted as a glob pattern and
expanded by the shell before reaching the process, causing a "no matches found" error. The brackets
must be quoted (`'sonnet[1m]'` or `"sonnet[1m]"`).

**No impact on Go implementation**: The Go `exec.Command` invocation passes args as a `[]string`
— no shell is involved, so the brackets pass through verbatim. The `-e ANTHROPIC_MODEL=sonnet[1m]`
flag on `tmux new-session` is equally unaffected.

**Shell-alias documentation**: The quickstart must note that users setting `ANTHROPIC_MODEL` in
their shell profile or alias must quote the value:

```zsh
claude-opus1m() { ANTHROPIC_MODEL='opus[1m]' CLAUDE_CODE_EFFORT_LEVEL=xhigh cs "$@"; }
```

## Complexity Tracking

No constitution violations — no complexity justification required.
