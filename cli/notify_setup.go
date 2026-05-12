package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dhruv/cs/internal/setup"
)

// notifyScriptInstallPath returns the path where the notify script is installed.
func notifyScriptInstallPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "cs", "notify.sh")
}

// tmuxConfPath returns the path to the user's tmux.conf.
func tmuxConfPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".tmux.conf")
}

// isNotifyInstalled reports whether the notify script is already installed.
func isNotifyInstalled(scriptPath string) bool {
	_, err := os.Stat(scriptPath)
	return err == nil
}

// installNotifyScript writes the embedded notify script to dest with mode 0700.
func installNotifyScript(dest string, script []byte) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	if err := os.WriteFile(dest, script, 0o700); err != nil {
		return fmt.Errorf("write notify script: %w", err)
	}
	return nil
}

// registerNotifyHooks merges cs hook entries into ~/.claude/settings.json.
func registerNotifyHooks(scriptPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	existing, err := setup.ReadHookSettings(settingsPath)
	if err != nil {
		return fmt.Errorf("read hook settings: %w", err)
	}

	entries := []setup.HookEntry{
		{
			EventType: "Notification",
			Matcher:   "permission_prompt|idle_prompt|elicitation_dialog",
			Command:   scriptPath,
		},
		{
			EventType: "Stop",
			Matcher:   "",
			Command:   scriptPath,
		},
	}

	merged, _ := setup.MergeHooks(existing, entries)
	return setup.WriteHookSettings(settingsPath, merged)
}

// fireTestNotification invokes the notify script with a test payload.
// It runs in the background so cs setup never blocks waiting for notification
// center delivery. The notification will appear momentarily after setup completes.
func fireTestNotification(scriptPath string) error {
	cwd, _ := os.Getwd()

	// Build payload via jq to ensure correct JSON escaping.
	payload, err := exec.Command("jq", "-n", //nolint:gosec
		"--arg", "cwd", cwd,
		`{"hook_event_name":"Notification","session_id":"cs-setup-test","cwd":$cwd,"message":"cs notifications are working — click to open Ghostty","notification_type":"test"}`).
		Output()
	if err != nil {
		payload = buildTestPayloadFallback(cwd)
	}

	// Start the notify script in the background; do not wait for it.
	cmd := exec.Command("bash", scriptPath) //nolint:gosec
	cmd.Stdin = strings.NewReader(strings.TrimSpace(string(payload)))
	// Discard stdout/stderr — terminal-notifier prints a notification ID we don't need.
	return cmd.Start()
}

// buildTestPayloadFallback constructs a safe JSON test payload without jq.
// CWD is only used for display and is already a filesystem path (no injection concern).
func buildTestPayloadFallback(cwd string) []byte {
	data, _ := json.Marshal(map[string]string{
		"hook_event_name":   "Notification",
		"session_id":        "cs-setup-test",
		"cwd":               cwd,
		"message":           "cs notifications active",
		"notification_type": "test",
	})
	return data
}

// removeNotifyInstall removes the notify script and strips hooks from settings.json.
func removeNotifyInstall(scriptPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	existing, err := setup.ReadHookSettings(settingsPath)
	if err != nil {
		return fmt.Errorf("read hook settings: %w", err)
	}
	cleaned, changed := setup.RemoveHooks(existing, scriptPath)
	if changed {
		if err := setup.WriteHookSettings(settingsPath, cleaned); err != nil {
			return fmt.Errorf("write hook settings: %w", err)
		}
	}

	if err := setup.RemoveCsBlock(tmuxConfPath()); err != nil {
		return fmt.Errorf("remove cs tmux block: %w", err)
	}

	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove notify script: %w", err)
	}
	return nil
}

// backupAndWrite copies src to src+".bk" then overwrites src with content.
func backupAndWrite(path string, content []byte) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if len(existing) > 0 {
		if err := os.WriteFile(path+".bk", existing, 0o644); err != nil {
			return fmt.Errorf("write backup: %w", err)
		}
	}
	return os.WriteFile(path, content, 0o644)
}

// readFileSilent reads a file and returns nil on error (used for diff display).
func readFileSilent(path string) []byte {
	data, _ := os.ReadFile(path)
	return data
}

// csBlock returns the cs-specific lines to append to tmux.conf.
func csBlock() []byte {
	// Extract only the cs-managed block from the embedded config.
	content := string(embeddedTmuxConf)
	beginIdx := strings.Index(content, setup.CsBlockBegin())
	endIdx := strings.Index(content, setup.CsBlockEnd())
	if beginIdx < 0 || endIdx <= beginIdx {
		return nil
	}
	// Return everything between (exclusive of) the markers.
	inner := content[beginIdx+len(setup.CsBlockBegin()) : endIdx]
	return []byte(strings.TrimSpace(inner) + "\n")
}
