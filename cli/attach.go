package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/dhruv/cs/internal/tmux"
)

// AttachCommand returns the cs attach <name> subcommand.
func AttachCommand(client tmux.TmuxClient) *cli.Command {
	return &cli.Command{
		Name:      "attach",
		Usage:     "Attach to a named session non-interactively",
		ArgsUsage: "<name>",
		Description: `Attaches to the named cs session, bypassing the interactive picker.
Exits 1 if already inside tmux or if the session is not found.

Environment:
  CS_TMUX_SOCKET   Path to the cs tmux socket file (default: ~/.local/share/cs/cs.sock)`,
		Action: func(_ context.Context, cmd *cli.Command) error {
			if os.Getenv("TMUX") != "" {
				fmt.Fprintln(os.Stderr, "cs: already inside a tmux session — detach first (Ctrl-b d)")
				return cli.Exit("", 1)
			}
			name := cmd.Args().First()
			if name == "" {
				return cli.Exit("usage: cs attach <name>", 2)
			}
			socketPath := cmd.Root().String("socket")
			if err := client.AttachSession(socketPath, name); err != nil {
				fmt.Fprintf(os.Stderr, "cs attach: %v\n", err)
				return cli.Exit("", 1)
			}
			return nil
		},
	}
}
