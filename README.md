```
 ██████╗███████╗
██╔════╝██╔════╝
██║     ███████╗
██║     ╚════██║
╚██████╗███████║
 ╚═════╝╚══════╝
```

# cs — your sessions, your focus.

[![Go](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go)](https://go.dev)
[![Build](https://img.shields.io/badge/build-passing-brightgreen)](#)
[![License](https://img.shields.io/badge/license-MIT-blue)](#license)

---

`cs` lets you run multiple Claude Code sessions in parallel, each named by **you** for a
specific context — `auth-redesign`, `write-release-notes`, `debug-api-timeout`.
No tmux knowledge required. One command to create, list, attach, or delete sessions.

```
$ cs

  > [ + new session ]
    auth-redesign          ~/dev/myapp
    write-release-notes    ~/dev/myapp
    debug-api-timeout      ~/dev/myapp

  ctrl-d: delete   enter: attach   esc: quit
```

---

## Quick Start

**Prerequisites** — macOS, [tmux](https://github.com/tmux/tmux) 3.0+, [fzf](https://github.com/junegunn/fzf) 0.35+:

```bash
brew install tmux fzf
```

**Install** (Go 1.22+ required — Homebrew tap coming soon):

```bash
git clone https://github.com/dhruv/cs.git
cd cs
make build
sudo mv cs /usr/local/bin/
```

**First run:**

```bash
cs setup    # verify prerequisites, create ~/.cs
cs          # open session picker — press Enter to create your first session
```

---

## How It Works

You name your sessions. Not Claude.

When you run `cs`, you choose a name that represents what you're working on right now:

| Session name | What you're doing |
|---|---|
| `auth-redesign` | Reworking authentication on a feature branch |
| `write-release-notes` | Drafting the changelog for the next release |
| `debug-api-timeout` | Tracking down a flaky timeout in production |
| `explore-graphql` | Spiking on a new technology, separate from your main work |

That name is yours — it reflects your cognitive context, not Claude's.

Under the hood, each session is a tmux window running Claude Code, managed through a dedicated
`cs` socket so it never interferes with your existing tmux setup. Jump between contexts
instantly without losing your place.

---

## Usage

| Command | Description |
|---------|-------------|
| `cs` | Open session picker — create a new session or attach to an existing one |
| `cs setup` | First-time setup: verify prerequisites, create `~/.cs` |
| `cs list` | Print all sessions as a table (non-interactive) |
| `cs list --json` | Print sessions as newline-delimited JSON |
| `cs --version` | Print version |

**Keyboard shortcuts in the picker:**

| Key | Action |
|-----|--------|
| `Enter` | Create new session (default) or attach to selected session |
| `ctrl-d` | Delete the selected session |
| `esc` | Quit without changes |

---

## Platform Support

| Platform | Status |
|----------|--------|
| macOS | ✅ Supported |
| Linux | 🔜 Planned |
| Windows | ❌ Not supported (tmux unavailable) |

---

## Contributing

`cs` is a personal tool that happens to be open-source. Issues and pull requests are welcome —
please open an issue first if you're planning a larger change so we can align before you invest
the time.

---

## License

MIT
