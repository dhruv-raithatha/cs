# Data Model: Session Model & Effort Selection

**Branch**: `003-session-model-effort` | **Date**: 2026-04-21

## Extended Entity: Session

`internal/session/session.go`

```go
type Session struct {
    Name        string
    WorkingDir  string
    Status      SessionStatus
    PaneCommand string
    Model       string // Claude model alias; empty for pre-existing sessions
    Effort      string // Effort level; empty for pre-existing sessions
}
```

**Invariants**:
- `Model` and `Effort` may be empty for sessions created before this feature; callers render them as `"unknown"` in UI.
- `Model` and `Effort` are never validated by `cs` — the values are passed through to Claude verbatim.

---

## Updated Interface: TmuxClient

`internal/tmux/client.go`

```go
type TmuxClient interface {
    ListSessions(socketPath string) ([]session.Session, error)
    NewSession(socketPath, name, workingDir, model, effort string) error
    AttachSession(socketPath, name string) error
    KillSession(socketPath, name string) error
    HasSession(socketPath, name string) (bool, error)
}
```

**Change**: `NewSession` gains two trailing parameters `model, effort string`.

---

## Updated Interface: session.Client

`internal/session/manager.go`

```go
type Client interface {
    ListSessions(socketPath string) ([]session.Session, error)
    NewSession(socketPath, name, workingDir, model, effort string) error
    AttachSession(socketPath, name string) error
    KillSession(socketPath, name string) error
    HasSession(socketPath, name string) (bool, error)
}
```

Same signature change as `TmuxClient`.

---

## Updated Method: Manager.NewSession

`internal/session/manager.go`

```go
func (m *Manager) NewSession(socketPath, name, workingDir, model, effort string) error
```

Passes `model` and `effort` through to `m.client.NewSession`.

---

## tmux Session Option Keys

| Option Key  | Value                  | Set via                                    |
|-------------|------------------------|--------------------------------------------|
| `@cs-model` | e.g. `opus`, `sonnet`  | `tmux set-option -t <name> @cs-model <v>` |
| `@cs-effort`| e.g. `high`, `medium`  | `tmux set-option -t <name> @cs-effort <v>`|

Queried in `list-sessions` via format tokens `#{@cs-model}` and `#{@cs-effort}`.

---

## tmux ListSessions Format String

**Before**: `#{session_name}:#{session_path}:#{pane_current_command}`  
**After**:  `#{session_name}:#{session_path}:#{pane_current_command}:#{@cs-model}:#{@cs-effort}`

Parse with `strings.SplitN(line, ":", 5)` → `[name, workingDir, paneCommand, model, effort]`.

---

## Known Model List (ordered)

```go
var knownModels = []string{"sonnet", "opus", "haiku", "sonnet[1m]", "opus[1m]"}
```

Defined in `cli/root.go`. Default resolved from `os.Getenv("ANTHROPIC_MODEL")`, falling back to `"sonnet"`.

---

## Known Effort List (ordered)

```go
var knownEfforts = []string{"low", "medium", "high", "xhigh"}
```

Defined in `cli/root.go`. Default resolved from `os.Getenv("CLAUDE_CODE_EFFORT_LEVEL")`, falling back to `"medium"`.

---

## Updated Test Double: FakeTmuxClient

`internal/tmux/fake.go`

```go
type FakeTmuxClient struct {
    Sessions         []session.Session
    ListSessionsErr  error
    NewSessionErr    error
    AttachSessionErr error
    KillSessionErr   error
    HasSessionResult bool
    HasSessionErr    error

    AttachedSession string
    KilledSession   string
    CreatedSession  string
    CreatedModel    string  // NEW
    CreatedEffort   string  // NEW
}

func (f *FakeTmuxClient) NewSession(_, name, _, model, effort string) error {
    f.CreatedSession = name
    f.CreatedModel   = model
    f.CreatedEffort  = effort
    return f.NewSessionErr
}
```

---

## Updated Function Signature: createNewSession

`cli/root.go`

```go
// Before:
func createNewSession(socketPath string, client tmux.TmuxClient, stdin io.Reader) error

// After:
func createNewSession(socketPath string, client tmux.TmuxClient, selector fzf.FuzzySelector, stdin io.Reader) error
```

Internal logic adds two `selector.Select()` calls (model, then effort) between name prompt and session creation.

---

## Picker Entry Format

**Before**: `%-20s %-40s [dead?]`  
**After**:  `%-20s %-28s %-12s %-7s [dead?]`

Column map: `name | workingDir | model | effort | [dead]`

Max width on 80-col terminal: `20+1+28+1+12+1+7+1+6 = 77` ✓

Empty model/effort from pre-existing sessions rendered as `"unknown"` in the display line.

---

## cs list Output

`cli/list.go` — both `printTable` (human) and `printJSON` (machine) updated:

**printTable header**: `NAME  WORKING DIR  MODEL  EFFORT  STATUS`

**printJSON fields added**: `"model"` and `"effort"` keys (empty string for pre-existing sessions — not `"unknown"`; callers decide how to render).
