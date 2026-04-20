# Quick-Start Content Design: cs README

This document defines the exact content that will appear in each README section.
It is the design artifact for the README implementation task.

---

## ASCII Logo

```
   ___  ___
  / __\/ __\
 / /  /__\//
/ /___ \/\\
\____/\____/  cs — your sessions, your focus.
```

> Alternatively, a simpler wordmark:

```
 ██████╗███████╗
██╔════╝██╔════╝
██║     ███████╗
██║     ╚════██║
╚██████╗███████║
 ╚═════╝╚══════╝  cs
```

> Final choice is determined during implementation. Either ASCII art or a bold styled text header.
> The block-letter style should feel like a CLI tool, not a SaaS product.

---

## Tagline

```
cs — your sessions, your focus.
```

---

## Value Proposition (2-3 sentences)

```markdown
`cs` lets you run multiple Claude Code sessions in parallel, each named by you for a
specific context — `auth-redesign`, `write-release-notes`, `debug-api-timeout`.
No tmux knowledge required. One command to create, list, attach, or delete sessions.
```

---

## Demo Block (static terminal simulation)

```
$ cs

  > [ + new session ]
    auth-redesign          ~/dev/myapp
    write-release-notes    ~/dev/myapp
    debug-api-timeout      ~/dev/myapp

  ctrl-d: delete   enter: attach   esc: quit
```

---

## Quick-Start Section (5 steps)

```markdown
### Install

**macOS (Homebrew)**
```bash
# Coming soon — build from source for now
```

**Build from source** (requires Go 1.22+)
```bash
git clone https://github.com/dhruv/cs.git
cd cs
make build        # produces ./cs binary
sudo mv cs /usr/local/bin/
```

### First run

```bash
cs setup          # checks tmux + fzf, creates ~/.cs directory
cs                # opens session picker — Enter to create your first session
```
```

---

## Concept / How It Works Section

```markdown
### How it works

You name your sessions. Not Claude.

When you run `cs`, you choose a name that represents what you're working on:
`auth-redesign`, `write-release-notes`, `explore-graphql`. That name is yours —
it reflects your cognitive context, not Claude's.

Under the hood, each session is a tmux window running Claude Code, managed through
a dedicated `cs` socket so it never interferes with your existing tmux setup.
You can have as many sessions as you need and switch between them instantly.

**The mental model:**
- One session per focus area
- Jump between contexts without losing your place
- Dead sessions (Claude exited) appear in the list so you can clean them up
```

---

## Usage Reference Table

| Command | Description |
|---------|-------------|
| `cs` | Open session picker (create or attach) |
| `cs setup` | First-time setup: verify prerequisites, create `~/.cs` directory |
| `cs list` | Print all sessions as a table (non-interactive) |
| `cs list --json` | Print sessions as newline-delimited JSON |
| `cs --version` | Print version |

---

## Prerequisites & Platforms

```markdown
### Prerequisites

- **macOS** (Linux support planned)
- [tmux](https://github.com/tmux/tmux) 3.0+
- [fzf](https://github.com/junegunn/fzf) 0.35+
- Go 1.22+ (build from source only)

Install prerequisites on macOS:
```bash
brew install tmux fzf
```
```

---

## Contributing Section

```markdown
### Contributing

`cs` is a personal tool that happens to be open-source. Issues and PRs are welcome —
open an issue first if you're planning a larger change so we can discuss it before
you invest the time.
```
