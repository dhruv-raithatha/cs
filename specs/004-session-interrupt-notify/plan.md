# Implementation Plan: Interrupt-Driven Session Notifications

**Branch**: `004-session-interrupt-notify` | **Date**: 2026-05-11 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/004-session-interrupt-notify/spec.md`

## Summary

Add Ghostty-native, interrupt-driven notifications to `cs`: a revised notify script delivered with
the binary (embedded asset) that uses `terminal-notifier -sender com.mitchellh.ghostty` for visual
fidelity with Ghostty; a stall detection mechanism via tmux's built-in `monitor-silence` + `alert-silence`
hook wired into each `cs`-created session; and a multi-step expansion of `cs setup` that detects Ghostty,
checks for `terminal-notifier`, installs the notify script, registers Claude Code hooks in
`~/.claude/settings.json`, and fires a live test notification.

## Technical Context

**Language/Version**: Go 1.26.2
**Primary Dependencies**: `github.com/urfave/cli/v3 v3.8.0`, `github.com/stretchr/testify v1.11.1` (no new deps required)
**Storage**: `~/.claude/settings.json` (hook registration), `~/.local/share/cs/` (notify script install path)
**Testing**: `go test -race ./...`, table-driven tests, `//go:build integration` for subprocess tests
**Target Platform**: macOS (darwin/arm64, darwin/amd64), Ghostty as recommended terminal
**Project Type**: CLI binary
**Performance Goals**: Notification delivery < 5 seconds end-to-end; stall detection fires within
30 seconds of threshold crossing
**Constraints**: No new Go dependencies; no persistent daemon; no elevated privileges
**Scale/Scope**: Single-developer local tooling; sessions bounded by tmux socket capacity

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Modern Go, Version-Locked | ✅ PASS | Go 1.26.2; `/use-modern-go` must be invoked before writing any Go code |
| II. TDD, Red-Green-Refactor | ✅ PASS | Test files required before each implementation task |
| III. Interface at the Boundary | ✅ PASS | `HookRegistrar`, `ScriptInstaller` interfaces defined; no inline concrete calls in logic |
| IV. Structured Contextual Logging | ✅ PASS | All new Go functions accept `*slog.Logger` |
| V. Explicit Error Handling | ✅ PASS | `fmt.Errorf("...: %w", err)` at all layer boundaries; no silent discards |
| VI. Minimal Dependency Footprint | ✅ PASS | No new Go deps; `terminal-notifier` and `jq` are user-installed shell-side tools |
| VII. Unix Composability | ✅ PASS | New `cs setup` steps remain interactive-only |

*Post-design re-check: No violations found after Phase 1 design — see data-model.md.*

## Security Analysis

> Performed on the proposed design using the existing dotfiles `notify.sh` as the reference
> implementation. All issues are addressed in the implementation requirements below.

### Critical: Shell injection via TMUX_TARGET in -execute command

**Location**: Existing notify.sh line 69; proposed new script replicates the same pattern.

**Issue**: The `-execute` argument passed to `terminal-notifier` is a shell command string built
by interpolating `$TMUX_TARGET` (which comes from `tmux list-panes`) inside single quotes:

```bash
EXECUTE_CMD="tmux select-window -t '${TMUX_TARGET%.*}' && ..."
```

A tmux session name containing a single quote (e.g. `foo'bar`) breaks the quoting and allows
arbitrary shell command injection that executes when the user clicks the notification.

**Fix**: Sanitize `TMUX_TARGET` immediately after extraction. Allow only `[A-Za-z0-9_\-.:]+`.
Reject (drop click action, still show notification) if the value contains any other character.
Use a dedicated validation function tested independently.

### Critical: JSON injection in stall event synthesis

**Location**: Phase 1 stall invocation — the script must synthesise JSON from tmux variables.

**Issue**: Manual string interpolation of tmux values (session name, CWD) into a JSON string
breaks JSON validity and enables injection if either value contains `"` or `\`:

```bash
# Unsafe
printf '{"session_id":"%s","cwd":"%s"}' "$SESSION_NAME" "$CWD"
```

**Fix**: Use `jq -n --arg` to construct JSON safely — `jq` handles escaping correctly:

```bash
jq -n --arg id "$SESSION_NAME" --arg cwd "$CWD" --arg msg "Session idle for ${IDLE_MINS}m" \
  '{"notification_type":"stall","session_id":$id,"cwd":$cwd,"message":$msg}'
```

### Medium: Log file permissions

**Location**: `~/.cs/notification.log` created with `>>` (inherits umask, typically 0644).

**Issue**: The log contains session paths and message previews. On a shared machine this leaks
working directory and context to other users.

**Fix**: On first write, explicitly create the file with mode 0600:

```bash
install -m 0600 /dev/null ~/.cs/notification.log 2>/dev/null || true
```

(safe — `install` is a no-op if the file already exists with correct permissions.)

### Low: ANSI/terminal control sequences in message content

**Location**: `$MSG` extracted from Claude's output and passed to `-message`.

**Issue**: Claude may include ANSI colour codes or OSC sequences in its output. `terminal-notifier`
strips most of these, but they could appear in logs.

**Fix**: Strip ANSI escape sequences before logging or display using `sed 's/\x1b\[[0-9;]*m//g'`.
Apply only to the log line, not to the notification message (terminal-notifier handles that).

### Out of scope / accepted risks

- **tmux socket permissions** (`~/.local/share/cs/cs.sock`): Any local user with filesystem access
  could issue commands to the cs tmux server. This is an existing constraint of the cs design and
  is unchanged by this feature. Mitigation: `cs.sock` is in a user-owned directory (`~/.local/`),
  so standard macOS DAC prevents other users from reaching it on a single-user machine.
- **Brew install execution**: Package name `terminal-notifier` is hardcoded, not user-supplied.
  No injection risk.

---

## Project Structure

### Documentation (this feature)

```text
specs/004-session-interrupt-notify/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   ├── notify-script-stdin.md   # JSON payload contract for notify script
│   └── settings-hooks.md        # ~/.claude/settings.json hook config contract
└── tasks.md             # Phase 2 output (/speckit.tasks — not created here)
```

### Source Code

```text
cli/
├── setup.go                     # MODIFIED: add 3 new setup steps
├── setup_test.go                # MODIFIED: tests for new steps
└── assets/
    ├── tmux.conf                # MODIFIED: add allow-passthrough, alert-silence hook
    └── notify.sh                # NEW: embedded notify script (replaces dotfiles version)

internal/
├── setup/
│   ├── checker.go               # MODIFIED: GhosttyInstalled(), TerminalNotifierInstalled()
│   ├── checker_test.go          # MODIFIED: tests for new checks
│   ├── hooks.go                 # NEW: ReadHookSettings(), WriteHookSettings(), MergeHooks()
│   └── hooks_test.go            # NEW
└── session/
    └── manager.go               # MODIFIED: set monitor-silence on new session window

internal/tmux/
    exec.go                      # MODIFIED: SetWindowOption() for monitor-silence
    fake.go                      # MODIFIED: stub SetWindowOption()
```

---

## Implementation Phases

### Phase 0: Notify Script (Foundation)

**Goal**: Deliver the Ghostty-native notify script as an embedded binary asset, superseding the
dotfiles version.

**File**: `cli/assets/notify.sh`

The script must:

1. Accept JSON on stdin (Claude Code hook payload: `session_id`, `cwd`, `message`, `hook_event_name`,
   `notification_type`). Parse all fields using `jq -r`; never interpolate raw JSON strings via `eval`.
2. Accept a stall invocation via positional arg `stall` — context comes from tmux `run-shell`
   environment variables; JSON is synthesised using `jq -n --arg` (see Security Analysis).
3. **Stop hook**: exit 0 with no notification and no file write. The token-usage-to-file behaviour
   from the dotfiles version is out of scope for this feature; if a statusline integration is
   wanted, that is a separate feature with its own spec.
4. Resolve the tmux pane by walking the PID tree to find the claude process TTY, then matching
   against `tmux -S ~/.local/share/cs/cs.sock list-panes -a`.
5. Sanitize `TMUX_TARGET` immediately after extraction: allow only `[A-Za-z0-9_\-.:]+`; if the
   value fails validation, set `EXECUTE_CMD=""` (notification still fires, just without click-to-jump).
6. Dispatch via `terminal-notifier -sender com.mitchellh.ghostty`.
7. Set `-group "cs-${SESSION_ID:0:8}"` for replace-not-stack behaviour.
8. Set `-execute` only when `EXECUTE_CMD` is non-empty and `TMUX_TARGET` passed validation.
9. Create `~/.cs/notification.log` with mode 0600 on first write; strip ANSI sequences in log lines.
10. Gracefully degrade when `terminal-notifier` is absent: log warning, exit 0 (never crash the hook).

**Key delta from existing dotfiles notify.sh:**

| Aspect | Before | After |
| --- | --- | --- |
| Sender | `com.apple.Terminal` | `com.mitchellh.ghostty` |
| tmux socket | bare `tmux list-panes` | `-S ~/.local/share/cs/cs.sock` |
| Stop hook | writes to `~/.cache/claude-context.txt` | exits 0, no file write |
| TMUX_TARGET validation | none | allowlist `[A-Za-z0-9_\-.:]+` |
| Stall JSON construction | N/A | `jq -n --arg` |
| Log permissions | inherits umask | 0600 explicitly |
| Install location | ad-hoc dotfiles | `~/.local/share/cs/notify.sh` |

---

### Phase 1: Stall Detection via tmux

**Goal**: Detect silent sessions without a daemon, using tmux's built-in `monitor-silence`.

**tmux.conf additions** (`cli/assets/tmux.conf`):

```tmux
# Ghostty passthrough: allow OSC sequences to reach the host terminal
set -g allow-passthrough on

# Stall detection: fires when a window has been silent for monitor-silence seconds
set-hook -g alert-silence 'run-shell -b "~/.local/share/cs/notify.sh stall"'
```

**Session creation** (`internal/session/manager.go`, `internal/tmux/exec.go`):

When `NewSession` creates a window, set the window's `monitor-silence` option:

```go
threshold := cmp.Or(os.Getenv("CS_STALL_THRESHOLD"), "180")
client.SetWindowOption(ctx, name, "monitor-silence", threshold)
```

`SetWindowOption` shells out to `tmux -S <socket> set-option -t <session> monitor-silence <value>`.

**Stall invocation contract**: tmux fires `alert-silence`, calling `notify.sh stall`. The script
reads `$TMUX_SESSION`, `$TMUX_WINDOW` from the run-shell environment, computes idle minutes, and
synthesises a JSON payload using `jq -n --arg`.

---

### Phase 2: cs setup Expansion

**Goal**: Add two new steps after the tmux.conf step, update the tmux.conf step itself.

**Step order** (final):

1. Deps check (existing)
2. PATH setup (existing)
3. tmux.conf reconciliation (existing step, **redesigned** — see below)
4. **[NEW]** Ghostty recommendation
5. **[NEW]** Notification system install
6. Summary (existing, updated)

#### Redesigned Step 3 — tmux.conf Reconciliation

The current step only offers to copy when `~/.tmux.conf` is absent. Users with an existing
config would silently skip the new cs-specific additions (`allow-passthrough`, `alert-silence`
hook). The new behaviour:

**Case A — no existing `~/.tmux.conf`:** Offer to install the embedded config wholesale (unchanged behaviour).

**Case B — existing config, identical to embedded:** Print "tmux config up to date ✓", proceed.

**Case C — existing config, differs from embedded:**

```text
Your ~/.tmux.conf has local changes. Here's what cs would add:

--- ~/.tmux.conf
+++ embedded

[unified diff, highlighted, maximum 40 lines shown]

How would you like to proceed?
  [a] Append cs additions to your existing config (recommended)
  [r] Replace — back up your config to ~/.tmux.conf.bk, then install
  [s] Skip

>
```

On `[a]` (append):

- Check whether `# --- cs begin ---` markers already exist in `~/.tmux.conf`
- If yes: replace the content between existing markers (idempotent update)
- If no: append the following block to the end of the existing file:

```tmux
# --- cs begin ---
# Added by cs setup. Safe to remove if cs is uninstalled.
set -g allow-passthrough on
set-hook -g alert-silence 'run-shell -b "~/.local/share/cs/notify.sh stall"'
# --- cs end ---
```

On `[r]` (replace):

- Copy `~/.tmux.conf` → `~/.tmux.conf.bk` (inform user of backup location)
- Write embedded config to `~/.tmux.conf`

On `[s]` (skip): proceed without changes.

**cs setup remove path**: strip the `# --- cs begin ---` … `# --- cs end ---` block from
`~/.tmux.conf` if present (no other lines touched).

**Go helpers needed** (`internal/setup/checker.go`):

```go
// TmuxConfStatus returns whether the existing tmux.conf matches, differs, or is absent.
func TmuxConfStatus(embedded []byte) (TmuxConfState, error)

// AppendCsBlock appends or replaces the cs-managed block in path.
func AppendCsBlock(path string, block []byte) error

// RemoveCsBlock strips the cs-managed block from path.
func RemoveCsBlock(path string) error

// UnifiedDiff returns a human-readable diff string (up to maxLines lines).
func UnifiedDiff(existing, incoming []byte, maxLines int) string
```

#### Step 4 — Ghostty Recommendation

```text
cs delivers Ghostty-native notifications for Claude sessions.

  Ghostty detected ✓  →  proceed
  Ghostty not found   →  To install: brew install --cask ghostty
                         [Press Enter to continue without Ghostty]
```

Detection (`setup.GhosttyInstalled()`): checks `/Applications/Ghostty.app` or `ghostty` on PATH.

#### Step 5 — Notification System

```text
Set up interrupt-driven notifications? Claude sessions will alert you
when they need input, so you can truly multi-task. [Y/n]
```

On accept:

1. Check `terminal-notifier` (`setup.TerminalNotifierInstalled()`)
   - If absent: offer `brew install terminal-notifier`
2. Install notify script: write embedded `assets/notify.sh` → `~/.local/share/cs/notify.sh`, chmod 0700
3. Register hooks via `setup.MergeHooks()`:
   - `Notification` events: matcher `permission_prompt|idle_prompt|elicitation_dialog`
   - `Stop` events: no-op hook (empty matcher, command exits 0 immediately) — registers the hook
     so Claude Code does not time out waiting, but the script does nothing on Stop
4. Fire test notification using `jq -n` to build the payload (never raw string interpolation)
5. Print confirmation; instruct user to allow notifications in macOS System Settings if prompted

On re-run when already installed:

```text
Notifications are installed.
  [t] Send test notification
  [u] Update notify script
  [r] Remove notifications
  [s] Skip
```

**Removal**: `setup.RemoveHooks()` strips cs entries from `~/.claude/settings.json`;
`os.Remove("~/.local/share/cs/notify.sh")`; `setup.RemoveCsBlock("~/.tmux.conf")`.

---

### Phase 3: Hook Registration (Go)

**File**: `internal/setup/hooks.go`

```go
type HookEntry struct {
    EventType string  // "Notification" | "Stop"
    Matcher   string  // regex or empty string
    Command   string  // absolute path — validated before accept
}

// ReadHookSettings reads path and returns the full parsed settings map.
func ReadHookSettings(path string) (map[string]any, error)

// MergeHooks adds cs hook entries without overwriting existing user entries.
// Idempotent: duplicate commands for the same event type are not added.
func MergeHooks(existing map[string]any, entries []HookEntry) (map[string]any, bool)

// RemoveHooks strips entries whose Command matches scriptPath.
func RemoveHooks(existing map[string]any, scriptPath string) (map[string]any, bool)

// WriteHookSettings atomically writes the updated map to path.
// Uses os.CreateTemp + os.Rename to prevent partial writes.
func WriteHookSettings(path string, settings map[string]any) error
```

`HookEntry.Command` MUST be validated as an absolute path before `MergeHooks` accepts it.

Hook format written to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "Notification": [
      {
        "matcher": "permission_prompt|idle_prompt|elicitation_dialog",
        "hooks": [{"type": "command", "command": "~/.local/share/cs/notify.sh"}]
      }
    ],
    "Stop": [
      {
        "matcher": "",
        "hooks": [{"type": "command", "command": "~/.local/share/cs/notify.sh"}]
      }
    ]
  }
}
```

---

## Complexity Tracking

No constitution violations. No complexity table required.
