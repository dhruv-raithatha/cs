package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dhruv/cs/internal/setup"
)

// SetupCommand returns the cs setup subcommand.
func SetupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Check and optionally install required dependencies",
		Description: `Checks that tmux, fzf, and claude are installed and on PATH.
For each missing dependency, offers to install via Homebrew (or npm for claude).
Also creates ~/.local/share/cs/ if it does not exist.

Environment:
  CS_TMUX_SOCKET   Path to the cs tmux socket file (default: ~/.local/share/cs/cs.sock)`,
		Action: func(_ context.Context, _ *cli.Command) error {
			return runSetup(setup.Check, setup.EnsureDataDir, os.Stdin, os.Stdout)
		},
	}
}

type checkerFunc func() []setup.DependencyStatus
type ensureDirFunc func() error

func runSetup(check checkerFunc, ensureDir ensureDirFunc, stdin io.Reader, out io.Writer) error {
	fprint := func(format string, args ...any) {
		_, _ = fmt.Fprintf(out, format, args...)
	}

	fprint("Checking dependencies...\n")

	statuses := check()
	for i := range statuses {
		d := &statuses[i]
		if d.Found {
			fprint("  ✓ %s %s\n", d.Name, versionShort(d.Version))
		} else {
			fprint("  ✗ %s — not found\n", d.Name)
			if d.InstallCmd != "" {
				fprint("    Install with: %s [Y/n] ", d.InstallCmd)
				scanner := bufio.NewScanner(stdin)
				if scanner.Scan() {
					answer := strings.TrimSpace(scanner.Text())
					if answer == "" || strings.EqualFold(answer, "y") {
						if err := runInstallCmd(d.InstallCmd); err != nil {
							fprint("    Install failed: %v\n", err)
						} else {
							d.Found = true
							fprint("  ✓ %s installed\n", d.Name)
						}
					}
				}
			}
		}
	}

	if err := ensureDir(); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	allFound := true
	for _, d := range statuses {
		if !d.Found {
			allFound = false
		}
	}

	if allFound {
		fprint("\nSetup complete. Run `cs` to start.\n")
		return nil
	}
	fprint("\nSome dependencies are still missing.\n")
	return cli.Exit("", 1)
}

func versionShort(v string) string {
	if v == "" {
		return ""
	}
	parts := strings.Fields(v)
	for _, p := range parts {
		if len(p) > 0 && (p[0] >= '0' && p[0] <= '9' || p[0] == 'v') {
			return p
		}
	}
	return parts[len(parts)-1]
}

func runInstallCmd(cmd string) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}
	c := exec.Command(parts[0], parts[1:]...) //nolint:gosec
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
