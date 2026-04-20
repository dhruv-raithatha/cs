package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/dhruv/cs/internal/session"
	"github.com/dhruv/cs/internal/tmux"
)

// ListCommand returns the cs list [--json] subcommand.
func ListCommand(client tmux.TmuxClient) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all cs-managed sessions",
		Description: `Lists all sessions on the cs tmux socket.
Default output is a human-readable table. Use --json for machine-readable output.

Environment:
  CS_TMUX_SOCKET   Path to the cs tmux socket file (default: ~/.local/share/cs/cs.sock)`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output newline-delimited JSON",
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			socketPath := cmd.Root().String("socket")
			mgr := session.NewManager(client)
			sessions, err := mgr.List(socketPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "cs list: %v\n", err)
				return cli.Exit("", 2)
			}
			if cmd.Bool("json") {
				return printJSON(sessions)
			}
			printTable(sessions)
			return nil
		},
	}
}

func printTable(sessions []session.Session) {
	fmt.Printf("%-20s %-40s %s\n", "NAME", "WORKING DIR", "STATUS")
	for _, s := range sessions {
		fmt.Printf("%-20s %-40s %s\n", s.Name, s.WorkingDir, s.Status)
	}
}

func printJSON(sessions []session.Session) error {
	for _, s := range sessions {
		obj := map[string]string{
			"name":        s.Name,
			"working_dir": s.WorkingDir,
			"status":      s.Status.String(),
		}
		b, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("json: %w", err)
		}
		fmt.Println(string(b))
	}
	return nil
}
