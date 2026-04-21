package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhruv/cs/internal/fzf"
	"github.com/dhruv/cs/internal/session"
	"github.com/dhruv/cs/internal/tmux"
)

const testSocket = "test.sock"

// runTest drives runWithConfirmReader with auto-confirm and the provided stdin text.
func runTest(t *testing.T, client *tmux.FakeTmuxClient, selector *fzf.FakeFuzzySelector, stdinText string) error {
	t.Helper()
	return runWithConfirmReader(testSocket, client, selector, func(_ string) bool { return true }, strings.NewReader(stdinText))
}

func runTestWithConfirm(t *testing.T, client *tmux.FakeTmuxClient, selector *fzf.FakeFuzzySelector, stdinText string, confirm func(string) bool) error {
	t.Helper()
	return runWithConfirmReader(testSocket, client, selector, confirm, strings.NewReader(stdinText))
}

func TestRun_AlreadyInsideTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
	client := &tmux.FakeTmuxClient{}
	selector := &fzf.FakeFuzzySelector{}
	err := runTest(t, client, selector, "")
	assert.Error(t, err)
	assert.Empty(t, client.AttachedSession)
}

func TestRun_NoSessions_CreatesNew(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("ANTHROPIC_MODEL", "")
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	client := &tmux.FakeTmuxClient{}
	selector := &fzf.FakeFuzzySelector{}
	err := runTest(t, client, selector, "my-project\n")
	require.NoError(t, err)
	assert.Equal(t, "my-project", client.CreatedSession)
	assert.Equal(t, "my-project", client.AttachedSession)
}

func TestRun_NoSessions_EmptyNameThenValid(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("ANTHROPIC_MODEL", "")
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	client := &tmux.FakeTmuxClient{}
	selector := &fzf.FakeFuzzySelector{}
	// First line empty (re-prompt), second line valid
	err := runTest(t, client, selector, "\nmy-project\n")
	require.NoError(t, err)
	assert.Equal(t, "my-project", client.CreatedSession)
}

func TestRun_NoSessions_CancelWithEOF(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{}
	selector := &fzf.FakeFuzzySelector{}
	// Empty reader simulates Ctrl-d / EOF
	err := runTest(t, client, selector, "")
	assert.NoError(t, err)
	assert.Empty(t, client.CreatedSession)
}

func TestRun_HasSessions_AttachExisting(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "work", WorkingDir: "/tmp", PaneCommand: "claude"},
		},
	}
	selector := &fzf.FakeFuzzySelector{
		Selections: []string{"work                 /tmp                        unknown      unknown"},
	}
	err := runTest(t, client, selector, "")
	require.NoError(t, err)
	assert.Equal(t, "work", client.AttachedSession)
}

func TestRun_HasSessions_SelectNew(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("ANTHROPIC_MODEL", "")
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "old", WorkingDir: "/tmp", PaneCommand: "zsh"},
		},
	}
	// User picks [ + new session ] in fzf, then types name via stdin
	selector := &fzf.FakeFuzzySelector{
		Selections: []string{newSessionEntry},
	}
	err := runTest(t, client, selector, "fresh\n")
	require.NoError(t, err)
	assert.Equal(t, "fresh", client.CreatedSession)
}

func TestRun_FzfCancel(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "work", WorkingDir: "/tmp", PaneCommand: "claude"},
		},
	}
	selector := &fzf.FakeFuzzySelector{
		Err: errors.New("fzf: exit status 130"),
	}
	err := runTest(t, client, selector, "")
	assert.NoError(t, err)
}

func TestRun_DeleteConfirm(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "stale", WorkingDir: "/tmp", PaneCommand: "zsh"},
		},
	}
	selector := &fzf.FakeFuzzySelector{
		Selections: []string{deletePrefix + "stale   /tmp   [dead]"},
	}
	err := runTestWithConfirm(t, client, selector, "", func(_ string) bool { return true })
	require.NoError(t, err)
	assert.Equal(t, "stale", client.KilledSession)
}

func TestRun_DeleteCancel(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "stale", WorkingDir: "/tmp", PaneCommand: "zsh"},
		},
	}
	selector := &fzf.FakeFuzzySelector{
		Selections: []string{deletePrefix + "stale   /tmp   [dead]"},
	}
	err := runTestWithConfirm(t, client, selector, "", func(_ string) bool { return false })
	require.NoError(t, err)
	assert.Empty(t, client.KilledSession)
}

// T012: createNewSession forwards model and effort from fzf selections to the tmux client.
func TestCreateNewSession_ModelEffortSelection(t *testing.T) {
	tests := []struct {
		name              string
		selections        []string
		envModel          string
		envEffort         string
		wantModel         string
		wantEffort        string
		wantFirstModel    string // if non-empty, assert selector.Calls[0].Items[0] equals this
		wantFirstEffort   string // if non-empty, assert selector.Calls[1].Items[0] equals this
	}{
		{
			name:       "fzf returns model and effort",
			selections: []string{"opus", "high"},
			wantModel:  "opus",
			wantEffort: "high",
		},
		{
			name:       "fzf returns empty — falls back to built-in defaults",
			selections: nil,
			wantModel:  "sonnet",
			wantEffort: "medium",
		},
		{
			name:            "ANTHROPIC_MODEL env sets haiku as first model option",
			selections:      nil,
			envModel:        "haiku",
			wantModel:       "haiku",
			wantEffort:      "medium",
			wantFirstModel:  "haiku",
		},
		{
			name:            "CLAUDE_CODE_EFFORT_LEVEL env sets xhigh as first effort option",
			selections:      nil,
			envEffort:       "xhigh",
			wantModel:       "sonnet",
			wantEffort:      "xhigh",
			wantFirstEffort: "xhigh",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TMUX", "")
			t.Setenv("ANTHROPIC_MODEL", tt.envModel)
			t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", tt.envEffort)

			client := &tmux.FakeTmuxClient{}
			selector := &fzf.FakeFuzzySelector{Selections: tt.selections}
			err := runTest(t, client, selector, "my-session\n")
			require.NoError(t, err)

			assert.Equal(t, tt.wantModel, client.CreatedModel)
			assert.Equal(t, tt.wantEffort, client.CreatedEffort)

			if tt.wantFirstModel != "" {
				require.GreaterOrEqual(t, len(selector.Calls), 1)
				require.NotEmpty(t, selector.Calls[0].Items)
				assert.Equal(t, tt.wantFirstModel, selector.Calls[0].Items[0])
			}
			if tt.wantFirstEffort != "" {
				require.GreaterOrEqual(t, len(selector.Calls), 2)
				require.NotEmpty(t, selector.Calls[1].Items)
				assert.Equal(t, tt.wantFirstEffort, selector.Calls[1].Items[0])
			}
		})
	}
}

// T016: runPicker formats each entry with model and effort columns.
func TestRunPicker_EntryFormat(t *testing.T) {
	tests := []struct {
		name       string
		session    session.Session
		wantModel  string
		wantEffort string
		wantDead   bool
	}{
		{
			name: "active session with model and effort",
			session: session.Session{
				Name: "work", WorkingDir: "/home/dev", PaneCommand: "claude",
				Model: "opus", Effort: "high",
			},
			wantModel:  "opus",
			wantEffort: "high",
		},
		{
			name: "pre-existing session without model or effort shows unknown",
			session: session.Session{
				Name: "old", WorkingDir: "/tmp", PaneCommand: "zsh",
			},
			wantModel:  "unknown",
			wantEffort: "unknown",
			wantDead:   true,
		},
		{
			name: "dead session with model and effort retains dead tag",
			session: session.Session{
				Name: "stale", WorkingDir: "/var", PaneCommand: "bash",
				Model: "haiku", Effort: "low",
			},
			wantModel:  "haiku",
			wantEffort: "low",
			wantDead:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TMUX", "")
			client := &tmux.FakeTmuxClient{Sessions: []session.Session{tt.session}}
			// Cancel immediately so we only need to inspect the picker items.
			selector := &fzf.FakeFuzzySelector{Err: errors.New("cancelled")}
			err := runTest(t, client, selector, "")
			assert.NoError(t, err)

			require.Len(t, selector.Calls, 1)
			items := selector.Calls[0].Items
			require.GreaterOrEqual(t, len(items), 2)

			entry := items[1] // items[0] is newSessionEntry
			assert.Contains(t, entry, tt.wantModel)
			assert.Contains(t, entry, tt.wantEffort)
			if tt.wantDead {
				assert.Contains(t, entry, "[dead]")
			}
		})
	}
}

// TestOrderedWithDefault verifies the orderedWithDefault helper.
func TestOrderedWithDefault(t *testing.T) {
	list := []string{"a", "b", "c", "d"}
	assert.Equal(t, []string{"c", "a", "b", "d"}, orderedWithDefault(list, "c"))
	assert.Equal(t, []string{"a", "b", "c", "d"}, orderedWithDefault(list, "a")) // already first
	assert.Equal(t, []string{"a", "b", "c", "d"}, orderedWithDefault(list, "z")) // not found
	assert.Equal(t, []string{"a", "b", "c", "d"}, orderedWithDefault(list, ""))  // empty
}
