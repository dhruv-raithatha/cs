# Quickstart: Session Model & Effort Selection

**Branch**: `003-session-model-effort` | **Date**: 2026-04-21

## Creating a New Session

```
$ cs

New session name: infra-refactor
Model:
> sonnet          ← pre-selected (from $ANTHROPIC_MODEL or default)
  opus
  haiku
  sonnet[1m]
  opus[1m]

Effort:
> medium          ← pre-selected (from $CLAUDE_CODE_EFFORT_LEVEL or default)
  low
  high
  xhigh
```

Press Enter on each prompt to accept the default, or navigate with arrow keys and press Enter to select.

**Result**: A tmux session named `infra-refactor` starts with `claude` running under `ANTHROPIC_MODEL=sonnet` and `CLAUDE_CODE_EFFORT_LEVEL=medium`.

---

## Session List with Model & Effort

```
$ cs

> [ + new session ]
  infra-refactor       ~/dev/infra               sonnet       medium
  auth-rework          ~/dev/backend              opus         high   [dead]
  quick-fix            ~/dev/frontend             haiku        low
```

Sessions created before this feature show `unknown` in the model and effort columns:

```
  old-session          ~/dev/legacy               unknown      unknown
```

---

## Using Shell Aliases (Zero Extra Keystrokes)

If you use a shell alias like `claude-opus` that sets environment variables:

```zsh
claude-opus()   { ANTHROPIC_MODEL=opus              CLAUDE_CODE_EFFORT_LEVEL=xhigh cs "$@"; }
claude-opus1m() { ANTHROPIC_MODEL='opus[1m]'        CLAUDE_CODE_EFFORT_LEVEL=xhigh cs "$@"; }
claude-s1m()    { ANTHROPIC_MODEL='sonnet[1m]'      CLAUDE_CODE_EFFORT_LEVEL=high  cs "$@"; }
```

> **Zsh note**: Model aliases containing brackets (e.g. `sonnet[1m]`, `opus[1m]`) **must be
> quoted** when assigned to environment variables in zsh, otherwise zsh treats the brackets as a
> glob pattern and raises "no matches found". Single quotes are safest: `ANTHROPIC_MODEL='opus[1m]'`.

Then running any of these and creating a session will pre-select the matching model for that alias — pressing Enter twice accepts both defaults without changing anything.

---

## cs list

```
$ cs list
NAME                 WORKING DIR                  MODEL        EFFORT  STATUS
infra-refactor       /Users/dhruv/dev/infra       sonnet       medium  active
auth-rework          /Users/dhruv/dev/backend      opus         high    dead

$ cs list --json
{"name":"infra-refactor","working_dir":"/Users/dhruv/dev/infra","model":"sonnet","effort":"medium","status":"active"}
{"name":"auth-rework","working_dir":"/Users/dhruv/dev/backend","model":"opus","effort":"high","status":"dead"}
```

---

## Testing the Feature

```bash
# Verify model/effort are set in the tmux session environment
tmux -S ~/.local/share/cs/cs.sock show-environment -t infra-refactor ANTHROPIC_MODEL
# → ANTHROPIC_MODEL=sonnet

tmux -S ~/.local/share/cs/cs.sock show-options -t infra-refactor @cs-model
# → @cs-model sonnet
```
