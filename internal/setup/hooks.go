package setup

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
)

// HookEntry describes a single Claude Code hook to register.
type HookEntry struct {
	EventType string // "Notification" | "Stop"
	Matcher   string // regex or empty string
	Command   string // absolute path to the notify script
}

// HookGroup mirrors the Claude Code settings.json hook group schema.
type HookGroup struct {
	Matcher string    `json:"matcher"`
	Hooks   []HookDef `json:"hooks"`
}

// HookDef mirrors the Claude Code hook definition schema.
type HookDef struct {
	Type    string `json:"type"`    // always "command"
	Command string `json:"command"` // absolute path
}

// ReadHookSettings reads the file at path and returns the parsed settings map.
// If the file does not exist, an empty map is returned without error.
// Returns an error only on malformed JSON.
func ReadHookSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return result, nil
}

// MergeHooks adds cs hook entries to existing settings without overwriting
// unrelated entries. It is idempotent: calling it twice with the same entries
// produces the same result. Command must be an absolute path; relative paths
// are silently rejected (changed = false).
func MergeHooks(existing map[string]any, entries []HookEntry) (result map[string]any, changed bool) {
	// Deep-copy to avoid mutating the caller's map.
	result = deepCopyMap(existing)

	for _, entry := range entries {
		if !filepath.IsAbs(entry.Command) {
			continue // reject relative paths
		}
		if addHookEntry(result, entry) {
			changed = true
		}
	}
	return result, changed
}

// RemoveHooks strips all hook entries whose command field matches scriptPath.
// Empty arrays and empty "hooks" keys are pruned. Returns (modified, changed).
func RemoveHooks(existing map[string]any, scriptPath string) (map[string]any, bool) {
	result := deepCopyMap(existing)

	hooksRaw, ok := result["hooks"]
	if !ok {
		return result, false
	}
	hooksMap, ok := hooksRaw.(map[string]any)
	if !ok {
		return result, false
	}

	changed := false
	for eventType, groupsRaw := range hooksMap {
		groups, ok := groupsRaw.([]any)
		if !ok {
			continue
		}
		newGroups, groupChanged := removeCommandFromGroups(groups, scriptPath)
		if groupChanged {
			changed = true
		}
		if len(newGroups) == 0 {
			delete(hooksMap, eventType)
		} else {
			hooksMap[eventType] = newGroups
		}
	}

	if len(hooksMap) == 0 {
		delete(result, "hooks")
	}
	return result, changed
}

// WriteHookSettings atomically writes settings to path using a temp file in
// the same directory followed by os.Rename. Returns an error if the parent
// directory does not exist.
func WriteHookSettings(path string, settings map[string]any) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".settings-*.json.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op after rename

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}

// ── internal helpers ─────────────────────────────────────────────────────────

// addHookEntry adds one HookEntry to the settings map if not already present.
// Returns true if a change was made.
func addHookEntry(settings map[string]any, entry HookEntry) bool {
	// Ensure hooks map exists.
	hooksRaw, ok := settings["hooks"]
	if !ok {
		settings["hooks"] = map[string]any{}
		hooksRaw = settings["hooks"]
	}
	hooksMap, ok := hooksRaw.(map[string]any)
	if !ok {
		return false
	}

	// Ensure event type array exists.
	groupsRaw, ok := hooksMap[entry.EventType]
	if !ok {
		groupsRaw = []any{}
	}
	groups, ok := groupsRaw.([]any)
	if !ok {
		return false
	}

	// Check whether the command already exists in any group for this event type.
	for _, g := range groups {
		group, ok := g.(map[string]any)
		if !ok {
			continue
		}
		defs, ok := group["hooks"].([]any)
		if !ok {
			continue
		}
		for _, d := range defs {
			def, ok := d.(map[string]any)
			if !ok {
				continue
			}
			if def["command"] == entry.Command {
				return false // already present
			}
		}
	}

	// Append a new group for this entry.
	newGroup := map[string]any{
		"matcher": entry.Matcher,
		"hooks": []any{
			map[string]any{"type": "command", "command": entry.Command},
		},
	}
	hooksMap[entry.EventType] = append(groups, newGroup)
	settings["hooks"] = hooksMap
	return true
}

// removeCommandFromGroups strips defs matching scriptPath from each group.
// Groups with no remaining defs are removed. Returns (cleaned groups, changed).
func removeCommandFromGroups(groups []any, scriptPath string) ([]any, bool) {
	changed := false
	var result []any
	for _, gRaw := range groups {
		group, ok := gRaw.(map[string]any)
		if !ok {
			result = append(result, gRaw)
			continue
		}
		defs, ok := group["hooks"].([]any)
		if !ok {
			result = append(result, gRaw)
			continue
		}
		var newDefs []any
		for _, dRaw := range defs {
			def, ok := dRaw.(map[string]any)
			if !ok {
				newDefs = append(newDefs, dRaw)
				continue
			}
			if def["command"] == scriptPath {
				changed = true
				continue // drop it
			}
			newDefs = append(newDefs, dRaw)
		}
		if len(newDefs) == 0 {
			changed = true
			continue // drop empty group
		}
		newGroup := make(map[string]any, len(group))
		maps.Copy(newGroup, group)
		newGroup["hooks"] = newDefs
		result = append(result, newGroup)
	}
	return result, changed
}

// deepCopyMap deep-copies m through a JSON round-trip so mutations to the
// returned map never alias the input.
func deepCopyMap(m map[string]any) map[string]any {
	data, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	if out == nil {
		return map[string]any{}
	}
	return out
}
