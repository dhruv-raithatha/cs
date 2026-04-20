package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/dhruv/cs/internal/fzf"
	"github.com/dhruv/cs/internal/session"
	"github.com/dhruv/cs/internal/tmux"
)

const (
	newSessionEntry = "[ + new session ]"
	deletePrefix    = "__delete__:"
)

// RootAction is the default command — interactive session picker.
func RootAction(client tmux.TmuxClient, selector fzf.FuzzySelector) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		socketPath := cmd.String("socket")
		return runWithConfirm(socketPath, client, selector, confirmFromStdin)
	}
}

func runWithConfirm(
	socketPath string,
	client tmux.TmuxClient,
	selector fzf.FuzzySelector,
	confirm func(msg string) bool,
) error {
	return runWithConfirmReader(socketPath, client, selector, confirm, os.Stdin)
}

func runWithConfirmReader(
	socketPath string,
	client tmux.TmuxClient,
	selector fzf.FuzzySelector,
	confirm func(msg string) bool,
	stdin io.Reader,
) error {
	if os.Getenv("TMUX") != "" {
		return fmt.Errorf("cs: already inside a tmux session — detach first (Ctrl-b d)")
	}

	mgr := session.NewManager(client)
	sessions, err := mgr.List(socketPath)
	if err != nil {
		return fmt.Errorf("cs: %w", err)
	}

	if len(sessions) == 0 {
		return createNewSession(socketPath, client, stdin)
	}

	return runPicker(socketPath, client, selector, sessions, confirm, stdin)
}

func runPicker(
	socketPath string,
	client tmux.TmuxClient,
	selector fzf.FuzzySelector,
	sessions []session.Session,
	confirm func(msg string) bool,
	stdin io.Reader,
) error {
	items := make([]string, 0, len(sessions)+1)
	items = append(items, newSessionEntry)
	for _, s := range sessions {
		line := fmt.Sprintf("%-20s %-40s", s.Name, s.WorkingDir)
		if s.Status == session.Dead {
			line += " [dead]"
		}
		items = append(items, strings.TrimRight(line, " "))
	}

	header := "ctrl-d: delete   enter: attach   esc: quit"
	selected, err := selector.Select(items, "> ", header)
	if err != nil {
		// User cancelled (fzf exit 130) — not an error
		return nil
	}
	if selected == "" {
		return nil
	}

	if strings.HasPrefix(selected, deletePrefix) {
		return handleDelete(socketPath, client, selected, confirm)
	}

	if selected == newSessionEntry {
		return createNewSession(socketPath, client, stdin)
	}

	// Parse session name (first whitespace-delimited token)
	name := strings.Fields(selected)[0]
	return client.AttachSession(socketPath, name)
}

func createNewSession(socketPath string, client tmux.TmuxClient, stdin io.Reader) error {
	scanner := bufio.NewScanner(stdin)
	for {
		fmt.Fprint(os.Stderr, "New session name: ")
		if !scanner.Scan() {
			return nil // EOF or Ctrl-d — cancelled
		}
		name := strings.TrimSpace(scanner.Text())
		if name == "" {
			fmt.Fprintln(os.Stderr, "Session name cannot be empty.")
			continue
		}
		mgr := session.NewManager(client)
		return mgr.NewSession(socketPath, name, currentDir())
	}
}

func handleDelete(socketPath string, client tmux.TmuxClient, selected string, confirm func(string) bool) error {
	raw := strings.TrimPrefix(selected, deletePrefix)
	name := strings.Fields(raw)[0]

	if !confirm(fmt.Sprintf("Delete session '%s'? [y/N]: ", name)) {
		return nil
	}
	mgr := session.NewManager(client)
	return mgr.Kill(socketPath, name)
}

func confirmFromStdin(msg string) bool {
	fmt.Fprint(os.Stderr, msg)
	var answer string
	_, _ = fmt.Scanln(&answer)
	return strings.EqualFold(strings.TrimSpace(answer), "y")
}

func currentDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return os.Getenv("HOME")
	}
	return dir
}
