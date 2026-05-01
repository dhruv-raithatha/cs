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

## Pre-commit Hook

Run `make hooks` after clone to install `.githooks/pre-commit`. It runs tests,
golangci-lint, and markdownlint-cli2 (via npx) on staged files.
