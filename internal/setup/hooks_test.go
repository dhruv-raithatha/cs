package setup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── ReadHookSettings ─────────────────────────────────────────────────────────

func TestReadHookSettings_FileAbsent_ReturnsEmptyMap(t *testing.T) {
	result, err := ReadHookSettings("/nonexistent/path/settings.json")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)
}

func TestReadHookSettings_MalformedJSON_ReturnsError(t *testing.T) {
	f := writeTempFile(t, "bad json {{")
	_, err := ReadHookSettings(f)
	assert.Error(t, err)
}

func TestReadHookSettings_ValidJSON_RoundTrips(t *testing.T) {
	content := `{"model":"opus","hooks":{"Stop":[]}}`
	f := writeTempFile(t, content)

	result, err := ReadHookSettings(f)
	require.NoError(t, err)
	assert.Equal(t, "opus", result["model"])
}

// ── MergeHooks ───────────────────────────────────────────────────────────────

func TestMergeHooks_AddsEntry(t *testing.T) {
	existing := map[string]any{}
	entry := HookEntry{
		EventType: "Notification",
		Matcher:   "permission_prompt",
		Command:   "/home/user/.local/share/cs/notify.sh",
	}

	merged, changed := MergeHooks(existing, []HookEntry{entry})
	assert.True(t, changed)

	hooks := merged["hooks"].(map[string]any)
	notifHooks := hooks["Notification"].([]any)
	require.Len(t, notifHooks, 1)

	group := notifHooks[0].(map[string]any)
	assert.Equal(t, "permission_prompt", group["matcher"])
	defs := group["hooks"].([]any)
	require.Len(t, defs, 1)
	def := defs[0].(map[string]any)
	assert.Equal(t, "command", def["type"])
	assert.Equal(t, "/home/user/.local/share/cs/notify.sh", def["command"])
}

func TestMergeHooks_Idempotent_NoDuplicates(t *testing.T) {
	existing := map[string]any{}
	entry := HookEntry{
		EventType: "Notification",
		Matcher:   "permission_prompt",
		Command:   "/absolute/notify.sh",
	}

	merged, _ := MergeHooks(existing, []HookEntry{entry})
	merged2, changed := MergeHooks(merged, []HookEntry{entry})
	assert.False(t, changed, "re-merging same entry must be a no-op")

	hooks := merged2["hooks"].(map[string]any)
	notifHooks := hooks["Notification"].([]any)
	assert.Len(t, notifHooks, 1, "no duplicate groups")
}

func TestMergeHooks_RejectsRelativePath(t *testing.T) {
	existing := map[string]any{}
	entry := HookEntry{
		EventType: "Notification",
		Matcher:   "",
		Command:   "relative/path/notify.sh",
	}

	merged, changed := MergeHooks(existing, []HookEntry{entry})
	assert.False(t, changed, "relative path must be rejected")
	_, hasHooks := merged["hooks"]
	assert.False(t, hasHooks, "hooks key must not be added for rejected entry")
}

func TestMergeHooks_PreservesExistingUserEntries(t *testing.T) {
	existing := map[string]any{
		"model": "opus",
		"hooks": map[string]any{
			"Notification": []any{
				map[string]any{
					"matcher": "other_event",
					"hooks": []any{
						map[string]any{"type": "command", "command": "/user/custom.sh"},
					},
				},
			},
		},
	}
	entry := HookEntry{EventType: "Stop", Matcher: "", Command: "/absolute/notify.sh"}

	merged, changed := MergeHooks(existing, []HookEntry{entry})
	assert.True(t, changed)

	hooks := merged["hooks"].(map[string]any)
	// User's Notification entry must survive
	notifHooks := hooks["Notification"].([]any)
	assert.Len(t, notifHooks, 1)
	// Stop entry added
	stopHooks := hooks["Stop"].([]any)
	assert.Len(t, stopHooks, 1)
}

// ── RemoveHooks ──────────────────────────────────────────────────────────────

func TestRemoveHooks_StripsMatchingCommand(t *testing.T) {
	scriptPath := "/absolute/notify.sh"
	settings := buildSettingsWithHook(t, "Notification", "matcher", scriptPath)

	result, changed := RemoveHooks(settings, scriptPath)
	assert.True(t, changed)

	hooks, ok := result["hooks"]
	assert.False(t, ok || hooks != nil, "hooks key must be removed when empty")
}

func TestRemoveHooks_PrunesEmptyArrays(t *testing.T) {
	scriptPath := "/absolute/notify.sh"
	settings := buildSettingsWithHook(t, "Stop", "", scriptPath)

	result, changed := RemoveHooks(settings, scriptPath)
	assert.True(t, changed)
	_, hasHooks := result["hooks"]
	assert.False(t, hasHooks)
}

func TestRemoveHooks_PreservesOtherCommands(t *testing.T) {
	scriptPath := "/absolute/notify.sh"
	settings := map[string]any{
		"hooks": map[string]any{
			"Notification": []any{
				map[string]any{
					"matcher": "m",
					"hooks": []any{
						map[string]any{"type": "command", "command": scriptPath},
						map[string]any{"type": "command", "command": "/other/script.sh"},
					},
				},
			},
		},
	}

	result, changed := RemoveHooks(settings, scriptPath)
	assert.True(t, changed)
	hooks := result["hooks"].(map[string]any)
	notifHooks := hooks["Notification"].([]any)
	// Group should remain with the other command
	require.Len(t, notifHooks, 1)
	group := notifHooks[0].(map[string]any)
	defs := group["hooks"].([]any)
	assert.Len(t, defs, 1)
	def := defs[0].(map[string]any)
	assert.Equal(t, "/other/script.sh", def["command"])
}

func TestRemoveHooks_NoMatch_NoChange(t *testing.T) {
	settings := map[string]any{"model": "opus"}
	result, changed := RemoveHooks(settings, "/nonexistent.sh")
	assert.False(t, changed)
	assert.Equal(t, "opus", result["model"])
}

// ── WriteHookSettings ────────────────────────────────────────────────────────

func TestWriteHookSettings_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	settings := map[string]any{"model": "opus"}
	require.NoError(t, WriteHookSettings(path, settings))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "opus", result["model"])
}

func TestWriteHookSettings_AtomicWrite_ExistingFileReplaced(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	require.NoError(t, os.WriteFile(path, []byte(`{"old":"value"}`), 0o600))
	require.NoError(t, WriteHookSettings(path, map[string]any{"new": "value"}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "value", result["new"])
	_, hasOld := result["old"]
	assert.False(t, hasOld)
}

func TestWriteHookSettings_MissingParentDir_ReturnsError(t *testing.T) {
	err := WriteHookSettings("/nonexistent/deep/path/settings.json", map[string]any{})
	assert.Error(t, err)
}

// ── helpers ──────────────────────────────────────────────────────────────────

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "settings*.json")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func buildSettingsWithHook(t *testing.T, eventType, matcher, command string) map[string]any {
	t.Helper()
	return map[string]any{
		"hooks": map[string]any{
			eventType: []any{
				map[string]any{
					"matcher": matcher,
					"hooks": []any{
						map[string]any{"type": "command", "command": command},
					},
				},
			},
		},
	}
}
