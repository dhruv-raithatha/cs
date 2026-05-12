//go:build integration

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// notifyScriptPath returns the path to the installed notify.sh for integration tests.
// It writes the embedded script to a temp file for testing.
func notifyScriptPath(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "notify.sh")
	require.NoError(t, os.WriteFile(dst, notifyShScript, 0o700))
	return dst
}

// mockBin creates a mock binary in a temp dir that records its arguments to argsFile.
// Returns the dir containing the mock (prepend to PATH).
func mockBin(t *testing.T, name string) (dir string, argsFile string) {
	t.Helper()
	dir = t.TempDir()
	argsFile = filepath.Join(dir, name+".args")
	script := fmt.Sprintf(`#!/usr/bin/env bash
printf '%%s\n' "$@" > %q
exit 0
`, argsFile)
	binPath := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(binPath, []byte(script), 0o700))
	return dir, argsFile
}

// readArgs reads the recorded arguments from a mock binary's argsFile.
func readArgs(t *testing.T, argsFile string) []string {
	t.Helper()
	data, err := os.ReadFile(argsFile)
	if err != nil {
		return nil
	}
	var args []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			args = append(args, line)
		}
	}
	return args
}

func runNotify(t *testing.T, scriptPath, extraPath, payload string, env []string) error {
	t.Helper()
	cmd := exec.Command("bash", scriptPath)
	cmd.Stdin = strings.NewReader(payload)

	// Check whether the caller is overriding PATH in env.
	callerOverridesPath := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			callerOverridesPath = true
		}
	}

	if callerOverridesPath {
		// Use caller's PATH verbatim; only add non-PATH vars from os.Environ().
		var base []string
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "PATH=") {
				base = append(base, e)
			}
		}
		cmd.Env = append(base, env...)
	} else {
		// Prepend extraPath to the existing PATH.
		path := extraPath + ":" + os.Getenv("PATH")
		cmd.Env = append(os.Environ(), "PATH="+path)
		cmd.Env = append(cmd.Env, env...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("notify.sh output: %s", out)
	}
	return err
}

func TestNotifyScript_GhosttyNativeSender(t *testing.T) {
	script := notifyScriptPath(t)
	mockDir, argsFile := mockBin(t, "terminal-notifier")

	payload := `{"hook_event_name":"Notification","session_id":"abc12345-test","cwd":"/Users/dev/myproject/src","message":"Can I run: make build?","notification_type":"permission_prompt"}`

	err := runNotify(t, script, mockDir, payload, []string{
		"CS_TMUX_SOCKET=/nonexistent.sock",
	})
	require.NoError(t, err)

	args := readArgs(t, argsFile)
	require.NotEmpty(t, args, "terminal-notifier should have been called")

	// Verify Ghostty sender identity
	assert.Contains(t, args, "com.mitchellh.ghostty", "sender must be Ghostty bundle ID")
	assert.Contains(t, args, "-sender", "must include -sender flag")

	// Verify group ID uses truncated session ID (replace-not-stack)
	found := false
	for _, a := range args {
		if strings.HasPrefix(a, "cs-") {
			found = true
			break
		}
	}
	assert.True(t, found, "group arg must start with cs-")

	// Verify message preview present
	assert.Contains(t, args, "Can I run: make build?")
}

func TestNotifyScript_StopHook_NoNotification(t *testing.T) {
	script := notifyScriptPath(t)
	mockDir, argsFile := mockBin(t, "terminal-notifier")

	payload := `{"hook_event_name":"Stop","session_id":"abc12345","cwd":"/tmp","usage":{"input_tokens":1000,"output_tokens":200}}`

	err := runNotify(t, script, mockDir, payload, nil)
	require.NoError(t, err)

	// terminal-notifier must NOT have been called
	assert.Nil(t, readArgs(t, argsFile), "Stop hook must not dispatch notification")
}

func TestNotifyScript_TMUXTargetWithQuote_DropsExecute(t *testing.T) {
	script := notifyScriptPath(t)
	mockDir, argsFile := mockBin(t, "terminal-notifier")

	// We can't inject TMUX_TARGET directly (it's resolved from TTY lookup),
	// so we verify indirectly: the script should still send a notification
	// but -execute should only appear when TMUX_TARGET is safe.
	// This test verifies that a notification fires even when pane resolution fails.
	payload := `{"hook_event_name":"Notification","session_id":"test-safe","cwd":"/tmp/proj","message":"hello","notification_type":"permission_prompt"}`

	err := runNotify(t, script, mockDir, payload, []string{
		"CS_TMUX_SOCKET=/nonexistent.sock",
	})
	require.NoError(t, err)

	args := readArgs(t, argsFile)
	require.NotEmpty(t, args, "notification must fire even without pane resolution")

	// -execute must not appear when pane resolution fails (no valid TMUX_TARGET)
	assert.NotContains(t, args, "-execute", "no -execute when pane unresolved")
}

func TestNotifyScript_TerminalNotifierAbsent_GracefulDegradation(t *testing.T) {
	script := notifyScriptPath(t)

	// Build a PATH that excludes terminal-notifier's directory so it is truly absent.
	// We keep essential system dirs (jq, bash, awk, sed, etc.) so the script can run.
	tnPath, err := exec.LookPath("terminal-notifier")
	if err != nil {
		t.Skip("terminal-notifier not on PATH — already absent, test not needed")
	}
	tnDir := filepath.Dir(tnPath)

	var pathParts []string
	for _, p := range filepath.SplitList(os.Getenv("PATH")) {
		if p != tnDir {
			pathParts = append(pathParts, p)
		}
	}
	noTNPath := strings.Join(pathParts, ":")

	payload := `{"hook_event_name":"Notification","session_id":"abc","cwd":"/tmp","message":"test","notification_type":"permission_prompt"}`

	// Script must exit 0 even when terminal-notifier is absent.
	noTNErr := runNotify(t, script, "", payload, []string{
		"PATH=" + noTNPath,
		"CS_TMUX_SOCKET=/nonexistent.sock",
	})
	assert.NoError(t, noTNErr, "must exit 0 even when terminal-notifier absent")
}

func TestNotifyScript_LogCreatedWithSecurePermissions(t *testing.T) {
	script := notifyScriptPath(t)
	mockDir, _ := mockBin(t, "terminal-notifier")

	// Use a temp HOME so log lands in a predictable location
	tmpHome := t.TempDir()
	payload := `{"hook_event_name":"Notification","session_id":"log-test","cwd":"/tmp","message":"log check","notification_type":"permission_prompt"}`

	err := runNotify(t, script, mockDir, payload, []string{
		"HOME=" + tmpHome,
		"CS_TMUX_SOCKET=/nonexistent.sock",
	})
	require.NoError(t, err)

	logFile := filepath.Join(tmpHome, ".cs", "notification.log")
	info, statErr := os.Stat(logFile)
	require.NoError(t, statErr, "log file must be created")
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), "log must be mode 0600")
}

func TestNotifyScript_StallMode_FiresNotification(t *testing.T) {
	script := notifyScriptPath(t)
	mockDir, argsFile := mockBin(t, "terminal-notifier")

	// Mock tmux display-message to return predictable values
	tmuxMockDir := t.TempDir()
	tmuxScript := `#!/usr/bin/env bash
# Mock: return cwd or idle seconds based on format arg
last="${@: -1}"
case "$last" in
  *pane_current_path*) echo "/Users/dev/stall-project" ;;
  *window_silence_interval*) echo "190" ;;  # ~3 minutes
  *) echo "" ;;
esac
exit 0
`
	require.NoError(t, os.WriteFile(filepath.Join(tmuxMockDir, "tmux"), []byte(tmuxScript), 0o700))

	// Stall invocation: positional arg "stall"
	cmd := exec.Command("bash", script, "stall")
	cmd.Env = append(os.Environ(),
		"PATH="+mockDir+":"+tmuxMockDir+":"+os.Getenv("PATH"),
		"TMUX_SESSION=my-stall-session",
		"TMUX_WINDOW=0",
		"CS_TMUX_SOCKET=/nonexistent.sock",
	)
	out, err := cmd.CombinedOutput()
	t.Logf("stall output: %s", out)
	require.NoError(t, err)

	args := readArgs(t, argsFile)
	require.NotEmpty(t, args, "stall must fire terminal-notifier")

	// Message must mention idle time
	found := false
	for _, a := range args {
		if strings.Contains(a, "idle") {
			found = true
			break
		}
	}
	assert.True(t, found, "stall notification message must mention idle")
	assert.Contains(t, args, "com.mitchellh.ghostty")
}
