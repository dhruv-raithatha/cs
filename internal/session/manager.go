// Package session provides the SessionManager that orchestrates tmux session operations.
package session

import "fmt"

// Client is the subset of TmuxClient that SessionManager needs.
// The full interface is defined in internal/tmux to avoid import cycles.
type Client interface {
	ListSessions(socketPath string) ([]Session, error)
	NewSession(socketPath, name, workingDir, model, effort string) error
	AttachSession(socketPath, name string) error
	KillSession(socketPath, name string) error
	HasSession(socketPath, name string) (bool, error)
}

// Manager provides business logic over a tmux socket.
type Manager struct {
	client Client
}

// NewManager creates a Manager backed by the given Client.
func NewManager(client Client) *Manager {
	return &Manager{client: client}
}

// List returns all sessions on the given socket with computed status.
func (m *Manager) List(socketPath string) ([]Session, error) {
	sessions, err := m.client.ListSessions(socketPath)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	for i := range sessions {
		sessions[i].Status = deriveStatus(sessions[i].PaneCommand)
	}
	return sessions, nil
}

// NewSession creates a new session (or attaches if it already exists).
func (m *Manager) NewSession(socketPath, name, workingDir, model, effort string) error {
	if name == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	exists, err := m.client.HasSession(socketPath, name)
	if err != nil {
		return fmt.Errorf("has-session %q: %w", name, err)
	}
	if exists {
		return m.client.AttachSession(socketPath, name)
	}
	if err := m.client.NewSession(socketPath, name, workingDir, model, effort); err != nil {
		return err
	}
	return m.client.AttachSession(socketPath, name)
}

// Kill removes a session by name.
func (m *Manager) Kill(socketPath, name string) error {
	if err := m.client.KillSession(socketPath, name); err != nil {
		return fmt.Errorf("kill session %q: %w", name, err)
	}
	return nil
}

func deriveStatus(paneCommand string) SessionStatus {
	shells := map[string]bool{
		"zsh": true, "bash": true, "sh": true, "fish": true,
		"dash": true, "tcsh": true, "csh": true, "ksh": true,
	}
	if shells[paneCommand] || paneCommand == "" {
		return Dead
	}
	return Active
}
