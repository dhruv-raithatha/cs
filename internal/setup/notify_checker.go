package setup

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// csBlockBegin and csBlockEnd are the markers wrapping cs-managed tmux.conf additions.
const (
	csBlockBegin = "# --- cs begin ---"
	csBlockEnd   = "# --- cs end ---"
)

// CsBlockBegin returns the begin marker string (exported for cli package use).
func CsBlockBegin() string { return csBlockBegin }

// CsBlockEnd returns the end marker string (exported for cli package use).
func CsBlockEnd() string { return csBlockEnd }

// TmuxConfState describes the relationship between the existing tmux.conf and the embedded one.
type TmuxConfState int

const (
	TmuxConfAbsent   TmuxConfState = iota // ~/.tmux.conf does not exist
	TmuxConfIdentical                     // exists and matches embedded byte-for-byte
	TmuxConfDiffers                       // exists but differs from embedded
)

// GhosttyInstalled reports whether Ghostty is available on this machine.
// It checks for /Applications/Ghostty.app first, then falls back to PATH.
func GhosttyInstalled() bool {
	return ghosttyInstalledAt("/Applications/Ghostty.app", os.Getenv("PATH"))
}

// ghosttyInstalledAt is the testable inner implementation.
func ghosttyInstalledAt(bundlePath, pathEnv string) bool {
	if _, err := os.Stat(bundlePath); err == nil {
		return true
	}
	// Check PATH
	for _, dir := range filepath.SplitList(pathEnv) {
		candidate := filepath.Join(dir, "ghostty")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// TerminalNotifierInstalled reports whether terminal-notifier is on PATH.
func TerminalNotifierInstalled() bool {
	return terminalNotifierInstalledOnPath(os.Getenv("PATH"))
}

// terminalNotifierInstalledOnPath is the testable inner implementation.
func terminalNotifierInstalledOnPath(pathEnv string) bool {
	for _, dir := range filepath.SplitList(pathEnv) {
		candidate := filepath.Join(dir, "terminal-notifier")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// TmuxConfStatus returns the state of ~/.tmux.conf relative to the embedded config.
func TmuxConfStatus(path string, embedded []byte) (TmuxConfState, error) {
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return TmuxConfAbsent, nil
	}
	if err != nil {
		return TmuxConfAbsent, fmt.Errorf("read %s: %w", path, err)
	}
	if bytes.Equal(existing, embedded) {
		return TmuxConfIdentical, nil
	}
	return TmuxConfDiffers, nil
}

// AppendCsBlock appends (or replaces) the cs-managed block in the file at path.
// If the file already contains cs markers, only the content between them is updated.
// Uses an atomic write (temp file + rename).
func AppendCsBlock(path string, block []byte) error {
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var out []byte
	s := string(existing)

	beginIdx := strings.Index(s, csBlockBegin)
	endIdx := strings.Index(s, csBlockEnd)

	if beginIdx >= 0 && endIdx > beginIdx {
		// Replace existing cs block content.
		before := s[:beginIdx]
		after := s[endIdx+len(csBlockEnd):]
		out = []byte(before + buildCsBlock(block) + after)
	} else {
		// Append new cs block.
		out = append(existing, []byte("\n"+buildCsBlock(block))...)
	}

	return atomicWrite(path, out, 0o644)
}

// RemoveCsBlock strips the cs-managed block from the file at path.
// If the file does not exist or contains no markers, it returns nil.
func RemoveCsBlock(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	s := string(data)
	beginIdx := strings.Index(s, csBlockBegin)
	endIdx := strings.Index(s, csBlockEnd)
	if beginIdx < 0 || endIdx <= beginIdx {
		return nil // no markers, nothing to remove
	}

	// Strip the block including a leading newline if present.
	before := strings.TrimRight(s[:beginIdx], "\n")
	after := s[endIdx+len(csBlockEnd):]
	if len(before) > 0 {
		before += "\n"
	}
	out := before + strings.TrimLeft(after, "\n")
	if len(out) > 0 && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	return atomicWrite(path, []byte(out), 0o644)
}

// UnifiedDiff returns a human-readable diff between existing and incoming,
// limited to maxLines output lines.
func UnifiedDiff(existing, incoming []byte, maxLines int) string {
	if bytes.Equal(existing, incoming) {
		return ""
	}

	// Write both sides to temp files and run diff -u.
	tmpA, err := os.CreateTemp("", "tmux-existing-*.conf")
	if err != nil {
		return ""
	}
	defer func() { _ = os.Remove(tmpA.Name()) }()

	tmpB, err := os.CreateTemp("", "tmux-incoming-*.conf")
	if err != nil {
		return ""
	}
	defer func() { _ = os.Remove(tmpB.Name()) }()

	if _, err := tmpA.Write(existing); err != nil || tmpA.Close() != nil {
		return ""
	}
	if _, err := tmpB.Write(incoming); err != nil || tmpB.Close() != nil {
		return ""
	}

	out, _ := exec.Command("diff", "-u", "--label", "~/.tmux.conf", "--label", "embedded", //nolint:gosec
		tmpA.Name(), tmpB.Name()).Output()
	// diff exits 1 when files differ — that's expected, ignore the error.

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) > maxLines {
		total := len(lines)
		lines = lines[:maxLines-1]
		lines = append(lines, fmt.Sprintf("... (%d more lines)", total-(maxLines-1)))
	}
	return strings.Join(lines, "\n")
}

// buildCsBlock wraps block content with the cs begin/end markers.
func buildCsBlock(block []byte) string {
	return csBlockBegin + "\n" + string(block) + csBlockEnd + "\n"
}

// atomicWrite writes data to path using a temp file in the same directory + rename.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmux-*.conf.tmp")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	return os.Rename(tmpPath, path)
}
