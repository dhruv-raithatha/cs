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

//go:embed assets/notify.sh
var notifyShScript []byte

type setupDeps struct {
	check          func() []setup.DependencyStatus
	ensureDir      func() error
	pathCheck      func() (onPath bool, rcFile string)
	appendToRC     func(rcFile, line string) error
	// tmux.conf reconciliation
	tmuxConfExists func() bool // legacy: kept for backwards-compat with existing tests
	copyTmuxConf   func() error
	tmuxConfStatus func() (setup.TmuxConfState, error)
	diff           func() string
	appendCsBlock  func(block []byte) error
	replaceTmuxConf func() error
	// Ghostty recommendation
	ghosttyInstalled func() bool
	// Notification system
	terminalNotifierInstalled func() bool
	notifyInstalled           func() bool
	installNotify             func() error
	registerHooks             func() error
	testNotification          func() error
	removeNotify              func() error
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
			notifyScriptDest := notifyScriptInstallPath()
			deps := setupDeps{
				check:      setup.Check,
				ensureDir:  setup.EnsureDataDir,
				pathCheck:  setup.CheckLocalBinOnPath,
				appendToRC: setup.AppendToShellRC,
				// tmux.conf
				tmuxConfExists:  setup.TmuxConfExists,
				copyTmuxConf:    func() error { return setup.WriteTmuxConf(embeddedTmuxConf) },
				tmuxConfStatus:  func() (setup.TmuxConfState, error) { return setup.TmuxConfStatus(tmuxConfPath(), embeddedTmuxConf) },
				diff:            func() string { return setup.UnifiedDiff(readFileSilent(tmuxConfPath()), embeddedTmuxConf, 40) },
				appendCsBlock:   func(block []byte) error { return setup.AppendCsBlock(tmuxConfPath(), block) },
				replaceTmuxConf: func() error { return backupAndWrite(tmuxConfPath(), embeddedTmuxConf) },
				// Ghostty
				ghosttyInstalled: setup.GhosttyInstalled,
				// Notifications
				terminalNotifierInstalled: setup.TerminalNotifierInstalled,
				notifyInstalled:           func() bool { return isNotifyInstalled(notifyScriptDest) },
				installNotify: func() error {
					return installNotifyScript(notifyScriptDest, notifyShScript)
				},
				registerHooks: func() error { return registerNotifyHooks(notifyScriptDest) },
				testNotification: func() error {
					return fireTestNotification(notifyScriptDest)
				},
				removeNotify: func() error { return removeNotifyInstall(notifyScriptDest) },
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
	fprint("\nChecking tmux config...\n")
	if deps.tmuxConfStatus != nil {
		state, err := deps.tmuxConfStatus()
		if err != nil {
			fprint("  ⚠ Could not check tmux config: %v\n", err)
		} else {
			switch state {
			case setup.TmuxConfAbsent:
				fprint("  ✗ ~/.tmux.conf not found\n")
				if deps.replaceTmuxConf != nil {
					if confirm("    Install the cs tmux config to ~/.tmux.conf? [Y/n] ") {
						if err := deps.replaceTmuxConf(); err != nil {
							fprint("    Failed: %v\n", err)
						} else {
							fprint("  ✓ Installed. Load plugins inside tmux with: C-a I\n")
						}
					}
				}
			case setup.TmuxConfIdentical:
				fprint("  ✓ tmux config up to date\n")
			case setup.TmuxConfDiffers:
				fprint("  ~ ~/.tmux.conf has local changes. What cs would add:\n\n")
				if deps.diff != nil {
					fprint("%s\n\n", deps.diff())
				}
				fprint("  [a]ppend cs additions  [r]eplace (backs up to .tmux.conf.bk)  [s]kip\n")
				fprint("  Choice: ")
				if !scanner.Scan() {
					break
				}
				choice := strings.TrimSpace(strings.ToLower(scanner.Text()))
				switch choice {
				case "a":
					if deps.appendCsBlock != nil {
						if err := deps.appendCsBlock(csBlock()); err != nil {
							fprint("  Failed to append: %v\n", err)
						} else {
							fprint("  ✓ cs additions appended to ~/.tmux.conf\n")
						}
					}
				case "r":
					if deps.replaceTmuxConf != nil {
						if err := deps.replaceTmuxConf(); err != nil {
							fprint("  Failed to replace: %v\n", err)
						} else {
							fprint("  ✓ Replaced (backup at ~/.tmux.conf.bk)\n")
						}
					}
				default:
					fprint("  Skipped.\n")
				}
			}
		}
	} else if deps.tmuxConfExists != nil {
		// Legacy path for tests that only set tmuxConfExists.
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

	// ── Ghostty recommendation ───────────────────────────────────────────────
	if deps.ghosttyInstalled != nil {
		fprint("\nChecking terminal...\n")
		if deps.ghosttyInstalled() {
			fprint("  ✓ Ghostty detected — native notifications enabled\n")
		} else {
			fprint("  ~ Ghostty not found. cs delivers native notifications via Ghostty.\n")
			fprint("    To install: brew install --cask ghostty\n")
			fprint("    Press Enter to continue without Ghostty (notifications will be degraded): ")
			scanner.Scan()
		}
	}

	// ── Notification system ──────────────────────────────────────────────────
	if deps.notifyInstalled != nil {
		fprint("\nNotification system...\n")
		if deps.notifyInstalled() {
			fprint("  Notifications are installed.\n")
			fprint("  [t] Send test notification  [u] Update script  [r] Remove  [s] Skip\n")
			fprint("  Choice: ")
			if !scanner.Scan() {
				goto summary
			}
			switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
			case "t":
				if deps.testNotification != nil {
					if err := deps.testNotification(); err != nil {
						fprint("  Test notification failed: %v\n", err)
					} else {
						fprint("  ✓ Test notification sent\n")
					}
				}
			case "u":
				if deps.installNotify != nil {
					if err := deps.installNotify(); err != nil {
						fprint("  Update failed: %v\n", err)
					} else {
						fprint("  ✓ Notify script updated\n")
					}
				}
			case "r":
				if deps.removeNotify != nil {
					if err := deps.removeNotify(); err != nil {
						fprint("  Remove failed: %v\n", err)
					} else {
						fprint("  ✓ Notifications removed\n")
					}
				}
			default:
				fprint("  Skipped.\n")
			}
		} else {
			if confirm("  Set up interrupt-driven notifications? Claude sessions will alert you\n  when they need input. [Y/n] ") {
				// Check terminal-notifier
				if deps.terminalNotifierInstalled != nil && !deps.terminalNotifierInstalled() {
					fprint("  ✗ terminal-notifier not found\n")
					if confirm("    Install with: brew install terminal-notifier [Y/n] ") {
						if err := runInstallCmd("brew install terminal-notifier"); err != nil {
							fprint("    Install failed: %v\n", err)
						} else {
							fprint("  ✓ terminal-notifier installed\n")
						}
					}
				}
				// Install script
				if deps.installNotify != nil {
					if err := deps.installNotify(); err != nil {
						fprint("  ✗ Failed to install notify script: %v\n", err)
					} else {
						fprint("  ✓ Notify script installed\n")
					}
				}
				// Register hooks
				if deps.registerHooks != nil {
					if err := deps.registerHooks(); err != nil {
						fprint("  ✗ Failed to register hooks: %v\n", err)
					} else {
						fprint("  ✓ Claude Code hooks registered\n")
					}
				}
				// Test notification
				if deps.testNotification != nil {
					if err := deps.testNotification(); err != nil {
						fprint("  ⚠ Test notification failed (check System Settings > Notifications): %v\n", err)
					} else {
						fprint("  ✓ Test notification sent — allow it in macOS System Settings if prompted\n")
					}
				}
			}
		}
	}

summary:
	// ── Summary ─────────────────────────────────────────────────────────────
	allFound := true
	for _, d := range statuses {
		if !d.Found {
			allFound = false
		}
	}

	notifyStatus := "not installed"
	if deps.notifyInstalled != nil && deps.notifyInstalled() {
		notifyStatus = "installed ✓"
	}

	fprint("\n")
	if allFound {
		fprint("Setup complete. Run `cs` to start.\n")
		fprint("Notifications: %s\n", notifyStatus)
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
