package tmux

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhruv/cs/internal/session"
)

// withFakeRunner replaces tmuxRunner for the duration of the test.
func withFakeRunner(t *testing.T, fn func(socketPath string, args ...string) (string, error)) {
	t.Helper()
	orig := tmuxRunner
	tmuxRunner = fn
	t.Cleanup(func() { tmuxRunner = orig })
}

func TestExecTmuxClient_ListSessions_ParsesOutput(t *testing.T) {
	withFakeRunner(t, func(_ string, args ...string) (string, error) {
		if args[0] == "list-sessions" {
			return "work:/home/dev:claude:opus:high:1776876938\nold:/tmp:zsh:::1776800000\n", nil
		}
		return "", nil
	})

	c := NewExecTmuxClient()
	sessions, err := c.ListSessions("fake.sock")
	require.NoError(t, err)
	require.Len(t, sessions, 2)

	assert.Equal(t, "work", sessions[0].Name)
	assert.Equal(t, "opus", sessions[0].Model)
	assert.Equal(t, "high", sessions[0].Effort)
	assert.Equal(t, session.Active, sessions[0].Status)

	assert.Equal(t, "old", sessions[1].Name)
	assert.Equal(t, "", sessions[1].Model)
	assert.Equal(t, "", sessions[1].Effort)
	assert.Equal(t, session.Dead, sessions[1].Status)
}

func TestExecTmuxClient_ListSessions_SkipsInvalidAndEmptyLines(t *testing.T) {
	withFakeRunner(t, func(_ string, args ...string) (string, error) {
		if args[0] == "list-sessions" {
			// Include an empty line and a malformed line (old 5-part format)
			return "good:/tmp:claude:sonnet:low:1776876938\n\nbad-line\n", nil
		}
		return "", nil
	})

	c := NewExecTmuxClient()
	sessions, err := c.ListSessions("fake.sock")
	require.NoError(t, err)
	require.Len(t, sessions, 1) // only "good" is valid
	assert.Equal(t, "good", sessions[0].Name)
}

func TestExecTmuxClient_ListSessions_NoServer(t *testing.T) {
	withFakeRunner(t, func(_ string, args ...string) (string, error) {
		return "no server running", errors.New("tmux list-sessions: exit status 1: no server running")
	})

	c := NewExecTmuxClient()
	sessions, err := c.ListSessions("fake.sock")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestExecTmuxClient_NewSession_BuildsCommand(t *testing.T) {
	var capturedCalls [][]string
	withFakeRunner(t, func(_ string, args ...string) (string, error) {
		capturedCalls = append(capturedCalls, args)
		return "", nil
	})

	c := NewExecTmuxClient()
	err := c.NewSession("fake.sock", "my-session", "/home", "opus", "high")
	require.NoError(t, err)

	// Expect 3 calls: new-session, set-option @cs-model, set-option @cs-effort
	require.Len(t, capturedCalls, 3)

	newSessionArgs := strings.Join(capturedCalls[0], " ")
	assert.Contains(t, newSessionArgs, "new-session")
	assert.Contains(t, newSessionArgs, "-e")
	assert.Contains(t, newSessionArgs, "ANTHROPIC_MODEL=opus")
	assert.Contains(t, newSessionArgs, "--effort high")
	assert.NotContains(t, newSessionArgs, "CLAUDE_CODE_EFFORT_LEVEL")

	assert.Equal(t, []string{"set-option", "-t", "my-session", "@cs-model", "opus"}, capturedCalls[1])
	assert.Equal(t, []string{"set-option", "-t", "my-session", "@cs-effort", "high"}, capturedCalls[2])
}

func TestExecTmuxClient_NewSession_Error(t *testing.T) {
	withFakeRunner(t, func(_ string, args ...string) (string, error) {
		return "", errors.New("tmux error")
	})

	c := NewExecTmuxClient()
	err := c.NewSession("fake.sock", "sess", "/tmp", "sonnet", "medium")
	assert.Error(t, err)
}

func TestExecTmuxClient_KillSession(t *testing.T) {
	var capturedArgs []string
	withFakeRunner(t, func(_ string, args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	})

	c := NewExecTmuxClient()
	err := c.KillSession("fake.sock", "target")
	require.NoError(t, err)
	assert.Equal(t, []string{"kill-session", "-t", "target"}, capturedArgs)
}

func TestExecTmuxClient_HasSession_Found(t *testing.T) {
	withFakeRunner(t, func(_ string, _ ...string) (string, error) {
		return "", nil // no error means session exists
	})

	c := NewExecTmuxClient()
	found, err := c.HasSession("fake.sock", "exists")
	require.NoError(t, err)
	assert.True(t, found)
}

func TestExecTmuxClient_HasSession_NotFound(t *testing.T) {
	withFakeRunner(t, func(_ string, _ ...string) (string, error) {
		return "", errors.New("no session") // error means not found
	})

	c := NewExecTmuxClient()
	found, err := c.HasSession("fake.sock", "missing")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestExecTmuxClient_ListSessions_OtherError(t *testing.T) {
	withFakeRunner(t, func(_ string, _ ...string) (string, error) {
		return "some other error", errors.New("tmux: unexpected failure")
	})

	c := NewExecTmuxClient()
	sessions, err := c.ListSessions("fake.sock")
	require.NoError(t, err) // other errors silently return empty
	assert.Empty(t, sessions)
}

func TestExecTmuxClient_NewSession_SetModelOptionError(t *testing.T) {
	call := 0
	withFakeRunner(t, func(_ string, args ...string) (string, error) {
		call++
		if call == 1 {
			return "", nil // new-session succeeds
		}
		return "", errors.New("set-option failed") // first set-option fails
	})

	c := NewExecTmuxClient()
	err := c.NewSession("fake.sock", "sess", "/tmp", "opus", "high")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "@cs-model")
}

func TestExecTmuxClient_NewSession_SetEffortOptionError(t *testing.T) {
	call := 0
	withFakeRunner(t, func(_ string, args ...string) (string, error) {
		call++
		if call <= 2 {
			return "", nil // new-session and @cs-model succeed
		}
		return "", errors.New("set-option failed") // @cs-effort fails
	})

	c := NewExecTmuxClient()
	err := c.NewSession("fake.sock", "sess", "/tmp", "opus", "high")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "@cs-effort")
}

func TestExecTmuxClient_KillSession_Error(t *testing.T) {
	withFakeRunner(t, func(_ string, _ ...string) (string, error) {
		return "", errors.New("kill failed")
	})

	c := NewExecTmuxClient()
	err := c.KillSession("fake.sock", "target")
	assert.Error(t, err)
}

func withFakeInteractive(t *testing.T, fn func(socketPath, name string) error) {
	t.Helper()
	orig := interactiveRunner
	interactiveRunner = fn
	t.Cleanup(func() { interactiveRunner = orig })
}

func TestExecTmuxClient_AttachSession_Success(t *testing.T) {
	var attachedSocket, attachedName string
	withFakeInteractive(t, func(socketPath, name string) error {
		attachedSocket = socketPath
		attachedName = name
		return nil
	})

	c := NewExecTmuxClient()
	err := c.AttachSession("my.sock", "work")
	require.NoError(t, err)
	assert.Equal(t, "my.sock", attachedSocket)
	assert.Equal(t, "work", attachedName)
}

func TestExecTmuxClient_AttachSession_Error(t *testing.T) {
	withFakeInteractive(t, func(_, _ string) error {
		return errors.New("no TTY")
	})

	c := NewExecTmuxClient()
	err := c.AttachSession("my.sock", "work")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attach-session")
}
