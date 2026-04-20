package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhruv/cs/internal/setup"
)

func makeChecker(statuses []setup.DependencyStatus) checkerFunc {
	return func() []setup.DependencyStatus { return statuses }
}

func nopEnsureDir() error { return nil }

func TestRunSetup_AllFound(t *testing.T) {
	check := makeChecker([]setup.DependencyStatus{
		{Name: "tmux", Found: true, Version: "tmux 3.4"},
		{Name: "fzf", Found: true, Version: "0.71.0"},
		{Name: "claude", Found: true, Version: "1.0.0"},
	})
	var out bytes.Buffer
	err := runSetup(check, nopEnsureDir, strings.NewReader(""), &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Setup complete")
	assert.Contains(t, out.String(), "✓ tmux")
}

func TestRunSetup_MissingDep_UserDeclines(t *testing.T) {
	check := makeChecker([]setup.DependencyStatus{
		{Name: "tmux", Found: false, InstallCmd: "brew install tmux"},
		{Name: "fzf", Found: true, Version: "0.71.0"},
		{Name: "claude", Found: true, Version: "1.0.0"},
	})
	var out bytes.Buffer
	// User declines install
	err := runSetup(check, nopEnsureDir, strings.NewReader("n\n"), &out)
	assert.Error(t, err)
	assert.Contains(t, out.String(), "✗ tmux")
	assert.Contains(t, out.String(), "still missing")
}

func TestRunSetup_MissingDep_NoInstallCmd(t *testing.T) {
	check := makeChecker([]setup.DependencyStatus{
		{Name: "tmux", Found: false, InstallCmd: ""},
	})
	var out bytes.Buffer
	err := runSetup(check, nopEnsureDir, strings.NewReader(""), &out)
	assert.Error(t, err)
}

func TestRunSetup_EnsureDirError(t *testing.T) {
	check := makeChecker([]setup.DependencyStatus{
		{Name: "tmux", Found: true, Version: "3.4"},
	})
	var out bytes.Buffer
	ensureErr := func() error { return assert.AnError }
	err := runSetup(check, ensureErr, strings.NewReader(""), &out)
	assert.Error(t, err)
}
