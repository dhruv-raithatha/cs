package cli

import (
	"bufio"
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
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

// knownModels is the ordered list of Claude model aliases presented in the picker.
var knownModels = []string{"sonnet", "opus", "haiku", "sonnet[1m]", "opus[1m]"}

// knownEfforts is the ordered list of effort levels presented in the picker.
var knownEfforts = []string{"low", "medium", "high", "xhigh"}

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
		return createNewSession(socketPath, client, selector, stdin)
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
		model := cmp.Or(s.Model, "unknown")
		effort := cmp.Or(s.Effort, "unknown")
		line := fmt.Sprintf("%-20s %-28s %-12s %-7s", s.Name, s.WorkingDir, model, effort)
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
		return createNewSession(socketPath, client, selector, stdin)
	}

	// Parse session name (first whitespace-delimited token)
	name := strings.Fields(selected)[0]
	return client.AttachSession(socketPath, name)
}

// createNewSession prompts for a name, model, and effort, then creates the session.
func createNewSession(socketPath string, client tmux.TmuxClient, selector fzf.FuzzySelector, stdin io.Reader) error {
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
		model, err := pickModel(selector)
		if err != nil {
			return nil // user cancelled during model selection
		}
		effort, err := pickEffort(selector)
		if err != nil {
			return nil // user cancelled during effort selection
		}
		mgr := session.NewManager(client)
		return mgr.NewSession(socketPath, name, currentDir(), model, effort)
	}
}

// orderedWithDefault returns a copy of list with def moved to the front.
// If def is already first or not found, the original slice is returned unchanged.
func orderedWithDefault(list []string, def string) []string {
	idx := slices.Index(list, def)
	if idx <= 0 {
		return list
	}
	result := slices.Clone(list)
	result = slices.Delete(result, idx, idx+1)
	return slices.Insert(result, 0, def)
}

func pickModel(selector fzf.FuzzySelector) (string, error) {
	def := cmp.Or(os.Getenv("ANTHROPIC_MODEL"), "sonnet")
	items := orderedWithDefault(knownModels, def)
	selected, err := selector.Select(items, "Model: ", "enter: select model")
	if err != nil {
		return "", err
	}
	return cmp.Or(selected, def), nil
}

func pickEffort(selector fzf.FuzzySelector) (string, error) {
	def := cmp.Or(os.Getenv("CLAUDE_CODE_EFFORT_LEVEL"), "medium")
	items := orderedWithDefault(knownEfforts, def)
	selected, err := selector.Select(items, "Effort: ", "enter: select effort level")
	if err != nil {
		return "", err
	}
	return cmp.Or(selected, def), nil
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
