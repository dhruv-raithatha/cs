# cs — your sessions, your focus

```text
 ██████╗███████╗
██╔════╝██╔════╝
██║     ███████╗
██║     ╚════██║
╚██████╗███████║
 ╚═════╝╚══════╝
```

[![Go](https://img.shields.io/badge/go-1.26%2B-00ADD8?logo=go)](https://go.dev)
![Build](https://img.shields.io/badge/build-passing-brightgreen)
[![License](https://img.shields.io/badge/license-MIT-blue)](#license)

---

`cs` lets you run multiple Claude Code sessions in parallel, each named by **you** for a
specific context — `auth-redesign`, `write-release-notes`, `debug-api-timeout`.
No tmux knowledge required. One command to create, list, attach, or delete sessions.

```text
$ cs

  > [ + new session ]
    auth-redesign          ~/dev/myapp   sonnet[1m]  medium  2h
    write-release-notes    ~/dev/myapp   opus        high    1d
    debug-api-timeout      ~/dev/myapp   sonnet[1m]  low     3h

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
make install   # builds and copies to ~/.local/bin/cs
```

**First run:**

```bash
cs setup    # verify prerequisites, update PATH, optionally copy tmux config
cs          # open session picker — press Enter to create your first session
```

---

## How It Works

You name your sessions. Not Claude.

When you run `cs`, you choose a name that represents what you're working on right now:

| Session name | What you're doing |
| --- | --- |
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
| --- | --- |
| `cs` | Open session picker — create a new session or attach to an existing one |
| `cs setup` | First-time setup: install deps, update PATH, copy tmux config |
| `cs list` | Print active sessions (name, current path, model, effort, age) |
| `cs list --all` | Print all sessions including dead ones, with status indicator |
| `cs list --json` | Print sessions as newline-delimited JSON |
| `cs delete <name>` | Delete a session by name |
| `cs --version` | Print version |

**Keyboard shortcuts in the picker:**

| Key | Action |
| --- | --- |
| `Enter` | Create new session (default) or attach to selected session |
| `ctrl-d` | Delete the selected session |
| `esc` | Quit without changes |

---

## Optional: Enhanced tmux Configuration

`cs` works with any tmux setup, but since every session runs inside tmux, a
well-tuned config makes a real difference — especially for multi-session,
multi-pane Claude Code workflows.

A ready-to-use config lives in [`dotfiles/tmux.conf`](dotfiles/tmux.conf). It
addresses three specific pain points that come up when using `cs` heavily:

| Pain point | What the config does |
| --- | --- |
| Shift+Enter doesn't work inside tmux | `extended-keys on` passes the key through to Claude Code |
| Sessions lost after reboot or crash | `tmux-resurrect` + `tmux-continuum` auto-save and restore |
| Switching between many cs sessions is slow | Mouse click on status bar, `C-a Tab` for last window, `C-a S` to browse |

`cs setup` will offer to copy this config to `~/.tmux.conf` automatically.

**Manual install:**

```bash
cp dotfiles/tmux.conf ~/.tmux.conf
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
tmux source ~/.tmux.conf   # inside tmux
# Press: C-a I             # install plugins
```

**Key shortcuts added:**

| Key | Action |
| --- | --- |
| `C-a a` | Open Claude in a popup (closes when done) |
| `C-a A` | Open Claude in a new persistent window |
| `C-a W` | Side-by-side: editor left, Claude right |
| `C-a \|` / `C-a -` | Vertical / horizontal split (opens in same dir) |
| `C-a h/j/k/l` | Move between panes (vim-style) |
| `C-a Tab` | Jump to last active window |
| `C-a g` | Floating shell popup |
| Right-click pane | Context menu: zoom / split / kill |

The config is self-contained with inline comments — feel free to adapt it.

---

## Contributing

After cloning, install the git hooks to get tests and lint on every commit:

```bash
make hooks
```

The pre-commit hook runs `go test -race ./...`, `golangci-lint`, and `markdownlint-cli2`
(via `npx`) on staged files.

`cs` is a personal tool that happens to be open-source. Issues and pull requests are welcome —
please open an issue first if you're planning a larger change so we can align before you invest
the time.

---

## Platform Support

| Platform | Status |
| --- | --- |
| macOS | Supported |
| Linux | Planned |
| Windows | Not supported (tmux unavailable) |

---

## License

MIT
