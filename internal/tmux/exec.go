package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dhruv/cs/internal/session"
)

type execTmuxClient struct{}

// NewExecTmuxClient returns a TmuxClient that shells out to tmux.
func NewExecTmuxClient() TmuxClient {
	return &execTmuxClient{}
}

func (c *execTmuxClient) ListSessions(socketPath string) ([]session.Session, error) {
	// Format: name:workingDir:paneCommand
	out, err := runTmux(socketPath, "list-sessions", "-F", "#{session_name}:#{session_path}:#{pane_current_command}")
	if err != nil {
		// tmux exits non-zero when there are no sessions
		if strings.Contains(err.Error(), "no server running") || strings.Contains(out, "no server running") {
			return nil, nil
		}
		return nil, nil //nolint:nilerr // empty session list is not an error
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	var sessions []session.Session
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		s := session.Session{
			Name:        parts[0],
			WorkingDir:  parts[1],
			PaneCommand: parts[2],
		}
		s.Status = deriveStatus(s.PaneCommand)
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (c *execTmuxClient) NewSession(socketPath, name, workingDir string) error {
	_, err := runTmux(socketPath, "new-session", "-d", "-s", name, "-c", workingDir, "claude")
	if err != nil {
		return fmt.Errorf("new-session %q: %w", name, err)
	}
	return nil
}

func (c *execTmuxClient) AttachSession(socketPath, name string) error {
	if err := runInteractive(socketPath, name); err != nil {
		return fmt.Errorf("attach-session %q: %w", name, err)
	}
	return nil
}

func (c *execTmuxClient) KillSession(socketPath, name string) error {
	_, err := runTmux(socketPath, "kill-session", "-t", name)
	if err != nil {
		return fmt.Errorf("kill-session %q: %w", name, err)
	}
	return nil
}

func (c *execTmuxClient) HasSession(socketPath, name string) (bool, error) {
	_, err := runTmux(socketPath, "has-session", "-t", name)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func runTmux(socketPath string, args ...string) (string, error) {
	base := []string{"-S", socketPath}
	cmd := exec.Command("tmux", append(base, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("tmux %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func runInteractive(socketPath, name string) error {
	cmd := exec.Command("tmux", "-S", socketPath, "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func deriveStatus(paneCommand string) session.SessionStatus {
	shells := map[string]bool{
		"zsh": true, "bash": true, "sh": true, "fish": true, "dash": true,
	}
	if paneCommand == "claude" {
		return session.Active
	}
	if shells[paneCommand] || paneCommand == "" {
		return session.Dead
	}
	return session.Dead
}
