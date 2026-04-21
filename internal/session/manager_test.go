package session

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeClient is a minimal TmuxClient stub defined in this package for tests.
// We avoid importing internal/tmux to prevent circular deps; the interface is redefined here.
type fakeClient struct {
	sessions        []Session
	listErr         error
	newSessionErr   error
	attachErr       error
	killErr         error
	hasSessionValue bool
	hasSessionErr   error

	attachedSession string
	killedSession   string
	createdSession  string
	createdModel    string
	createdEffort   string
}

func (f *fakeClient) ListSessions(_ string) ([]Session, error) {
	return f.sessions, f.listErr
}
func (f *fakeClient) NewSession(_, name, _, model, effort string) error {
	f.createdSession = name
	f.createdModel = model
	f.createdEffort = effort
	return f.newSessionErr
}
func (f *fakeClient) AttachSession(_, name string) error {
	f.attachedSession = name
	return f.attachErr
}
func (f *fakeClient) KillSession(_, name string) error {
	f.killedSession = name
	return f.killErr
}
func (f *fakeClient) HasSession(_ string, _ string) (bool, error) {
	return f.hasSessionValue, f.hasSessionErr
}

func TestSessionManager_List_ActiveStatus(t *testing.T) {
	client := &fakeClient{
		sessions: []Session{
			{Name: "work", WorkingDir: "/tmp", PaneCommand: "claude"},
		},
	}
	m := NewManager(client)
	sessions, err := m.List("fake.sock")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, Active, sessions[0].Status)
}

func TestSessionManager_List_DeadStatus(t *testing.T) {
	client := &fakeClient{
		sessions: []Session{
			{Name: "old", WorkingDir: "/tmp", PaneCommand: "zsh"},
		},
	}
	m := NewManager(client)
	sessions, err := m.List("fake.sock")
	require.NoError(t, err)
	require.Len(t, sessions, 1)
	assert.Equal(t, Dead, sessions[0].Status)
}

func TestSessionManager_List_Empty(t *testing.T) {
	client := &fakeClient{}
	m := NewManager(client)
	sessions, err := m.List("fake.sock")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestSessionManager_List_Error(t *testing.T) {
	client := &fakeClient{listErr: errors.New("tmux unavailable")}
	m := NewManager(client)
	_, err := m.List("fake.sock")
	assert.Error(t, err)
}

func TestSessionManager_NewSession_EmptyName(t *testing.T) {
	client := &fakeClient{}
	m := NewManager(client)
	err := m.NewSession("fake.sock", "", "/tmp", "", "")
	assert.Error(t, err)
}

func TestSessionManager_NewSession_AlreadyExists(t *testing.T) {
	client := &fakeClient{hasSessionValue: true}
	m := NewManager(client)
	err := m.NewSession("fake.sock", "existing", "/tmp", "", "")
	require.NoError(t, err)
	assert.Equal(t, "existing", client.attachedSession)
	assert.Empty(t, client.createdSession)
}

func TestSessionManager_NewSession_Creates(t *testing.T) {
	client := &fakeClient{hasSessionValue: false}
	m := NewManager(client)
	err := m.NewSession("fake.sock", "new-session", "/tmp", "", "")
	require.NoError(t, err)
	assert.Equal(t, "new-session", client.createdSession)
	assert.Equal(t, "new-session", client.attachedSession)
}

// T006: Manager.NewSession forwards model and effort to the underlying client.
func TestSessionManager_NewSession_ForwardsModelEffort(t *testing.T) {
	tests := []struct {
		name   string
		model  string
		effort string
	}{
		{"opus and high", "opus", "high"},
		{"haiku and low", "haiku", "low"},
		{"empty strings", "", ""},
		{"model only", "sonnet", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &fakeClient{}
			m := NewManager(client)
			err := m.NewSession("fake.sock", "sess", "/tmp", tt.model, tt.effort)
			require.NoError(t, err)
			assert.Equal(t, tt.model, client.createdModel)
			assert.Equal(t, tt.effort, client.createdEffort)
		})
	}
}

func TestSessionManager_Kill_Success(t *testing.T) {
	client := &fakeClient{}
	m := NewManager(client)
	err := m.Kill("fake.sock", "to-kill")
	require.NoError(t, err)
	assert.Equal(t, "to-kill", client.killedSession)
}

func TestSessionManager_Kill_Error(t *testing.T) {
	client := &fakeClient{killErr: errors.New("not found")}
	m := NewManager(client)
	err := m.Kill("fake.sock", "ghost")
	assert.Error(t, err)
}
