# CLI Contract: cs

**Version**: v1 | **Date**: 2026-04-20

This document defines the command-line interface contract for the `cs` binary. It is
technology-agnostic at the spec level but informed by the urfave/cli framework chosen in research.

---

## Global Flags

| Flag | Short | Default | Env Override | Description |
|---|---|---|---|---|
| `--socket <path>` | — | `~/.local/share/cs/cs.sock` | `CS_TMUX_SOCKET` | Path to the cs tmux socket file |
| `--help` | `-h` | — | — | Show help for any command |

---

## Commands

### `cs` (default — no subcommand)

**Purpose**: Interactive session picker. Attach to an existing session or create a new one.

**Behavior**:
1. If already inside a tmux session → print error and exit 1:
   `"cs: already inside a tmux session — detach first (Ctrl-b d)"`
2. If no sessions exist → prompt for a new session name, then create and attach.
3. If sessions exist → open fzf picker listing all sessions with name, working dir, and status.
   - Selecting an existing session → attach.
   - Choosing "[ + new session ]" → prompt for name → create and attach.
   - Pressing the delete keybind on a session → confirm → kill session → return to picker.

**fzf display format** (one line per session):
```
<name>   <working-dir>   [dead]?
```

**fzf keybindings**:
| Key | Action |
|---|---|
| Enter | Attach to selected session (or confirm new) |
| Ctrl-d | Delete selected session (with confirmation) |
| Esc / Ctrl-c | Exit without action |

**Exit codes**:
| Code | Meaning |
|---|---|
| 0 | Session attached or created successfully |
| 1 | User cancelled (Esc), already in tmux, or name conflict resolved |
| 2 | Unexpected internal error |

---

### `cs setup`

**Purpose**: First-run dependency check and optional installation.

**Behavior**:
1. Check for `tmux`, `fzf`, `claude` on PATH.
2. Print status for each: `✓ tmux 3.4` or `✗ fzf — not found`.
3. For each missing dep, offer to install via Homebrew (or npm for `claude`).
4. Create `~/.local/share/cs/` directory if it does not exist.
5. Exit 0 if all deps present; exit 1 if any dep remains missing after user declines install.

**Output** (non-JSON):
```
Checking dependencies...
  ✓ tmux 3.4
  ✗ fzf — not found
    Install with: brew install fzf [Y/n]
  ✓ claude 1.2.3

Setup complete. Run `cs` to start.
```

**Exit codes**:
| Code | Meaning |
|---|---|
| 0 | All dependencies satisfied |
| 1 | One or more dependencies missing after setup |

---

### `cs list [--json]`

**Purpose**: List all cs-managed sessions non-interactively.

**Default output** (human-readable):
```
NAME          WORKING DIR             STATUS
my-project    /Users/dhruv/dev/foo    active
old-work      /Users/dhruv/dev/bar    dead
```

**`--json` output** (newline-delimited JSON, one object per session):
```json
{"name":"my-project","working_dir":"/Users/dhruv/dev/foo","status":"active"}
{"name":"old-work","working_dir":"/Users/dhruv/dev/bar","status":"dead"}
```

**Behavior when not a TTY**: `cs list` without `--json` still outputs human-readable table.
`--json` is always machine-readable regardless of TTY state.

**Exit codes**:
| Code | Meaning |
|---|---|
| 0 | Success (zero sessions is still 0) |
| 2 | Internal error querying tmux |

---

### `cs attach <name>`

**Purpose**: Attach to a named session non-interactively (bypasses picker).

**Exit codes**:
| Code | Meaning |
|---|---|
| 0 | Successfully attached |
| 1 | Session not found |
| 2 | Internal error |

---

### `cs delete <name>`

**Purpose**: Kill a named session non-interactively (bypasses picker).

**Behavior**: Kills the tmux session. Does not prompt for confirmation (non-interactive path;
confirmation is in the interactive picker flow).

**Exit codes**:
| Code | Meaning |
|---|---|
| 0 | Session deleted |
| 1 | Session not found |
| 2 | Internal error |

---

### `cs version`

**Purpose**: Print the cs version string.

**Output**: `cs version v0.1.0`

**Exit code**: Always 0.

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `CS_TMUX_SOCKET` | `~/.local/share/cs/cs.sock` | Path to the cs tmux socket file |

All env vars are documented in `cs --help` output.
