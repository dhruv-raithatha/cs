package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/dhruv/cs/internal/session"
	"github.com/dhruv/cs/internal/tmux"
)

const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiDim    = "\033[2m"
	maxPathLen = 38
)

// ListCommand returns the cs list [--json] [--all] subcommand.
func ListCommand(client tmux.TmuxClient) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all cs-managed sessions",
		Description: `Lists all sessions on the cs tmux socket.
Default output shows only active sessions. Use --all to include dead sessions.
Use --json for machine-readable output.

Environment:
  CS_TMUX_SOCKET   Path to the cs tmux socket file (default: ~/.local/share/cs/cs.sock)`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Output newline-delimited JSON",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Include dead sessions",
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
				return printJSON(os.Stdout, sessions)
			}
			printTable(os.Stdout, sessions, cmd.Bool("all"), isTTY())
			return nil
		},
	}
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func printTable(w io.Writer, sessions []session.Session, showAll bool, color bool) {
	var rows []session.Session
	for _, s := range sessions {
		if showAll || s.Status == session.Active {
			rows = append(rows, s)
		}
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	if showAll {
		fmt.Fprintln(tw, "  NAME\tWORKING DIR\tMODEL\tEFFORT\tAGE\tSTATUS")
	} else {
		fmt.Fprintln(tw, "NAME\tWORKING DIR\tMODEL\tEFFORT\tAGE")
	}

	for _, s := range rows {
		dir := abbreviatePath(s.WorkingDir)
		age := relativeAge(s.CreatedAt)

		if showAll {
			dot, prefix, suffix := statusGlyph(s.Status, color)
			fmt.Fprintf(tw, "%s %s%s\t%s\t%s\t%s\t%s\t%s%s\n",
				dot,
				prefix, s.Name,
				dir,
				s.Model,
				s.Effort,
				age,
				s.Status,
				suffix,
			)
		} else {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				s.Name,
				dir,
				s.Model,
				s.Effort,
				age,
			)
		}
	}
}

func statusGlyph(status session.SessionStatus, color bool) (dot, prefix, suffix string) {
	if !color {
		if status == session.Active {
			return "●", "", ""
		}
		return "○", "", ""
	}
	if status == session.Active {
		return ansiGreen + "●" + ansiReset, "", ""
	}
	return "○", ansiDim, ansiReset
}

func abbreviatePath(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && len(p) >= len(home) && p[:len(home)] == home {
		p = "~" + p[len(home):]
	}
	if len(p) > maxPathLen {
		p = "…" + p[len(p)-maxPathLen+1:]
	}
	return p
}

func relativeAge(createdAt int64) string {
	if createdAt == 0 {
		return ""
	}
	d := time.Since(time.Unix(createdAt, 0))
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dw", int(d.Hours()/(24*7)))
	}
}

func printJSON(w io.Writer, sessions []session.Session) error {
	for _, s := range sessions {
		obj := map[string]any{
			"name":        s.Name,
			"working_dir": s.WorkingDir,
			"model":       s.Model,
			"effort":      s.Effort,
			"status":      s.Status.String(),
			"created_at":  s.CreatedAt,
		}
		b, err := json.Marshal(obj)
		if err != nil {
			return fmt.Errorf("json: %w", err)
		}
		fmt.Fprintln(w, string(b))
	}
	return nil
}
