# cs — Claude Sessions CLI

A tmux-based session manager for running parallel Claude Code sessions.

## Build & Install

```bash
make build       # builds ./cs binary
make install     # builds and copies to ~/.local/bin/cs
make test        # go test -race ./...
make lint        # golangci-lint run
make hooks       # installs .githooks/pre-commit into .git/hooks/
```

## Architecture

| Package | Purpose |
| --- | --- |
| `cmd/cs` | Entrypoint, wires dependencies |
| `cli/` | Subcommand implementations (list, setup, attach, delete, root) |
| `internal/session` | `Session` type and `Manager` (list/create/kill logic) |
| `internal/tmux` | Exec client: shells out to tmux, parses `list-sessions` output |
| `internal/fzf` | Fuzzy selector wrapper |
| `internal/setup` | Dependency checks, PATH management, tmux.conf helpers |

## Key Details

**tmux socket** — `~/.local/share/cs/cs.sock`, isolated from the user's existing tmux.

**Status detection** — `claude` CLI is a symlink to a versioned binary (e.g. `2.1.126`).
`pane_current_command` returns the binary name, not `"claude"`. Status logic is inverted:
shell or empty = dead, anything else = active.

**Session metadata** — model and effort stored as tmux session options `@cs-model`/`@cs-effort`.
Creation time from `#{session_created}`. Current path from `#{pane_current_path}` (not session
start path, so it reflects where Claude has actually navigated).

**tmux.conf embed** — `cli/assets/tmux.conf` is a copy of `dotfiles/tmux.conf`, embedded
into the binary at compile time so `cs setup` can offer to install it without the repo present.
Keep both files in sync when updating the dotfiles config.

**Install location** — `~/.local/bin/cs`. `cs setup` detects if this is on PATH and offers
to add it to `~/.zshrc` or `~/.bashrc` based on `$SHELL`.

**macOS 26 code signing** — macOS 26 (Tahoe) tightened `taskgated` enforcement: Go's
linker-signed adhoc signature is now rejected at launch with `SIGKILL (Code Signature Invalid)`.
`make build` and `make install` both run `codesign --sign - --force` after copying the binary
to fix this. The `|| true` makes it a no-op on Linux. Symptom: `cs` exits 137 immediately with
a crash report in `~/Library/Logs/DiagnosticReports/` citing `Taskgated Invalid Signature`.

## Pre-commit Hook

Run `make hooks` after clone to install `.githooks/pre-commit`. It runs tests,
golangci-lint, and markdownlint-cli2 (via npx) on staged files.

## Claude Code Integration Notes

**Model and effort — flags not env vars for effort** — `ANTHROPIC_MODEL` is passed as a
tmux env var (`-e`) and works fine. But effort must use the `--effort <level>` CLI flag,
not `CLAUDE_CODE_EFFORT_LEVEL`. When that env var is present in the process, Claude Code
permanently blocks `/effort` overrides for the session's lifetime with the message
`"CLAUDE_CODE_EFFORT_LEVEL=high overrides this session"`. The CLI flag sets the initial
level without polluting the env, so `/effort` can change it freely mid-session.

**Session recovery** — `claude --continue` (or `-c`) resumes the most recent conversation
in the current working directory. Useful for restart-after-crash flows; a future `cs restart`
command should use this.

**Hardened Runtime** — The Claude Code binary has `com.apple.security.cs.allow-jit` and
related JIT entitlements but no `get-task-allow`. `lldb` cannot attach. There is no
OS-level way to modify a running session's environment (e.g. to change effort level) on
macOS without restarting the process. `tmux set-environment` only affects new
windows/panes, not running processes.
