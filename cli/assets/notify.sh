#!/usr/bin/env bash
# notify.sh — Ghostty-native Claude session notification hook.
# Managed by cs setup. Do not edit manually; reinstall via: cs setup → notifications → update.
#
# Invocation modes:
#   stdin JSON  — Claude Code hook (Notification, Stop events)
#   stall       — tmux alert-silence hook (positional arg $1 == "stall")

set -euo pipefail

TERMINAL_APP="${CS_NOTIFY_TERMINAL:-Ghostty}"
CS_SOCK="${CS_TMUX_SOCKET:-$HOME/.local/share/cs/cs.sock}"
LOG_DIR="$HOME/.cs"
LOG_FILE="$LOG_DIR/notification.log"

# Ensure log directory and file exist with secure permissions (0600).
mkdir -p "$LOG_DIR"
install -m 0600 /dev/null "$LOG_FILE" 2>/dev/null || true

log() {
    local clean
    # Strip ANSI escape sequences before writing to the log.
    clean=$(printf '%s' "$1" | sed 's/\x1b\[[0-9;]*m//g')
    printf '%s: %s\n' "$(date)" "$clean" >> "$LOG_FILE"
}

# ── Derive notification content by invocation mode ───────────────────────────

HOOK_TYPE=""
SESSION_ID=""
CWD=""
MSG=""
NOTIFICATION_TYPE=""

if [[ "${1:-}" == "stall" ]]; then
    # Stall mode: invoked by tmux set-hook alert-silence.
    # TMUX_SESSION and TMUX_WINDOW are set by tmux run-shell.
    SESSION_ID="${TMUX_SESSION:-unknown}"
    NOTIFICATION_TYPE="stall"
    HOOK_TYPE="Notification"

    CWD=$(tmux -S "$CS_SOCK" display-message -p '#{pane_current_path}' 2>/dev/null || true)
    IDLE_SECS=$(tmux -S "$CS_SOCK" display-message -p '#{window_silence_interval}' 2>/dev/null || echo "0")
    IDLE_MINS=$(( (${IDLE_SECS:-0} + 59) / 60 ))
    MSG="Session idle for ${IDLE_MINS}m"
else
    # Hook mode: Claude Code sends JSON payload on stdin.
    INPUT=$(cat)
    HOOK_TYPE=$(printf '%s' "$INPUT" | jq -r '.hook_event_name // ""')
    SESSION_ID=$(printf '%s' "$INPUT" | jq -r '.session_id // ""')
    CWD=$(printf '%s' "$INPUT" | jq -r '.cwd // ""')
    MSG=$(printf '%s' "$INPUT" | jq -r '.message // ""')
    NOTIFICATION_TYPE=$(printf '%s' "$INPUT" | jq -r '.notification_type // ""')
fi

# ── Stop hook: exit 0, no notification, no file write ───────────────────────
if [[ "$HOOK_TYPE" == "Stop" ]]; then
    log "Stop hook in $CWD (session: $SESSION_ID) — no notification"
    exit 0
fi

# ── Resolve tmux pane by walking the PID tree ────────────────────────────────
TMUX_TARGET=""
CLAUDE_PID=""
CHECK_PID=$$

for _ in 1 2 3 4 5; do
    PARENT=$(ps -o ppid= -p "$CHECK_PID" 2>/dev/null | tr -d ' ')
    [[ -z "$PARENT" || "$PARENT" == "1" ]] && break
    PNAME=$(ps -o comm= -p "$PARENT" 2>/dev/null | xargs basename 2>/dev/null || true)
    if [[ "$PNAME" == "claude" ]]; then
        CLAUDE_PID="$PARENT"
        break
    fi
    CHECK_PID="$PARENT"
done

if [[ -n "$CLAUDE_PID" ]]; then
    CLAUDE_TTY=$(ps -o tty= -p "$CLAUDE_PID" 2>/dev/null | tr -d ' ')
    if [[ -n "$CLAUDE_TTY" && "$CLAUDE_TTY" != "??" ]]; then
        [[ "$CLAUDE_TTY" != /dev/* ]] && CLAUDE_TTY="/dev/$CLAUDE_TTY"
        TMUX_TARGET=$(tmux -S "$CS_SOCK" list-panes -a \
            -F '#{pane_tty} #{session_name}:#{window_index}.#{pane_index}' 2>/dev/null \
            | grep "^${CLAUDE_TTY} " | awk '{print $2}' || true)
    fi
fi

# Stall mode fallback: use session name directly as the target.
if [[ "${1:-}" == "stall" && -z "$TMUX_TARGET" ]]; then
    TMUX_TARGET="${TMUX_SESSION:-}"
fi

# ── Validate TMUX_TARGET before embedding in -execute shell string ───────────
# Allowlist: session names, window/pane indices, and separators only.
# Any character outside [A-Za-z0-9_.:-] is a potential injection vector.
EXECUTE_CMD=""
if [[ -n "$TMUX_TARGET" ]]; then
    if [[ "$TMUX_TARGET" =~ ^[A-Za-z0-9_.:-]+$ ]]; then
        WINDOW="${TMUX_TARGET%.*}"
        EXECUTE_CMD="tmux -S '${CS_SOCK}' select-window -t '${WINDOW}' \
&& tmux -S '${CS_SOCK}' select-pane -t '${TMUX_TARGET}' \
&& open -a '${TERMINAL_APP}'"
    else
        log "WARNING: TMUX_TARGET '${TMUX_TARGET}' failed allowlist — click-to-jump disabled"
    fi
fi

# ── Build notification content ───────────────────────────────────────────────
PROJECT_NAME=$(printf '%s' "$CWD" | awk -F'/' '{print $(NF-1)"/"$NF}')
TITLE="Claude: $PROJECT_NAME"
SUBTITLE="${SESSION_ID:0:8}"
MSG_PREVIEW="${MSG:0:120}"
GROUP_ID="cs-${SESSION_ID:0:8}"

# ── Dispatch via terminal-notifier ───────────────────────────────────────────
if ! command -v terminal-notifier &>/dev/null; then
    log "terminal-notifier not installed — notification skipped ($NOTIFICATION_TYPE in $CWD)"
    exit 0
fi

NOTIFY_ARGS=(
    -title    "$TITLE"
    -subtitle "$SUBTITLE"
    -message  "$MSG_PREVIEW"
    -sound    Glass
    -group    "$GROUP_ID"
    -sender   com.mitchellh.ghostty
)
if [[ -n "$EXECUTE_CMD" ]]; then
    NOTIFY_ARGS+=(-execute "$EXECUTE_CMD")
fi

# Run terminal-notifier in the background so we never block the hook caller.
# Redirect stdout/stderr to the log to suppress the notification ID output.
terminal-notifier "${NOTIFY_ARGS[@]}" >>"$LOG_FILE" 2>&1 &
log "$NOTIFICATION_TYPE in $CWD (session: $SESSION_ID) tmux=$TMUX_TARGET"
exit 0
