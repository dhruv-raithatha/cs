package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/dhruv/cs/internal/session"
	"github.com/dhruv/cs/internal/tmux"
)

// DeleteCommand returns the cs delete <name> subcommand.
func DeleteCommand(client tmux.TmuxClient) *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Kill a named session non-interactively",
		ArgsUsage: "<name>",
		Description: `Kills the named cs session without prompting for confirmation.
Use the interactive picker (cs) for a confirmation prompt before deleting.

Environment:
  CS_TMUX_SOCKET   Path to the cs tmux socket file (default: ~/.local/share/cs/cs.sock)`,
		Action: func(_ context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return cli.Exit("usage: cs delete <name>", 2)
			}
			socketPath := cmd.Root().String("socket")
			mgr := session.NewManager(client)
			if err := mgr.Kill(socketPath, name); err != nil {
				fmt.Fprintf(os.Stderr, "cs delete: %v\n", err)
				return cli.Exit("", 1)
			}
			fmt.Printf("Session '%s' deleted.\n", name)
			return nil
		},
	}
}
