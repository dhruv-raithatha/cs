# Research: cs — Claude Session Manager

**Phase**: 0 | **Date**: 2026-04-20 | **Branch**: `001-cs-tmux-session-manager`

## Decision 1: CLI Framework

**Decision**: `urfave/cli` (v3)

**Rationale**: urfave/cli has zero external dependencies (pure stdlib), which directly satisfies
Constitution Principle VI. It supports subcommands, accurate `--help` generation, env var binding,
and shell completion out of the box. For a ~5-subcommand single-developer tool, cobra's code
generation tooling and heavier dependency surface are unnecessary overhead.

**Alternatives considered**:
- `cobra` — larger ecosystem and more stars, but adds external deps (pflag). Better suited to
  multi-contributor projects or tools with 20+ subcommands.
- `stdlib flag` — no subcommand support without significant boilerplate; rejected.

---

## Decision 2: tmux Socket Path

**Decision**: Single socket file at `~/.local/share/cs/cs.sock` (overridable via `CS_TMUX_SOCKET`)

**Rationale**: tmux's `-S <path>` flag directs all tmux commands to a specific socket file. Every
cs session lives on one tmux server bound to this socket. This gives complete isolation from the
user's personal tmux environment (`tmux ls` without `-S` shows nothing from cs). The XDG data home
path (`~/.local/share/cs/`) keeps state out of dotfiles clutter and survives shell reloads.

**Correction from spec**: The spec referred to a "socket directory" — tmux uses a single socket
*file* per server, not a directory. The cs socket path is a file, not a folder. The directory
`~/.local/share/cs/` is created by `cs setup`; the socket file `cs.sock` is created by tmux on
first use.

**Alternatives considered**:
- `~/.cs/cs.sock` — simpler, but not XDG-compliant; pollutes the home directory.
- `/tmp/cs-<uid>.sock` — lost on reboot; unsuitable for persistent long-lived sessions.
- Separate socket per session — unnecessary complexity; one server handles N sessions cleanly.

---

## Decision 3: Dead Process Detection

**Decision**: Query `#{pane_current_command}` via `tmux list-panes -t <session> -F '#{pane_current_command}' -S <socket>`

**Rationale**: When cs creates a session, it runs `claude` as the initial pane command
(`tmux new-session -d -s <name> -c <dir> 'claude'`). If the pane's current command later reports
a shell name (`zsh`, `bash`, `sh`) or nothing, the Claude process has exited and the session is
`Dead`. This is a stateless runtime check with no metadata file to maintain.

**Alternatives considered**:
- Track PID in a metadata file — fragile (PIDs reused; file can go stale); extra state.
- Watch-style polling daemon — massive overkill for a local picker tool.
- `tmux list-panes ... #{pane_pid}` + `/proc/<pid>/cmdline` check — Linux-only; not portable to macOS.

---

## Decision 4: fzf Invocation Pattern

**Decision**: Pipe session list to fzf's stdin; capture selected line from stdout; let fzf auto-open `/dev/tty` for its UI

**Rationale**: When fzf's stdin is not a terminal (i.e., a pipe), fzf automatically reads its
keyboard UI from `/dev/tty` and renders there. The caller pipes the item list to `cmd.Stdin` via
`strings.NewReader`, captures `cmd.Stdout` into a `bytes.Buffer`, and reads the selected line after
`cmd.Wait()`. This is fzf's documented behavior and requires no special `/dev/tty` handling on the
Go side.

**Key flags used**: `--height 40%` (compact overlay), `--no-sort` (preserve session order),
`--prompt` (contextual prompt text), `--header` (keybinding hints).

**Alternatives considered**:
- fzf Go libraries (e.g., `ktr0731/go-fuzzyfinder`) — violates the constitution's preference for
  invoking the installed fzf binary. Adds a dependency and diverges from the user's configured fzf.

---

## Decision 5: Session Name Uniqueness

**Decision**: Rely on tmux server enforcement within the cs socket

**Rationale**: tmux returns a non-zero exit code when creating a session with a name that already
exists. The cs `SessionManager` detects this error and redirects to the attach flow (FR-012).
No additional uniqueness tracking is needed.

---

## Decision 6: `cs setup` Scope

**Decision**: Check for `tmux`, `fzf`, `claude` on PATH; offer Homebrew install for each missing dep; create `~/.local/share/cs/` directory

**Rationale**: These are the only runtime dependencies. Homebrew is the standard macOS package
manager. `claude` is installed via `npm i -g @anthropic-ai/claude-code` — `cs setup` will detect
whether `npm` is available and guide accordingly. `cs setup` also creates the socket directory so
the user does not need to do it manually.
