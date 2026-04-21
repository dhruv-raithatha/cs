package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T002: Session.Model and Session.Effort are zero-value strings when not set.
func TestSession_ModelEffort_ZeroValue(t *testing.T) {
	s := Session{Name: "x", WorkingDir: "/tmp", PaneCommand: "claude"}
	assert.Equal(t, "", s.Model)
	assert.Equal(t, "", s.Effort)
}

func TestSessionStatus_String(t *testing.T) {
	assert.Equal(t, "active", Active.String())
	assert.Equal(t, "dead", Dead.String())
	assert.Equal(t, "unknown", SessionStatus(99).String())
}
