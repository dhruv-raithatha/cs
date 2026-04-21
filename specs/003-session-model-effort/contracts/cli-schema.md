# CLI Contract: Session Model & Effort Selection

**Branch**: `003-session-model-effort` | **Date**: 2026-04-21

## cs (root command) — interactive picker

### Session List Entry Format (updated)

Each existing session is rendered as a single line:

```
%-20s %-28s %-12s %-7s [dead?]
  ^       ^         ^       ^       ^
  name    workingDir  model  effort  [dead] (optional)
```

- `model` and `effort` show `"unknown"` when the session has no stored values.
- `[dead]` suffix is appended only when `Status == Dead`.
- Line fits within 77 columns for standard 80-column terminals.

### New Session Creation Flow (updated)

After the name prompt, two additional fzf prompts appear in sequence:

**Model prompt**:
```
prompt: "Model: "
header: "enter: select model"
items:  [<env-default-or-sonnet>, ...remaining models in order...]
```

**Effort prompt**:
```
prompt: "Effort: "
header: "enter: select effort level"
items:  [<env-default-or-medium>, ...remaining effort levels in order...]
```

Pressing Enter on the pre-selected first item accepts the default. Pressing Escape (fzf exit 130) cancels session creation.

---

## cs list

### Human-readable table (updated)

```
NAME                 WORKING DIR                  MODEL        EFFORT  STATUS
<name>               <dir>                        <model>      <effort> <active|dead>
```

Header columns: `NAME`, `WORKING DIR`, `MODEL`, `EFFORT`, `STATUS`.

### JSON output (updated)

Each session emits one line of JSON with the following keys:

```json
{
  "name":        "<session name>",
  "working_dir": "<absolute path>",
  "model":       "<model alias or empty string>",
  "effort":      "<effort level or empty string>",
  "status":      "active | dead"
}
```

- `model` and `effort` are empty strings (not `"unknown"`) in JSON to allow callers to distinguish "no value recorded" from a literal value.

---

## tmux Session Options Written by cs

When `cs` creates a session, it writes two session options immediately after `new-session -d`:

| tmux command                                      | Purpose                     |
|---------------------------------------------------|-----------------------------|
| `tmux -S <socket> set-option -t <name> @cs-model <model>` | Persist model alias |
| `tmux -S <socket> set-option -t <name> @cs-effort <effort>` | Persist effort level |

These options are queried via the `list-sessions` format string `#{@cs-model}` and `#{@cs-effort}`.

---

## tmux new-session Invocation (updated)

```
tmux -S <socket> new-session -d -s <name> -c <workingDir> \
  -e ANTHROPIC_MODEL=<model> \
  -e CLAUDE_CODE_EFFORT_LEVEL=<effort> \
  claude
```

The `-e` flags inject the chosen values into the tmux session environment so `claude` inherits them without requiring the shell profile to be sourced.

---

## Environment Variables Read by cs

| Variable                  | Purpose                                       |
|---------------------------|-----------------------------------------------|
| `ANTHROPIC_MODEL`         | Pre-selects model in the model picker         |
| `CLAUDE_CODE_EFFORT_LEVEL`| Pre-selects effort level in the effort picker |
| `CS_TMUX_SOCKET`          | tmux socket path (existing, unchanged)        |
| `TMUX`                    | Guard: non-empty means already in tmux (existing) |

---

## Exit Codes (unchanged)

| Code | Meaning                                      |
|------|----------------------------------------------|
| 0    | Success                                      |
| 1    | User error (session not found, bad args)     |
| 2    | Internal / unexpected error                  |
