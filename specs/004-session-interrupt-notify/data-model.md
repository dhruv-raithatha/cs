# Data Model: Interrupt-Driven Session Notifications

**Branch**: `004-session-interrupt-notify` | **Date**: 2026-05-11

## Entities

### NotificationEvent (shell-side)

Represents a Claude Code hook payload received on stdin by the notify script.

| Field | Type | Source | Notes |
| --- | --- | --- | --- |
| `session_id` | string | Claude Code | UUID for the Claude session |
| `cwd` | string | Claude Code | Working directory of the Claude process |
| `message` | string | Claude Code | Human-readable event message |
| `hook_event_name` | string | Claude Code | `Notification`, `Stop`, etc. |
| `notification_type` | string | Claude Code | `permission_prompt`, `idle_prompt`, etc. |
| `usage.input_tokens` | int | Claude Code (Stop only) | For context cache display |
| `usage.output_tokens` | int | Claude Code (Stop only) | For context cache display |

### StallEvent (shell-side, tmux-generated)

Synthesised by the notify script when invoked with the `stall` positional argument from tmux's
`alert-silence` hook. Context comes from tmux environment variables available in `run-shell`.

| Field | Derived From | Notes |
| --- | --- | --- |
| `session_name` | `$TMUX_SESSION` or tmux format | cs session name |
| `cwd` | `tmux display-message -p #{pane_current_path}` | Current pane path |
| `idle_seconds` | `tmux display-message -p #{window_silence_interval}` | Elapsed silence |

### HookEntry (Go, `internal/setup/hooks.go`)

In-memory representation of a single Claude Code hook registration.

```go
type HookEntry struct {
    EventType string  // "Notification" | "Stop"
    Matcher   string  // regex or empty string
    Command   string  // absolute path to notify script
}
```

### HookSettings (Go, `internal/setup/hooks.go`)

Full parsed structure of `~/.claude/settings.json`, with the hooks subtree extracted.

```go
type HookSettings struct {
    Raw     map[string]any            // full settings (for round-trip preservation)
    Hooks   map[string][]HookGroup    // parsed hooks subtree
}

type HookGroup struct {
    Matcher string       `json:"matcher"`
    Hooks   []HookDef    `json:"hooks"`
}

type HookDef struct {
    Type    string  `json:"type"`    // always "command"
    Command string  `json:"command"`
}
```

## State Transitions

### Setup notification state

```text
Not installed
    │
    └─ [cs setup: opt in] ──────────────────────────────────────────────────────┐
                                                                                 │
                                                                         Installed
                                                                         (script + hooks)
                                                                                 │
                                                         ┌───────────────────────┤
                                                         │                       │
                                            [cs setup: update]       [cs setup: remove]
                                                         │                       │
                                                    Updated                Not installed
```

### Session notification lifecycle

```text
Session created (cs new)
    │
    └─ monitor-silence set on window
            │
            ├─ Claude emits event → hook fires → notify.sh (JSON stdin) → notification
            │
            └─ N seconds silence → alert-silence → notify.sh stall → stall notification
                    │
                    └─ session becomes active again → monitor-silence resets
```

## Validation Rules

- `HookEntry.Command` MUST be an absolute path (enforced during `MergeHooks`)
- `HookEntry.EventType` MUST be one of the known Claude Code event type strings
- `WriteHookSettings` MUST use atomic write (temp file + rename) to avoid partial writes
- Notification message preview MUST be capped at 120 characters (enforced in notify script)
- Stall threshold MUST be a positive integer (seconds); defaults to 180
