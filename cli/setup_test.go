package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhruv/cs/internal/setup"
)

func nopDeps(statuses []setup.DependencyStatus) setupDeps {
	return setupDeps{
		check:          func() []setup.DependencyStatus { return statuses },
		ensureDir:      func() error { return nil },
		pathCheck:      func() (bool, string) { return true, "" },
		tmuxConfExists: func() bool { return true },
	}
}

func TestRunSetup_AllFound(t *testing.T) {
	deps := nopDeps([]setup.DependencyStatus{
		{Name: "tmux", Found: true, Version: "tmux 3.4"},
		{Name: "fzf", Found: true, Version: "0.71.0"},
		{Name: "claude", Found: true, Version: "1.0.0"},
	})
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader(""), &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Setup complete")
	assert.Contains(t, out.String(), "✓ tmux")
}

func TestRunSetup_MissingDep_UserDeclines(t *testing.T) {
	deps := nopDeps([]setup.DependencyStatus{
		{Name: "tmux", Found: false, InstallCmd: "brew install tmux"},
		{Name: "fzf", Found: true, Version: "0.71.0"},
		{Name: "claude", Found: true, Version: "1.0.0"},
	})
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("n\n"), &out)
	assert.Error(t, err)
	assert.Contains(t, out.String(), "✗ tmux")
	assert.Contains(t, out.String(), "still missing")
}

func TestRunSetup_MissingDep_NoInstallCmd(t *testing.T) {
	deps := nopDeps([]setup.DependencyStatus{
		{Name: "tmux", Found: false, InstallCmd: ""},
	})
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader(""), &out)
	assert.Error(t, err)
}

func TestRunSetup_EnsureDirError(t *testing.T) {
	deps := nopDeps([]setup.DependencyStatus{
		{Name: "tmux", Found: true, Version: "3.4"},
	})
	deps.ensureDir = func() error { return assert.AnError }
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader(""), &out)
	assert.Error(t, err)
}

func TestRunSetup_PathNotFound_UserAccepts(t *testing.T) {
	var writtenRC, writtenLine string
	deps := nopDeps([]setup.DependencyStatus{
		{Name: "tmux", Found: true, Version: "3.4"},
	})
	deps.pathCheck = func() (bool, string) { return false, "/home/user/.zshrc" }
	deps.appendToRC = func(rc, line string) error {
		writtenRC = rc
		writtenLine = line
		return nil
	}
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("y\ny\n"), &out)
	require.NoError(t, err)
	assert.Equal(t, "/home/user/.zshrc", writtenRC)
	assert.Contains(t, writtenLine, ".local/bin")
}

func TestRunSetup_PathNotFound_UserDeclines(t *testing.T) {
	deps := nopDeps([]setup.DependencyStatus{
		{Name: "tmux", Found: true, Version: "3.4"},
	})
	deps.pathCheck = func() (bool, string) { return false, "/home/user/.zshrc" }
	deps.appendToRC = func(_, _ string) error {
		t.Fatal("should not write rc file when user declines")
		return nil
	}
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("n\n"), &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "not on PATH")
}

func TestRunSetup_TmuxConf_Offered(t *testing.T) {
	copied := false
	deps := nopDeps([]setup.DependencyStatus{
		{Name: "tmux", Found: true, Version: "3.4"},
	})
	deps.tmuxConfExists = func() bool { return false }
	deps.copyTmuxConf = func() error {
		copied = true
		return nil
	}
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("y\n"), &out)
	require.NoError(t, err)
	assert.True(t, copied)
	assert.Contains(t, out.String(), "Copied")
}

func TestRunSetup_TmuxConf_AlreadyExists(t *testing.T) {
	deps := nopDeps([]setup.DependencyStatus{
		{Name: "tmux", Found: true, Version: "3.4"},
	})
	deps.tmuxConfExists = func() bool { return true }
	deps.copyTmuxConf = func() error {
		t.Fatal("should not copy when ~/.tmux.conf already exists")
		return nil
	}
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader(""), &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "already exists")
}
