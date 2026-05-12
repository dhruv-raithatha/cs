package setup

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── GhosttyInstalled ─────────────────────────────────────────────────────────

func TestGhosttyInstalled_AppBundlePresent(t *testing.T) {
	// Create a fake /Applications/Ghostty.app directory in a temp dir.
	fakeApps := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(fakeApps, "Ghostty.app"), 0o755))
	assert.True(t, ghosttyInstalledAt(fakeApps+"/Ghostty.app", ""))
}

func TestGhosttyInstalled_AppBundleAbsent_BinaryOnPath(t *testing.T) {
	// Create a fake ghostty binary on PATH.
	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "ghostty"), []byte("#!/bin/sh"), 0o700))
	assert.True(t, ghosttyInstalledAt("/nonexistent/Ghostty.app", binDir+":"+os.Getenv("PATH")))
}

func TestGhosttyInstalled_Neither(t *testing.T) {
	assert.False(t, ghosttyInstalledAt("/nonexistent/Ghostty.app", "/nonexistent"))
}

// ── TerminalNotifierInstalled ─────────────────────────────────────────────────

func TestTerminalNotifierInstalled_Present(t *testing.T) {
	binDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "terminal-notifier"), []byte("#!/bin/sh"), 0o700))
	assert.True(t, terminalNotifierInstalledOnPath(binDir+":"+os.Getenv("PATH")))
}

func TestTerminalNotifierInstalled_Absent(t *testing.T) {
	assert.False(t, terminalNotifierInstalledOnPath("/nonexistent"))
}

// ── TmuxConfStatus ────────────────────────────────────────────────────────────

func TestTmuxConfStatus_Absent(t *testing.T) {
	embedded := []byte("# tmux config")
	state, err := TmuxConfStatus("/nonexistent/.tmux.conf", embedded)
	require.NoError(t, err)
	assert.Equal(t, TmuxConfAbsent, state)
}

func TestTmuxConfStatus_Identical(t *testing.T) {
	content := []byte("# tmux config\nset -g mouse on\n")
	path := filepath.Join(t.TempDir(), ".tmux.conf")
	require.NoError(t, os.WriteFile(path, content, 0o644))

	state, err := TmuxConfStatus(path, content)
	require.NoError(t, err)
	assert.Equal(t, TmuxConfIdentical, state)
}

func TestTmuxConfStatus_Differs(t *testing.T) {
	existing := []byte("# old config\n")
	embedded := []byte("# new config\nset -g mouse on\n")
	path := filepath.Join(t.TempDir(), ".tmux.conf")
	require.NoError(t, os.WriteFile(path, existing, 0o644))

	state, err := TmuxConfStatus(path, embedded)
	require.NoError(t, err)
	assert.Equal(t, TmuxConfDiffers, state)
}

// ── AppendCsBlock ─────────────────────────────────────────────────────────────

func TestAppendCsBlock_NoMarkers_AppendsBlock(t *testing.T) {
	existing := "# user config\nset -g mouse on\n"
	path := filepath.Join(t.TempDir(), ".tmux.conf")
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o644))

	block := []byte("allow-passthrough on\n")
	require.NoError(t, AppendCsBlock(path, block))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, "# user config")
	assert.Contains(t, s, csBlockBegin)
	assert.Contains(t, s, "allow-passthrough on")
	assert.Contains(t, s, csBlockEnd)
}

func TestAppendCsBlock_WithExistingMarkers_ReplacesBlock(t *testing.T) {
	existing := "# user\n" + csBlockBegin + "\nold content\n" + csBlockEnd + "\n"
	path := filepath.Join(t.TempDir(), ".tmux.conf")
	require.NoError(t, os.WriteFile(path, []byte(existing), 0o644))

	block := []byte("new content\n")
	require.NoError(t, AppendCsBlock(path, block))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(data)
	assert.Contains(t, s, "new content")
	assert.NotContains(t, s, "old content")
	// Only one pair of markers
	assert.Equal(t, 1, strings.Count(s, csBlockBegin))
}

func TestAppendCsBlock_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".tmux.conf")
	require.NoError(t, os.WriteFile(path, []byte("# base\n"), 0o644))

	block := []byte("allow-passthrough on\n")
	require.NoError(t, AppendCsBlock(path, block))
	require.NoError(t, AppendCsBlock(path, block)) // second call

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(data)
	assert.Equal(t, 1, strings.Count(s, csBlockBegin), "markers must appear exactly once")
}

// ── RemoveCsBlock ─────────────────────────────────────────────────────────────

func TestRemoveCsBlock_RemovesBlock(t *testing.T) {
	content := "# user\n" + csBlockBegin + "\nstuff\n" + csBlockEnd + "\n# after\n"
	path := filepath.Join(t.TempDir(), ".tmux.conf")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	require.NoError(t, RemoveCsBlock(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	s := string(data)
	assert.NotContains(t, s, csBlockBegin)
	assert.NotContains(t, s, "stuff")
	assert.Contains(t, s, "# user")
	assert.Contains(t, s, "# after")
}

func TestRemoveCsBlock_NoMarkers_NoChange(t *testing.T) {
	content := "# user config\n"
	path := filepath.Join(t.TempDir(), ".tmux.conf")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	require.NoError(t, RemoveCsBlock(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestRemoveCsBlock_FileAbsent_NoError(t *testing.T) {
	err := RemoveCsBlock("/nonexistent/.tmux.conf")
	assert.NoError(t, err)
}

// ── UnifiedDiff ───────────────────────────────────────────────────────────────

func TestUnifiedDiff_NoDiff_EmptyString(t *testing.T) {
	result := UnifiedDiff([]byte("same\n"), []byte("same\n"), 40)
	assert.Empty(t, result)
}

func TestUnifiedDiff_HasDiff_ContainsLines(t *testing.T) {
	a := []byte("line1\nline2\n")
	b := []byte("line1\nchanged\n")
	result := UnifiedDiff(a, b, 40)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "changed")
}

func TestUnifiedDiff_TruncatesToMaxLines(t *testing.T) {
	var sb strings.Builder
	for range 100 {
		sb.WriteString("old line\n")
	}
	a := []byte(sb.String())
	b := bytes.ReplaceAll(a, []byte("old"), []byte("new"))

	result := UnifiedDiff(a, b, 10)
	lines := strings.Split(strings.TrimSpace(result), "\n")
	assert.LessOrEqual(t, len(lines), 10)
}
