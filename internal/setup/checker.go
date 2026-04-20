// Package setup handles first-run dependency verification and environment preparation.
package setup

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DependencyStatus reports whether a required tool is available.
type DependencyStatus struct {
	Name       string
	Found      bool
	Version    string
	InstallCmd string
}

var installCmds = map[string]string{
	"tmux":   "brew install tmux",
	"fzf":    "brew install fzf",
	"claude": "npm i -g @anthropic-ai/claude-code",
}

var versionArgs = map[string][]string{
	"tmux":   {"-V"},
	"fzf":    {"--version"},
	"claude": {"--version"},
	"go":     {"version"},
}

// RequiredDeps is the ordered list of dependencies cs needs.
var RequiredDeps = []string{"tmux", "fzf", "claude"}

// checkDep looks up a single binary on PATH and captures its version.
func checkDep(name string) DependencyStatus {
	d := DependencyStatus{
		Name:       name,
		InstallCmd: installCmds[name],
	}

	if _, err := exec.LookPath(name); err != nil {
		return d
	}
	d.Found = true

	args := versionArgs[name]
	if args == nil {
		args = []string{"--version"}
	}
	out, err := exec.Command(name, args...).CombinedOutput()
	if err == nil {
		line := strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
		d.Version = line
	}
	return d
}

// Check verifies all required deps and returns their statuses.
func Check() []DependencyStatus {
	statuses := make([]DependencyStatus, len(RequiredDeps))
	for i, dep := range RequiredDeps {
		statuses[i] = checkDep(dep)
	}
	return statuses
}

// ensureDataDir creates the cs data directory if it does not exist.
func ensureDataDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create data dir %s: %w", path, err)
	}
	return nil
}

// EnsureDataDir creates ~/.local/share/cs/ if absent.
func EnsureDataDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	return ensureDataDir(home + "/.local/share/cs")
}
