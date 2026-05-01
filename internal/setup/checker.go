// Package setup handles first-run dependency verification and environment preparation.
package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

// LocalBinPath returns the path to ~/.local/bin.
func LocalBinPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "bin")
}

// CheckLocalBinOnPath returns whether ~/.local/bin is in PATH and, if not,
// the detected shell rc file to update.
func CheckLocalBinOnPath() (onPath bool, rcFile string) {
	localBin := LocalBinPath()
	if slices.Contains(filepath.SplitList(os.Getenv("PATH")), localBin) {
		return true, ""
	}
	home, _ := os.UserHomeDir()
	rc := filepath.Join(home, ".bashrc")
	if strings.HasSuffix(os.Getenv("SHELL"), "zsh") {
		rc = filepath.Join(home, ".zshrc")
	}
	return false, rc
}

// AppendToShellRC appends a line to the given shell rc file.
func AppendToShellRC(rcFile, line string) error {
	f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open %s: %w", rcFile, err)
	}
	defer func() { _ = f.Close() }()
	_, err = fmt.Fprintf(f, "\n%s\n", line)
	return err
}

// TmuxConfExists reports whether ~/.tmux.conf already exists.
func TmuxConfExists() bool {
	home, _ := os.UserHomeDir()
	_, err := os.Stat(filepath.Join(home, ".tmux.conf"))
	return err == nil
}

// WriteTmuxConf writes content to ~/.tmux.conf.
func WriteTmuxConf(content []byte) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dest := filepath.Join(home, ".tmux.conf")
	return os.WriteFile(dest, content, 0o644)
}
