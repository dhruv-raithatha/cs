# Contract: Notify Script Stdin Payload

**File**: `~/.local/share/cs/notify.sh`
**Invocation modes**: stdin JSON (Claude Code hooks) | positional arg `stall` (tmux alert-silence)

## Mode 1: Claude Code Hook (stdin JSON)

The script is invoked by Claude Code with a JSON object on stdin.

### Notification event

```json
{
  "session_id": "abc12345-...",
  "cwd": "/Users/dhruv/dev/myproject/src",
  "message": "I need your approval to run this command...",
  "hook_event_name": "Notification",
  "notification_type": "permission_prompt"
}
```

### Stop event

```json
{
  "session_id": "abc12345-...",
  "cwd": "/Users/dhruv/dev/myproject",
  "hook_event_name": "Stop",
  "usage": {
    "input_tokens": 45230,
    "output_tokens": 1204
  }
}
```

**Behaviour on Stop**: Write context cache to `~/.cache/claude-context.txt`; do NOT dispatch a
notification. Exit 0.

## Mode 2: Stall (tmux alert-silence)

Invoked by tmux as: `run-shell -b "~/.local/share/cs/notify.sh stall"`

When called with `stall` as `$1`, the script reads context from tmux variables available in
the `run-shell` environment:

- `#{session_name}` — cs session name
- `#{pane_current_path}` — working directory
- `#{window_silence_interval}` — seconds of silence

The script synthesises a notification with message: `"Session idle for <N> minutes"`.

## Output

- On success: exits 0, appends one line to `~/.cs/notification.log`
- On `terminal-notifier` absent: logs warning to `~/.cs/notification.log`, exits 0 (graceful)
- On malformed JSON: logs parse error, exits 0 (never crash the hook)

## Environment Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `CS_NOTIFY_TERMINAL` | `Ghostty` | Terminal app name for `open -a` on click |
| `CS_TMUX_SOCKET` | `~/.local/share/cs/cs.sock` | Socket for pane resolution |
| `CS_STALL_LABEL` | `stall` | Positional arg name for stall invocation |
