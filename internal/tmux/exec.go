package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/dhruv/cs/internal/session"
)

type execTmuxClient struct{}

// NewExecTmuxClient returns a TmuxClient that shells out to tmux.
func NewExecTmuxClient() TmuxClient {
	return &execTmuxClient{}
}

// parseSessionLine parses one line from the 6-part list-sessions format.
// Format: name:workingDir:paneCommand:@cs-model:@cs-effort:session_created
func parseSessionLine(line string) (session.Session, bool) {
	parts := strings.SplitN(line, ":", 6)
	if len(parts) != 6 {
		return session.Session{}, false
	}
	createdAt, _ := strconv.ParseInt(parts[5], 10, 64)
	s := session.Session{
		Name:        parts[0],
		WorkingDir:  parts[1],
		PaneCommand: parts[2],
		Model:       parts[3],
		Effort:      parts[4],
		CreatedAt:   createdAt,
	}
	s.Status = deriveStatus(s.PaneCommand)
	return s, true
}

func (c *execTmuxClient) ListSessions(socketPath string) ([]session.Session, error) {
	out, err := runTmux(socketPath, "list-sessions", "-F",
		"#{session_name}:#{pane_current_path}:#{pane_current_command}:#{@cs-model}:#{@cs-effort}:#{session_created}")
	if err != nil {
		if strings.Contains(err.Error(), "no server running") || strings.Contains(out, "no server running") {
			return nil, nil
		}
		return nil, nil //nolint:nilerr // empty session list is not an error
	}
	var sessions []session.Session
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		s, ok := parseSessionLine(line)
		if !ok {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (c *execTmuxClient) NewSession(socketPath, name, workingDir, model, effort string) error {
	_, err := runTmux(socketPath, "new-session", "-d", "-s", name, "-c", workingDir,
		"-e", "ANTHROPIC_MODEL="+model,
		"-e", "CLAUDE_CODE_EFFORT_LEVEL="+effort,
		"claude")
	if err != nil {
		return fmt.Errorf("new-session %q: %w", name, err)
	}
	if _, err := runTmux(socketPath, "set-option", "-t", name, "@cs-model", model); err != nil {
		return fmt.Errorf("set-option @cs-model %q: %w", name, err)
	}
	if _, err := runTmux(socketPath, "set-option", "-t", name, "@cs-effort", effort); err != nil {
		return fmt.Errorf("set-option @cs-effort %q: %w", name, err)
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

// tmuxRunner is the function used to execute tmux subcommands.
// Replaced in unit tests to avoid requiring a real tmux binary.
var tmuxRunner = defaultTmuxRunner

func runTmux(socketPath string, args ...string) (string, error) {
	return tmuxRunner(socketPath, args...)
}

func defaultTmuxRunner(socketPath string, args ...string) (string, error) {
	base := []string{"-S", socketPath}
	cmd := exec.Command("tmux", append(base, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("tmux %s: %w: %s", args[0], err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// interactiveRunner is the function used to attach interactively to a tmux session.
// Replaced in unit tests to avoid requiring a real TTY.
var interactiveRunner = defaultInteractiveRunner

func runInteractive(socketPath, name string) error {
	return interactiveRunner(socketPath, name)
}

func defaultInteractiveRunner(socketPath, name string) error {
	cmd := exec.Command("tmux", "-S", socketPath, "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func deriveStatus(paneCommand string) session.SessionStatus {
	shells := map[string]bool{
		"zsh": true, "bash": true, "sh": true, "fish": true,
		"dash": true, "tcsh": true, "csh": true, "ksh": true,
	}
	if shells[paneCommand] || paneCommand == "" {
		return session.Dead
	}
	return session.Active
}
