// Package session defines the core Session type and related enums.
package session

// SessionStatus indicates whether the session's Claude process is running.
type SessionStatus int

const (
	// Active means the pane's current command is "claude".
	Active SessionStatus = iota
	// Dead means the pane's current command is a shell or empty.
	Dead
)

func (s SessionStatus) String() string {
	switch s {
	case Active:
		return "active"
	case Dead:
		return "dead"
	default:
		return "unknown"
	}
}

// Session represents a single tmux session managed by cs.
type Session struct {
	Name        string
	WorkingDir  string
	Status      SessionStatus
	PaneCommand string
}
