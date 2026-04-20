package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhruv/cs/internal/fzf"
	"github.com/dhruv/cs/internal/session"
	"github.com/dhruv/cs/internal/tmux"
)

const testSocket = "test.sock"

// runTest drives runWithConfirmReader with auto-confirm and the provided stdin text.
func runTest(t *testing.T, client *tmux.FakeTmuxClient, selector *fzf.FakeFuzzySelector, stdinText string) error {
	t.Helper()
	return runWithConfirmReader(testSocket, client, selector, func(_ string) bool { return true }, strings.NewReader(stdinText))
}

func runTestWithConfirm(t *testing.T, client *tmux.FakeTmuxClient, selector *fzf.FakeFuzzySelector, stdinText string, confirm func(string) bool) error {
	t.Helper()
	return runWithConfirmReader(testSocket, client, selector, confirm, strings.NewReader(stdinText))
}

func TestRun_AlreadyInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
	client := &tmux.FakeTmuxClient{}
	selector := &fzf.FakeFuzzySelector{}
	err := runTest(t, client, selector, "")
	assert.Error(t, err)
	assert.Empty(t, client.AttachedSession)
}

func TestRun_NoSessions_CreatesNew(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{}
	selector := &fzf.FakeFuzzySelector{}
	err := runTest(t, client, selector, "my-project\n")
	require.NoError(t, err)
	assert.Equal(t, "my-project", client.CreatedSession)
	assert.Equal(t, "my-project", client.AttachedSession)
}

func TestRun_NoSessions_EmptyNameThenValid(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{}
	selector := &fzf.FakeFuzzySelector{}
	// First line empty (re-prompt), second line valid
	err := runTest(t, client, selector, "\nmy-project\n")
	require.NoError(t, err)
	assert.Equal(t, "my-project", client.CreatedSession)
}

func TestRun_NoSessions_CancelWithEOF(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{}
	selector := &fzf.FakeFuzzySelector{}
	// Empty reader simulates Ctrl-d / EOF
	err := runTest(t, client, selector, "")
	assert.NoError(t, err)
	assert.Empty(t, client.CreatedSession)
}

func TestRun_HasSessions_AttachExisting(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "work", WorkingDir: "/tmp", PaneCommand: "claude"},
		},
	}
	selector := &fzf.FakeFuzzySelector{
		Selections: []string{"work                 /tmp                                    "},
	}
	err := runTest(t, client, selector, "")
	require.NoError(t, err)
	assert.Equal(t, "work", client.AttachedSession)
}

func TestRun_HasSessions_SelectNew(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "old", WorkingDir: "/tmp", PaneCommand: "zsh"},
		},
	}
	// User picks [ + new session ] in fzf, then types name via stdin
	selector := &fzf.FakeFuzzySelector{
		Selections: []string{newSessionEntry},
	}
	err := runTest(t, client, selector, "fresh\n")
	require.NoError(t, err)
	assert.Equal(t, "fresh", client.CreatedSession)
}

func TestRun_FzfCancel(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "work", WorkingDir: "/tmp", PaneCommand: "claude"},
		},
	}
	selector := &fzf.FakeFuzzySelector{
		Err: errors.New("fzf: exit status 130"),
	}
	err := runTest(t, client, selector, "")
	assert.NoError(t, err)
}

func TestRun_DeleteConfirm(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "stale", WorkingDir: "/tmp", PaneCommand: "zsh"},
		},
	}
	selector := &fzf.FakeFuzzySelector{
		Selections: []string{deletePrefix + "stale   /tmp   [dead]"},
	}
	err := runTestWithConfirm(t, client, selector, "", func(_ string) bool { return true })
	require.NoError(t, err)
	assert.Equal(t, "stale", client.KilledSession)
}

func TestRun_DeleteCancel(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "stale", WorkingDir: "/tmp", PaneCommand: "zsh"},
		},
	}
	selector := &fzf.FakeFuzzySelector{
		Selections: []string{deletePrefix + "stale   /tmp   [dead]"},
	}
	err := runTestWithConfirm(t, client, selector, "", func(_ string) bool { return false })
	require.NoError(t, err)
	assert.Empty(t, client.KilledSession)
}
