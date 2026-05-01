package tmux

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dhruv/cs/internal/session"
)

// T008: Table-driven tests for the 6-part ListSessions parsing helper.
func TestParseSessionLine_ModelEffort(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantModel  string
		wantEffort string
		wantOK     bool
	}{
		{
			name:       "active session with model and effort",
			line:       "work:/tmp/work:claude:opus:high:1776876938",
			wantModel:  "opus",
			wantEffort: "high",
			wantOK:     true,
		},
		{
			name:       "pre-existing session — empty model and effort",
			line:       "old:/tmp/old:zsh:::1776876938",
			wantModel:  "",
			wantEffort: "",
			wantOK:     true,
		},
		{
			name:       "model with brackets",
			line:       "work:/home/dev:claude:sonnet[1m]:medium:1776876938",
			wantModel:  "sonnet[1m]",
			wantEffort: "medium",
			wantOK:     true,
		},
		{
			name:   "old 5-part format is rejected",
			line:   "work:/tmp:claude:opus:high",
			wantOK: false,
		},
		{
			name:   "empty line is rejected",
			line:   "",
			wantOK: false,
		},
		{
			name:       "session name, name and all empty fields",
			line:       "sess:/dir:zsh:::1776876938",
			wantModel:  "",
			wantEffort: "",
			wantOK:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseSessionLine(tt.line)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantModel, got.Model)
				assert.Equal(t, tt.wantEffort, got.Effort)
			}
		})
	}
}

func TestParseSessionLine_CreatedAt(t *testing.T) {
	got, ok := parseSessionLine("work:/tmp:claude:opus:high:1776876938")
	assert.True(t, ok)
	assert.Equal(t, int64(1776876938), got.CreatedAt)
}

func TestParseSessionLine_CreatedAt_Zero(t *testing.T) {
	got, ok := parseSessionLine("work:/tmp:zsh:::0")
	assert.True(t, ok)
	assert.Equal(t, int64(0), got.CreatedAt)
}

func TestDeriveStatus_VersionedBinary(t *testing.T) {
	// Claude CLI symlinks to a versioned binary; pane_current_command shows the version string.
	assert.Equal(t, session.Active, deriveStatus("2.1.126"))
	assert.Equal(t, session.Active, deriveStatus("claude"))
	assert.Equal(t, session.Dead, deriveStatus("zsh"))
	assert.Equal(t, session.Dead, deriveStatus("bash"))
	assert.Equal(t, session.Dead, deriveStatus(""))
}
