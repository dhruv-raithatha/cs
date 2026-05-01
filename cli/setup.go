package cli

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dhruv/cs/internal/setup"
)

//go:embed assets/tmux.conf
var embeddedTmuxConf []byte

type setupDeps struct {
	check          func() []setup.DependencyStatus
	ensureDir      func() error
	pathCheck      func() (onPath bool, rcFile string)
	appendToRC     func(rcFile, line string) error
	tmuxConfExists func() bool
	copyTmuxConf   func() error
}

// SetupCommand returns the cs setup subcommand.
func SetupCommand() *cli.Command {
	return &cli.Command{
		Name:  "setup",
		Usage: "Check and optionally install required dependencies",
		Description: `Checks that tmux, fzf, and claude are installed and on PATH.
For each missing dependency, offers to install via Homebrew (or npm for claude).
Also ensures ~/.local/bin is on PATH and offers to copy the optimized tmux config.

Environment:
  CS_TMUX_SOCKET   Path to the cs tmux socket file (default: ~/.local/share/cs/cs.sock)`,
		Action: func(_ context.Context, _ *cli.Command) error {
			deps := setupDeps{
				check:          setup.Check,
				ensureDir:      setup.EnsureDataDir,
				pathCheck:      setup.CheckLocalBinOnPath,
				appendToRC:     setup.AppendToShellRC,
				tmuxConfExists: setup.TmuxConfExists,
				copyTmuxConf:   func() error { return setup.WriteTmuxConf(embeddedTmuxConf) },
			}
			return runSetup(deps, os.Stdin, os.Stdout)
		},
	}
}

func runSetup(deps setupDeps, stdin io.Reader, out io.Writer) error {
	fprint := func(format string, args ...any) {
		_, _ = fmt.Fprintf(out, format, args...)
	}
	scanner := bufio.NewScanner(stdin)
	confirm := func(prompt string) bool {
		_, _ = fmt.Fprint(out, prompt)
		if !scanner.Scan() {
			return false
		}
		ans := strings.TrimSpace(scanner.Text())
		return ans == "" || strings.EqualFold(ans, "y")
	}

	// ── Dependencies ────────────────────────────────────────────────────────
	fprint("Checking dependencies...\n")
	statuses := deps.check()
	for i := range statuses {
		d := &statuses[i]
		if d.Found {
			fprint("  ✓ %s %s\n", d.Name, versionShort(d.Version))
		} else {
			fprint("  ✗ %s — not found\n", d.Name)
			if d.InstallCmd != "" {
				if confirm("    Install with: " + d.InstallCmd + " [Y/n] ") {
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

	if err := deps.ensureDir(); err != nil {
		return fmt.Errorf("setup: %w", err)
	}

	// ── PATH ────────────────────────────────────────────────────────────────
	if deps.pathCheck != nil {
		fprint("\nChecking PATH...\n")
		onPath, rcFile := deps.pathCheck()
		if onPath {
			fprint("  ✓ ~/.local/bin is on PATH\n")
		} else {
			fprint("  ✗ ~/.local/bin not on PATH (cs install location)\n")
			if rcFile != "" {
				line := `export PATH="$HOME/.local/bin:$PATH"`
				if confirm(fmt.Sprintf("    Add to %s? [Y/n] ", rcFile)) {
					if err := deps.appendToRC(rcFile, line); err != nil {
						fprint("    Failed: %v\n", err)
					} else {
						fprint("  ✓ Added. Restart your shell or: source %s\n", rcFile)
					}
				}
			}
		}
	}

	// ── tmux config ─────────────────────────────────────────────────────────
	if deps.tmuxConfExists != nil {
		fprint("\nChecking tmux config...\n")
		if deps.tmuxConfExists() {
			fprint("  ✓ ~/.tmux.conf already exists\n")
		} else {
			fprint("  ✗ ~/.tmux.conf not found\n")
			if deps.copyTmuxConf != nil {
				if confirm("    Copy the optimized cs tmux config to ~/.tmux.conf? [Y/n] ") {
					if err := deps.copyTmuxConf(); err != nil {
						fprint("    Failed: %v\n", err)
					} else {
						fprint("  ✓ Copied. Install plugins inside tmux with: C-a I\n")
					}
				}
			}
		}
	}

	// ── Summary ─────────────────────────────────────────────────────────────
	allFound := true
	for _, d := range statuses {
		if !d.Found {
			allFound = false
		}
	}

	fprint("\n")
	if allFound {
		fprint("Setup complete. Run `cs` to start.\n")
		return nil
	}
	fprint("Some dependencies are still missing.\n")
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
