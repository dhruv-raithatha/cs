package fzf

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type execFuzzySelector struct{}

// NewExecFuzzySelector returns a FuzzySelector that shells out to fzf.
func NewExecFuzzySelector() FuzzySelector {
	return &execFuzzySelector{}
}

// Select presents items in fzf. If the user presses ctrl-d on an item,
// the returned string is prefixed with "__delete__:" so callers can detect
// the delete intent without changing the interface.
func (s *execFuzzySelector) Select(items []string, prompt, header string) (string, error) {
	args := []string{
		"--height", "40%",
		"--no-sort",
		"--prompt", prompt,
		"--expect", "ctrl-d",
	}
	if header != "" {
		args = append(args, "--header", header)
	}

	var input string
	if len(items) > 0 {
		input = strings.Join(items, "\n")
	}

	cmd := exec.Command("fzf", args...)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("fzf: %w", err)
	}

	// With --expect, fzf outputs two lines: key (may be empty) + selected item
	lines := strings.SplitN(out.String(), "\n", 2)
	if len(lines) < 2 {
		return strings.TrimRight(out.String(), "\n"), nil
	}

	key := strings.TrimRight(lines[0], "\r\n")
	selected := strings.TrimRight(lines[1], "\r\n")

	if key == "ctrl-d" {
		return "__delete__:" + selected, nil
	}
	return selected, nil
}
