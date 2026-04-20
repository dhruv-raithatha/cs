package session

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSessionStatus_String(t *testing.T) {
	assert.Equal(t, "active", Active.String())
	assert.Equal(t, "dead", Dead.String())
	assert.Equal(t, "unknown", SessionStatus(99).String())
}
