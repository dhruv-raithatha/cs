# Research: Interrupt-Driven Session Notifications

**Branch**: `004-session-interrupt-notify` | **Date**: 2026-05-11

## Decisions

### 1. Ghostty-native notification mechanism

**Decision**: Use `terminal-notifier -sender com.mitchellh.ghostty`

**Rationale**: Ghostty's macOS bundle ID is `com.mitchellh.ghostty`. When `terminal-notifier`
receives this `-sender` value, macOS uses Ghostty's icon and app identity for the notification —
visually indistinguishable from a notification Ghostty itself generated. This approach:

- Retains click-to-jump capability (via terminal-notifier's `-execute` flag)
- Requires no changes to the notification delivery pipeline
- Works today without any Ghostty API or escape sequence investigation
- Degrades gracefully on non-Ghostty systems by making the sender configurable

**Alternatives considered**:

- OSC 9 escape sequences via tmux passthrough: Ghostty does support this, but OSC 9 provides
  no click-to-jump mechanism and the escape sequences require `allow-passthrough on` in tmux
  config plus the DCS passthrough wrapper. Rejected as primary mechanism; `allow-passthrough on`
  is still added to tmux.conf to future-proof.
- SwiftUI/native macOS notification API: Would require a separate compiled helper binary.
  Rejected — violates the no-extra-binary constraint.

---

### 2. Stall detection mechanism

**Decision**: tmux `monitor-silence` window option + `alert-silence` hook in tmux.conf

**Rationale**: tmux 3.6a (installed) has built-in `monitor-silence N` that fires `alert-silence`
when a window has produced no output for N seconds. No daemon, no polling loop, no additional
process. The hook calls `notify.sh stall` with the tmux session context available via tmux
format variables in `run-shell`.

**Alternatives considered**:

- Background polling script (cron/launchd): Rejected — requires external process management
  and elevated setup complexity.
- `cs`-side daemon that monitors pane activity: Rejected — violates the no-persistent-daemon
  constraint and would require IPC design.

---

### 3. Notify script delivery location

**Decision**: `~/.local/share/cs/notify.sh`, installed by `cs setup`; source embedded in binary as `cli/assets/notify.sh`

**Rationale**: The binary already embeds `cli/assets/tmux.conf` and installs it on user request.
Using the same pattern for the notify script means `cs setup` is the single authoritative
installation path — no manual dotfiles management needed. The embed ensures the script version
always matches the binary version.

**Alternatives considered**:

- Keeping it in dotfiles only: Rejected — requires manual wiring and defeats the `cs setup` goal.
- Inline shell execution from Go: Rejected — the script is too complex for inline embedding
  and `terminal-notifier` invocation is inherently shell-side.

---

### 4. Hook registration format

**Decision**: Merge into `~/.claude/settings.json` using Claude Code's hook schema

**Rationale**: `~/.claude/settings.json` does not currently have hooks configured (verified by
inspecting the live file). Claude Code expects hooks under the `"hooks"` key with event-type keys
(`"Notification"`, `"Stop"`) containing arrays of matcher+hook objects. The `MergeHooks` function
adds cs entries additively, preserving any user-defined entries.

**Hook format verified from dotfiles reference implementation**:
```json
{
  "hooks": {
    "Notification": [{"matcher": "...", "hooks": [{"type": "command", "command": "path"}]}],
    "Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "path"}]}]
  }
}
```

---

### 5. tmux socket scoping for pane resolution

**Decision**: Scope `tmux list-panes -a` to the cs socket (`-S ~/.local/share/cs/cs.sock`)

**Rationale**: The existing notify.sh resolves panes against the default tmux socket. Since `cs`
uses its own isolated socket, pane resolution must target that socket. This avoids matching panes
from unrelated tmux sessions the user may have running.

---

## Environment Findings

| Item | Status |
| --- | --- |
| Ghostty installed | `/Applications/Ghostty.app` (bundle ID: `com.mitchellh.ghostty`) |
| Ghostty in PATH | Not present — detection uses bundle path check |
| terminal-notifier | Installed at `/opt/homebrew/bin/terminal-notifier` v2.0.0 |
| jq | Installed at `/opt/homebrew/bin/jq` v1.8.1 |
| tmux version | 3.6a (supports `monitor-silence`) |
| Current hooks | None configured in `~/.claude/settings.json` |
| Current sender | `com.apple.Terminal` (dotfiles notify.sh line 87 — to be replaced) |
| allow-passthrough | Absent from current tmux.conf — to be added |
| SPECKIT markers in CLAUDE.md | Absent — to be added in Phase 1 |
