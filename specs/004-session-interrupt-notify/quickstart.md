# Quickstart: Interrupt-Driven Session Notifications

**Branch**: `004-session-interrupt-notify` | **Date**: 2026-05-11

## End-to-end verification (developer testing guide)

### Prerequisites

1. `cs` binary built and on PATH: `make install`
2. Ghostty installed at `/Applications/Ghostty.app`
3. Ghostty is the terminal you are running this in

### 1. Run setup

```bash
cs setup
```

Walk through the steps:

- Deps: skip if tmux/fzf/claude already present
- PATH: skip if already configured
- tmux.conf: install if not present
- **Ghostty**: expect "Ghostty detected ✓"
- **Notifications**: accept the opt-in

At the end of the notification step, a test notification fires. Verify it:

- Shows Ghostty's icon (not Terminal or a generic icon)
- Title says "Claude: `<project dir>`"
- Message says "cs notifications active — click to test jump"
- Clicking it focuses Ghostty

If macOS shows a system prompt to allow notifications from Ghostty, accept it.

### 2. Test human-in-the-loop notification

```bash
# Start a new cs session in any project
cs new myproject

# Inside the session, run a claude command that requires permission
# (or simply wait for Claude to ask a question)
```

Expected: notification appears within 5 seconds of Claude's question/permission prompt.
Click the notification: Ghostty focuses and the correct tmux pane becomes active.

### 3. Test stall detection

```bash
# Reduce stall threshold for testing
export CS_STALL_THRESHOLD=30  # 30 seconds

cs new stall-test

# Inside the session, run: claude (then do nothing for 30+ seconds)
```

Expected: after ~30 seconds of silence, a stall notification fires with message
"Session idle for N seconds".

### 4. Test replace-not-stack

Trigger two events in quick succession from the same session. Verify only one notification
appears in macOS notification center (the newer one replaced the older).

### 5. Verify removal

```bash
cs setup
# Choose "remove" when prompted about notifications
```

Verify: `~/.local/share/cs/notify.sh` deleted, cs hook entries removed from
`~/.claude/settings.json`.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Notification shows Terminal icon | Verify `terminal-notifier` version ≥ 2.0.0; earlier versions may ignore `-sender` |
| Click does nothing | Check `CS_TMUX_SOCKET` points to the correct cs socket; verify tmux pane resolution in `~/.cs/notification.log` |
| No stall notifications | Verify `monitor-silence` is set on the window: `tmux -S ~/.local/share/cs/cs.sock show-options -w monitor-silence` |
| Notifications not appearing at all | Check `terminal-notifier` is installed: `which terminal-notifier` |
| duplicate notifications | Check group ID in log — `-group "cs-<session_id>"` should deduplicate |
