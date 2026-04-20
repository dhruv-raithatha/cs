# Data Model: cs — Claude Session Manager

**Phase**: 1 | **Date**: 2026-04-20

> There is no persistent storage in cs v1. All state is derived at runtime by querying
> the tmux socket server. The entities below describe the runtime data shapes, not database schemas.

---

## Session

The central entity. Represents one tmux session on the cs socket server.

| Field | Type | Source | Notes |
|---|---|---|---|
| `Name` | `string` | User-supplied at creation | Unique within the cs tmux server; enforced by tmux |
| `WorkingDir` | `string` | Absolute path captured at creation | Stored as the tmux session's start directory (`-c` flag) |
| `Status` | `SessionStatus` | Derived at runtime | See enum below |
| `PaneCommand` | `string` | `#{pane_current_command}` query | Used to derive Status; not exposed directly to users |

### SessionStatus Enum

```
Active  — pane current command is "claude"
Dead    — pane current command is a shell (zsh, bash, sh) or empty
```

Status is computed on every list operation. It is never persisted.

---

## SocketConfig

Runtime configuration for the tmux socket. Not a stored entity — resolved at startup.

| Field | Type | Default | Override |
|---|---|---|---|
| `SocketPath` | `string` | `~/.local/share/cs/cs.sock` | `CS_TMUX_SOCKET` env var or `--socket` flag |

---

## SessionList

The ordered collection returned by a list operation, used to populate the fzf picker.

- Ordered by tmux's natural session ordering (creation time, ascending).
- Dead sessions are included and tagged `[dead]` in picker display.
- No filtering by default; `cs list` always shows all cs sessions.

---

## DependencyStatus

Used by `cs setup` only. Not a stored entity.

| Field | Type | Notes |
|---|---|---|
| `Name` | `string` | e.g., "tmux", "fzf", "claude" |
| `Found` | `bool` | true if binary exists on PATH |
| `Version` | `string` | stdout of `<name> --version` or `<name> -V`; empty if not found |
| `InstallCmd` | `string` | Homebrew install command to display if not found |

---

## Interfaces (boundary contracts for testability)

Per Constitution Principle III, every external system interaction is expressed as an interface.

### TmuxClient

```
ListSessions(socketPath string) ([]Session, error)
NewSession(socketPath, name, workingDir string) error
AttachSession(socketPath, name string) error
KillSession(socketPath, name string) error
HasSession(socketPath, name string) (bool, error)
```

### FuzzySelector

```
Select(items []string, prompt string, header string) (string, error)
```

Returns the selected item string, or an error if the user cancels (fzf exits non-zero).

### Executor

```
Run(name string, args ...string) (stdout string, err error)
RunInteractive(name string, args ...string) error
```

`Run` captures output; `RunInteractive` inherits the terminal (used for `cs attach` and
`tmux attach-session`).
