# Quickstart: cs — Claude Session Manager

## Prerequisites

Before building or using `cs`, you need:
- Go 1.24+ (`go version`)
- `tmux` (`brew install tmux`)
- `fzf` (`brew install fzf`)
- `claude` CLI (`npm i -g @anthropic-ai/claude-code`)

Or just run `cs setup` after building — it checks and offers to install everything.

## Build

```sh
go build -o cs ./cmd/cs
```

For a release (statically linked):
```sh
CGO_ENABLED=0 go build -ldflags="-s -w" -o cs ./cmd/cs
```

Cross-compile checks:
```sh
GOOS=linux  GOARCH=amd64 go build ./cmd/cs
GOOS=darwin GOARCH=arm64 go build ./cmd/cs
```

## First-time Setup

```sh
./cs setup
```

This checks dependencies, offers to install missing ones, and creates `~/.local/share/cs/`.

## Install

Move the binary somewhere on your PATH:
```sh
mv cs /usr/local/bin/cs
```

## Usage

```sh
cs              # Open session picker — attach or create
cs list         # List all sessions (human-readable)
cs list --json  # List sessions as newline-delimited JSON
cs attach foo   # Attach to session named "foo" directly
cs delete foo   # Kill session named "foo"
cs version      # Print version
```

## Environment

| Variable | Default | Description |
|---|---|---|
| `CS_TMUX_SOCKET` | `~/.local/share/cs/cs.sock` | Override the tmux socket path |

## Tests

```sh
go test ./...                                              # all unit tests
go test -race ./...                                        # with race detector
go test -tags integration ./...                            # integration tests (requires tmux + fzf)
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```

## Lint

```sh
go vet ./...
golangci-lint run
```
