package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhruv/cs/internal/setup"
)

// notifyNopDeps returns a setupDeps with all notification fields as harmless no-ops.
func notifyNopDeps() setupDeps {
	return setupDeps{
		check: func() []setup.DependencyStatus {
			return []setup.DependencyStatus{
				{Name: "tmux", Found: true, Version: "3.4"},
				{Name: "fzf", Found: true, Version: "0.71.0"},
				{Name: "claude", Found: true, Version: "1.0.0"},
			}
		},
		ensureDir:               func() error { return nil },
		pathCheck:               func() (bool, string) { return true, "" },
		tmuxConfStatus:          func() (setup.TmuxConfState, error) { return setup.TmuxConfIdentical, nil },
		appendCsBlock:           func([]byte) error { return nil },
		replaceTmuxConf:         func() error { return nil },
		ghosttyInstalled:        func() bool { return true },
		terminalNotifierInstalled: func() bool { return true },
		notifyInstalled:         func() bool { return false },
		installNotify:           func() error { return nil },
		testNotification:        func() error { return nil },
		registerHooks:           func() error { return nil },
		removeNotify:            func() error { return nil },
	}
}

// ── tmux.conf reconciliation step ────────────────────────────────────────────

func TestRunSetup_TmuxConf_Identical_NopSkip(t *testing.T) {
	deps := notifyNopDeps()
	deps.tmuxConfStatus = func() (setup.TmuxConfState, error) { return setup.TmuxConfIdentical, nil }
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader(""), &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "up to date")
}

func TestRunSetup_TmuxConf_Absent_OfferInstall(t *testing.T) {
	replaced := false
	deps := notifyNopDeps()
	deps.tmuxConfStatus = func() (setup.TmuxConfState, error) { return setup.TmuxConfAbsent, nil }
	deps.replaceTmuxConf = func() error { replaced = true; return nil }
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("y\n"), &out)
	require.NoError(t, err)
	assert.True(t, replaced)
}

func TestRunSetup_TmuxConf_Differs_ShowsDiffAndPrompts(t *testing.T) {
	deps := notifyNopDeps()
	deps.tmuxConfStatus = func() (setup.TmuxConfState, error) { return setup.TmuxConfDiffers, nil }
	deps.diff = func() string { return "--- old\n+++ new\n+allow-passthrough on\n" }

	var out bytes.Buffer
	// User chooses [s]kip
	err := runSetup(deps, strings.NewReader("s\n"), &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "allow-passthrough")
	assert.Contains(t, out.String(), "[a]ppend")
}

func TestRunSetup_TmuxConf_Differs_AppendChoice(t *testing.T) {
	appended := false
	deps := notifyNopDeps()
	deps.tmuxConfStatus = func() (setup.TmuxConfState, error) { return setup.TmuxConfDiffers, nil }
	deps.diff = func() string { return "+line\n" }
	deps.appendCsBlock = func([]byte) error { appended = true; return nil }

	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("a\n"), &out)
	require.NoError(t, err)
	assert.True(t, appended)
}

func TestRunSetup_TmuxConf_Differs_ReplaceChoice(t *testing.T) {
	replaced := false
	deps := notifyNopDeps()
	deps.tmuxConfStatus = func() (setup.TmuxConfState, error) { return setup.TmuxConfDiffers, nil }
	deps.diff = func() string { return "+line\n" }
	deps.replaceTmuxConf = func() error { replaced = true; return nil }

	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("r\n"), &out)
	require.NoError(t, err)
	assert.True(t, replaced)
}

// ── Ghostty recommendation step ───────────────────────────────────────────────

func TestRunSetup_Ghostty_Detected(t *testing.T) {
	deps := notifyNopDeps()
	deps.ghosttyInstalled = func() bool { return true }
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader(""), &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Ghostty detected")
}

func TestRunSetup_Ghostty_NotFound_ShowsInstallCmd(t *testing.T) {
	deps := notifyNopDeps()
	deps.ghosttyInstalled = func() bool { return false }
	var out bytes.Buffer
	// User presses Enter to continue without Ghostty
	err := runSetup(deps, strings.NewReader("\n"), &out)
	require.NoError(t, err)
	s := out.String()
	assert.Contains(t, s, "brew install --cask ghostty")
}

// ── Notification opt-in step ──────────────────────────────────────────────────

func TestRunSetup_Notify_UserDeclines(t *testing.T) {
	installed := false
	deps := notifyNopDeps()
	deps.notifyInstalled = func() bool { return false }
	deps.installNotify = func() error { installed = true; return nil }
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("n\n"), &out)
	require.NoError(t, err)
	assert.False(t, installed)
}

func TestRunSetup_Notify_UserAccepts_Installs(t *testing.T) {
	installed := false
	hooked := false
	tested := false
	deps := notifyNopDeps()
	deps.notifyInstalled = func() bool { return false }
	deps.terminalNotifierInstalled = func() bool { return true }
	deps.installNotify = func() error { installed = true; return nil }
	deps.registerHooks = func() error { hooked = true; return nil }
	deps.testNotification = func() error { tested = true; return nil }
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("y\n"), &out)
	require.NoError(t, err)
	assert.True(t, installed)
	assert.True(t, hooked)
	assert.True(t, tested)
}

func TestRunSetup_Notify_AlreadyInstalled_ShowsMenu(t *testing.T) {
	deps := notifyNopDeps()
	deps.notifyInstalled = func() bool { return true }
	var out bytes.Buffer
	// User presses [s]kip
	err := runSetup(deps, strings.NewReader("s\n"), &out)
	require.NoError(t, err)
	s := out.String()
	assert.Contains(t, s, "[t]")
	assert.Contains(t, s, "[u]")
	assert.Contains(t, s, "[r]")
}

func TestRunSetup_Notify_AlreadyInstalled_Remove(t *testing.T) {
	removed := false
	deps := notifyNopDeps()
	deps.notifyInstalled = func() bool { return true }
	deps.removeNotify = func() error { removed = true; return nil }
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("r\n"), &out)
	require.NoError(t, err)
	assert.True(t, removed)
}

func TestRunSetup_Notify_AlreadyInstalled_Test(t *testing.T) {
	tested := false
	deps := notifyNopDeps()
	deps.notifyInstalled = func() bool { return true }
	deps.testNotification = func() error { tested = true; return nil }
	var out bytes.Buffer
	err := runSetup(deps, strings.NewReader("t\n"), &out)
	require.NoError(t, err)
	assert.True(t, tested)
}
