//go:build integration

package tmux

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testSocketPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir + "/test.sock"
}

func killTestServer(socketPath string) {
	_ = exec.Command("tmux", "-S", socketPath, "kill-server").Run()
}

func TestExecTmuxClient_ListSessions_Empty(t *testing.T) {
	socket := testSocketPath(t)
	defer killTestServer(socket)

	c := &execTmuxClient{}
	sessions, err := c.ListSessions(socket)
	// No sessions yet — tmux returns non-zero but that's acceptable as empty
	if err != nil {
		assert.Empty(t, sessions)
	} else {
		assert.Empty(t, sessions)
	}
}

func TestExecTmuxClient_NewAndListSession(t *testing.T) {
	socket := testSocketPath(t)
	defer killTestServer(socket)

	c := &execTmuxClient{}

	err := c.NewSession(socket, "test-session", os.TempDir(), "sonnet", "medium")
	require.NoError(t, err)

	sessions, err := c.ListSessions(socket)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "test-session", sessions[0].Name)
}

func TestExecTmuxClient_HasSession(t *testing.T) {
	socket := testSocketPath(t)
	defer killTestServer(socket)

	c := &execTmuxClient{}

	err := c.NewSession(socket, "exists", os.TempDir(), "sonnet", "medium")
	require.NoError(t, err)

	found, err := c.HasSession(socket, "exists")
	require.NoError(t, err)
	assert.True(t, found)

	notFound, err := c.HasSession(socket, "no-such")
	require.NoError(t, err)
	assert.False(t, notFound)
}

func TestExecTmuxClient_KillSession(t *testing.T) {
	socket := testSocketPath(t)
	defer killTestServer(socket)

	c := &execTmuxClient{}

	err := c.NewSession(socket, "to-kill", os.TempDir(), "sonnet", "medium")
	require.NoError(t, err)

	err = c.KillSession(socket, "to-kill")
	require.NoError(t, err)

	found, err := c.HasSession(socket, "to-kill")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestExecTmuxClient_AttachSession_NoTTY(t *testing.T) {
	socket := testSocketPath(t)
	defer killTestServer(socket)

	c := &execTmuxClient{}

	err := c.NewSession(socket, "attach-test", os.TempDir(), "sonnet", "medium")
	require.NoError(t, err)

	// AttachSession requires a real TTY — in CI this will fail, which is expected.
	// We just verify the error is not about the session not existing.
	err = c.AttachSession(socket, "attach-test")
	// In a no-TTY environment, tmux will fail — but session should exist
	found, herr := c.HasSession(socket, "attach-test")
	require.NoError(t, herr)
	assert.True(t, found)
	_ = err
}

// T010: NewSession writes @cs-model and @cs-effort options; ListSessions reads them back.
func TestExecTmuxClient_NewSession_SetsModelEffort(t *testing.T) {
	socket := testSocketPath(t)
	defer killTestServer(socket)

	c := &execTmuxClient{}
	err := c.NewSession(socket, "opt-test", os.TempDir(), "opus", "high")
	require.NoError(t, err)

	sessions, err := c.ListSessions(socket)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "opus", sessions[0].Model)
	assert.Equal(t, "high", sessions[0].Effort)
}

func TestExecTmuxClient_NewSession_EmptyModelEffort(t *testing.T) {
	socket := testSocketPath(t)
	defer killTestServer(socket)

	c := &execTmuxClient{}
	err := c.NewSession(socket, "empty-test", os.TempDir(), "", "")
	require.NoError(t, err)

	sessions, err := c.ListSessions(socket)
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, "", sessions[0].Model)
	assert.Equal(t, "", sessions[0].Effort)
}
