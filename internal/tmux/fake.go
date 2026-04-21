package tmux

import "github.com/dhruv/cs/internal/session"

// FakeTmuxClient is a configurable test double for TmuxClient.
type FakeTmuxClient struct {
	Sessions         []session.Session
	ListSessionsErr  error
	NewSessionErr    error
	AttachSessionErr error
	KillSessionErr   error
	HasSessionResult bool
	HasSessionErr    error

	AttachedSession string
	KilledSession   string
	CreatedSession  string
	CreatedModel    string
	CreatedEffort   string
}

func (f *FakeTmuxClient) ListSessions(_ string) ([]session.Session, error) {
	return f.Sessions, f.ListSessionsErr
}

func (f *FakeTmuxClient) NewSession(_, name, _, model, effort string) error {
	f.CreatedSession = name
	f.CreatedModel = model
	f.CreatedEffort = effort
	return f.NewSessionErr
}

func (f *FakeTmuxClient) AttachSession(_, name string) error {
	f.AttachedSession = name
	return f.AttachSessionErr
}

func (f *FakeTmuxClient) KillSession(_, name string) error {
	f.KilledSession = name
	return f.KillSessionErr
}

func (f *FakeTmuxClient) HasSession(_ string, _ string) (bool, error) {
	return f.HasSessionResult, f.HasSessionErr
}
