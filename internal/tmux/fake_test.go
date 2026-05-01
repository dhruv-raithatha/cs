package tmux

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhruv/cs/internal/session"
)

func TestFakeTmuxClient_ListSessions(t *testing.T) {
	want := []session.Session{{Name: "s1", WorkingDir: "/tmp"}}
	f := &FakeTmuxClient{Sessions: want}
	got, err := f.ListSessions("any.sock")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestFakeTmuxClient_ListSessions_Error(t *testing.T) {
	f := &FakeTmuxClient{ListSessionsErr: errors.New("boom")}
	_, err := f.ListSessions("any.sock")
	assert.Error(t, err)
}

func TestFakeTmuxClient_NewSession_CapturesFields(t *testing.T) {
	f := &FakeTmuxClient{}
	err := f.NewSession("sock", "my-session", "/home", "opus", "high")
	require.NoError(t, err)
	assert.Equal(t, "my-session", f.CreatedSession)
	assert.Equal(t, "opus", f.CreatedModel)
	assert.Equal(t, "high", f.CreatedEffort)
}

func TestFakeTmuxClient_NewSession_Error(t *testing.T) {
	f := &FakeTmuxClient{NewSessionErr: errors.New("fail")}
	err := f.NewSession("sock", "s", "/", "", "")
	assert.Error(t, err)
}

func TestFakeTmuxClient_AttachSession(t *testing.T) {
	f := &FakeTmuxClient{}
	err := f.AttachSession("sock", "my-session")
	require.NoError(t, err)
	assert.Equal(t, "my-session", f.AttachedSession)
}

func TestFakeTmuxClient_KillSession(t *testing.T) {
	f := &FakeTmuxClient{}
	err := f.KillSession("sock", "target")
	require.NoError(t, err)
	assert.Equal(t, "target", f.KilledSession)
}

func TestFakeTmuxClient_HasSession(t *testing.T) {
	f := &FakeTmuxClient{HasSessionResult: true}
	found, err := f.HasSession("sock", "any")
	require.NoError(t, err)
	assert.True(t, found)

	f2 := &FakeTmuxClient{HasSessionResult: false}
	found, err = f2.HasSession("sock", "any")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestDeriveStatus(t *testing.T) {
	assert.Equal(t, session.Active, deriveStatus("claude"))
	assert.Equal(t, session.Active, deriveStatus("2.1.126")) // versioned claude binary
	assert.Equal(t, session.Dead, deriveStatus("zsh"))
	assert.Equal(t, session.Dead, deriveStatus("bash"))
	assert.Equal(t, session.Dead, deriveStatus(""))
	assert.Equal(t, session.Active, deriveStatus("vim")) // running something — not dead
}
