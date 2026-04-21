package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/dhruv/cs/internal/fzf"
	"github.com/dhruv/cs/internal/session"
	"github.com/dhruv/cs/internal/tmux"
)

func newTestApp(tmuxClient tmux.TmuxClient, selector fzf.FuzzySelector) *cli.Command {
	return &cli.Command{
		Name: "cs",
		// Prevent os.Exit from being called during tests
		ExitErrHandler: func(_ context.Context, _ *cli.Command, _ error) {},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "socket", Value: "test.sock"},
		},
		Action: RootAction(tmuxClient, selector),
		Commands: []*cli.Command{
			VersionCommand(),
			SetupCommand(),
			ListCommand(tmuxClient),
			AttachCommand(tmuxClient),
			DeleteCommand(tmuxClient),
		},
	}
}

func TestVersionCommand_Output(t *testing.T) {
	app := newTestApp(&tmux.FakeTmuxClient{}, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "version"})
	assert.NoError(t, err)
}

func TestListCommand_NoSessions(t *testing.T) {
	app := newTestApp(&tmux.FakeTmuxClient{}, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "list"})
	assert.NoError(t, err)
}

func TestListCommand_WithSessions_JSON(t *testing.T) {
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "work", WorkingDir: "/tmp", PaneCommand: "claude"},
		},
	}
	app := newTestApp(client, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "list", "--json"})
	assert.NoError(t, err)
}

func TestDeleteCommand_MissingArg(t *testing.T) {
	app := newTestApp(&tmux.FakeTmuxClient{}, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "delete"})
	assert.Error(t, err)
}

func TestDeleteCommand_KillsSession(t *testing.T) {
	client := &tmux.FakeTmuxClient{}
	app := newTestApp(client, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "delete", "my-session"})
	require.NoError(t, err)
	assert.Equal(t, "my-session", client.KilledSession)
}

func TestDeleteCommand_SessionNotFound(t *testing.T) {
	client := &tmux.FakeTmuxClient{KillSessionErr: assert.AnError}
	app := newTestApp(client, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "delete", "ghost"})
	assert.Error(t, err)
}

func TestAttachCommand_MissingArg(t *testing.T) {
	app := newTestApp(&tmux.FakeTmuxClient{}, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "attach"})
	assert.Error(t, err)
}

func TestAttachCommand_AlreadyInTmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1234,0")
	app := newTestApp(&tmux.FakeTmuxClient{}, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "attach", "foo"})
	assert.Error(t, err)
}

func TestAttachCommand_AttachesSession(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{}
	app := newTestApp(client, &fzf.FakeFuzzySelector{})
	err := app.Run(context.Background(), []string{"cs", "attach", "my-session"})
	require.NoError(t, err)
	assert.Equal(t, "my-session", client.AttachedSession)
}

func TestRootAction_Cancel(t *testing.T) {
	t.Setenv("TMUX", "")
	client := &tmux.FakeTmuxClient{
		Sessions: []session.Session{
			{Name: "work", WorkingDir: "/tmp", PaneCommand: "claude"},
		},
	}
	// fzf returns error (user pressed esc)
	selector := &fzf.FakeFuzzySelector{Err: errors.New("fzf: exit status 130")}
	app := newTestApp(client, selector)
	err := app.Run(context.Background(), []string{"cs"})
	assert.NoError(t, err)
}

func TestExpandHome_Exported(t *testing.T) {
	result := ExpandHome("~/test")
	assert.NotEqual(t, "~/test", result)
	assert.Contains(t, result, "/test")
}

// T018: printTable includes MODEL and EFFORT columns; printJSON includes "model" and "effort" keys.

func TestPrintTable_IncludesModelEffortColumns(t *testing.T) {
	sessions := []session.Session{
		{Name: "work", WorkingDir: "/tmp", Model: "opus", Effort: "high", Status: session.Active},
		{Name: "old", WorkingDir: "/tmp2", Model: "", Effort: "", Status: session.Dead},
	}
	var buf strings.Builder
	printTable(&buf, sessions)

	out := buf.String()
	assert.Contains(t, out, "MODEL")
	assert.Contains(t, out, "EFFORT")
	assert.Contains(t, out, "opus")
	assert.Contains(t, out, "high")
	// Pre-existing sessions with no model/effort show empty string (not "unknown") in cs list
	assert.NotContains(t, out, "unknown")
}

func TestPrintJSON_IncludesModelEffortKeys(t *testing.T) {
	sessions := []session.Session{
		{Name: "work", WorkingDir: "/tmp", Model: "opus", Effort: "high", Status: session.Active},
		{Name: "old", WorkingDir: "/tmp2", Model: "", Effort: "", Status: session.Dead},
	}
	var buf strings.Builder
	err := printJSON(&buf, sessions)
	require.NoError(t, err)

	out := buf.String()
	// Session with values
	assert.Contains(t, out, `"model":"opus"`)
	assert.Contains(t, out, `"effort":"high"`)
	// Pre-existing session — model and effort keys present as empty string
	assert.Contains(t, out, `"model":""`)
	assert.Contains(t, out, `"effort":""`)
}
