// Package tmux provides the TmuxClient interface and implementations for interacting with tmux.
package tmux

import "github.com/dhruv/cs/internal/session"

// TmuxClient defines the contract for tmux operations.
type TmuxClient interface {
	ListSessions(socketPath string) ([]session.Session, error)
	NewSession(socketPath, name, workingDir string) error
	AttachSession(socketPath, name string) error
	KillSession(socketPath, name string) error
	HasSession(socketPath, name string) (bool, error)
}
