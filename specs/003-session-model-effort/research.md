# Research: Session Model & Effort Selection

**Branch**: `003-session-model-effort` | **Date**: 2026-04-21

## Decision 1: Metadata Persistence Strategy

**Decision**: Store model and effort as tmux session options (`@cs-model`, `@cs-effort`) using `tmux set-option -t <session> @cs-model <value>`.

**Rationale**:
- Session options are queryable in a single `list-sessions` call via format tokens `#{@cs-model}` and `#{@cs-effort}`, adding zero per-session overhead compared to separate `show-environment` calls.
- Eliminates a separate metadata file and the synchronisation problem of keeping it consistent with actual tmux session lifetime.
- Survives process restarts: tmux holds the option for the lifetime of the session without any cs process running.
- Naturally returns empty string for sessions created before this feature — no migration step.

**Alternatives considered**:
- `~/.cs/metadata.json` keyed by session name: simple, but requires sync on kill/die and a separate I/O path.
- `tmux set-environment -t <session> ANTHROPIC_MODEL <value>`: environment variables are visible to child processes but NOT accessible via `list-sessions` format strings, requiring N extra `show-environment` calls (one per session per variable).

## Decision 2: Default Pre-Selection in fzf

**Decision**: Reorder the items slice so the env-var default is always the first element. With `--no-sort` already in use, fzf pre-positions the cursor on the first item.

**Rationale**:
- Requires no changes to the `FuzzySelector` interface — a stable interface is preferable to avoid cascading fake/test updates.
- Works with the existing `FakeFuzzySelector` (returns first `Selections` element or empty string when empty — treated as "use default").
- Consistent with how fzf behaves: cursor starts on item 1 with `--no-sort`, so pressing Enter immediately takes the default.

**Alternatives considered**:
- `--query <default>` fzf flag: pre-filters the list to the default string, which is visually clean but requires extending `FuzzySelector.Select` to accept a query parameter — more churn for a minor UX gain.
- `--default-accept` fzf flag: not universally available across fzf versions.

## Decision 3: Interface Change for NewSession

**Decision**: Add `model, effort string` as positional parameters to `TmuxClient.NewSession`, `session.Client.NewSession`, and `session.Manager.NewSession`.

**Rationale**:
- Consistent with the existing `(socketPath, name, workingDir string)` style already in the codebase.
- Minimal churn — affects only 3 files and their test doubles.
- No new types required for a 2-field extension.

**Alternatives considered**:
- `SessionConfig` struct: cleaner for 4+ fields, but introduces an extra type just to carry two strings that will never be set independently.

## Decision 4: Passing FuzzySelector to createNewSession

**Decision**: Add `selector fzf.FuzzySelector` as a parameter to `createNewSession` and update both call sites (`runWithConfirmReader` direct path and `runPicker` new-session branch).

**Rationale**:
- `createNewSession` already receives `client` and `stdin` from its callers; adding `selector` follows the same injection pattern.
- The selector is already in scope at both call sites (it is a parameter of `runWithConfirmReader`).
- No global state required.

## Decision 5: Known Model and Effort Lists

**Decision**: Hardcode a single ordered slice of known model aliases and effort levels. Model default is `os.Getenv("ANTHROPIC_MODEL")` falling back to `"sonnet"`. Effort default is `os.Getenv("CLAUDE_CODE_EFFORT_LEVEL")` falling back to `"medium"`.

**Known models (in display order)**:
```
sonnet, opus, haiku, sonnet[1m], opus[1m]
```

**Known effort levels (in display order)**:
```
low, medium, high, xhigh
```

**Rationale**:
- The model set is small and stable enough for v1; the list can be updated in one place (`cli/root.go`) without touching any other file.
- Dynamic API fetching adds a network dependency and latency to every session creation — out of scope per spec Scope Constraints.
- `max` is excluded: it is session-scoped only and not persistable, matching the constitution's "No config file in v1" constraint.

## Decision 6: Graceful Degradation for Empty FuzzySelector Response

**Decision**: If `selector.Select()` returns `("", nil)` during model or effort selection (impossible in real fzf with a non-empty list, but can happen in tests with an empty `FakeFuzzySelector`), treat it as "use the default" (env var or built-in).

**Rationale**: Keeps existing tests passing without modification — they supply no model/effort selections and the default behaviour is to fall through to env var resolution.
