package tmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T008: Table-driven tests for the 5-part ListSessions parsing helper.
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
			line:       "work:/tmp/work:claude:opus:high",
			wantModel:  "opus",
			wantEffort: "high",
			wantOK:     true,
		},
		{
			name:       "pre-existing session — empty model and effort",
			line:       "old:/tmp/old:zsh::",
			wantModel:  "",
			wantEffort: "",
			wantOK:     true,
		},
		{
			name:       "model with brackets",
			line:       "work:/home/dev:claude:sonnet[1m]:medium",
			wantModel:  "sonnet[1m]",
			wantEffort: "medium",
			wantOK:     true,
		},
		{
			name:   "old 3-part format is rejected",
			line:   "work:/tmp:claude",
			wantOK: false,
		},
		{
			name:   "empty line is rejected",
			line:   "",
			wantOK: false,
		},
		{
			name:       "session name, name and all empty fields",
			line:       "sess:/dir:zsh::",
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
