package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDep_Found(t *testing.T) {
	// "go" should always be on PATH in a dev environment
	dep := checkDep("go")
	assert.True(t, dep.Found)
	assert.NotEmpty(t, dep.Version)
	assert.Equal(t, "go", dep.Name)
}

func TestCheckDep_NotFoundFzf_HasInstallCmd(t *testing.T) {
	// Use fzf as the test dep since it has a known install cmd in the map.
	// Even if fzf is installed, we test the install cmd is populated correctly.
	dep := checkDep("fzf")
	assert.Equal(t, "brew install fzf", dep.InstallCmd)
}

func TestCheckDep_UnknownBinary_NotFound(t *testing.T) {
	dep := checkDep("__cs_nonexistent_binary_xyz__")
	assert.False(t, dep.Found)
	assert.Empty(t, dep.Version)
}

func TestEnsureDataDir_CreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cs-data")
	err := ensureDataDir(dir)
	require.NoError(t, err)
	_, err = os.Stat(dir)
	assert.NoError(t, err)
}

func TestEnsureDataDir_ExistingDirNoError(t *testing.T) {
	dir := t.TempDir()
	err := ensureDataDir(dir)
	assert.NoError(t, err)
}

func TestCheck_ReturnsAllDeps(t *testing.T) {
	statuses := Check()
	assert.Len(t, statuses, 3)
	names := make([]string, len(statuses))
	for i, s := range statuses {
		names[i] = s.Name
	}
	assert.Contains(t, names, "tmux")
	assert.Contains(t, names, "fzf")
	assert.Contains(t, names, "claude")
}

func TestEnsureDataDir_Default(t *testing.T) {
	// Just verify it doesn't error when the dir already exists
	err := EnsureDataDir()
	assert.NoError(t, err)
}

func TestDependencyStatus_Table(t *testing.T) {
	cases := []struct {
		name   string
		binary string
		found  bool
	}{
		{"go on PATH", "go", true},
		{"missing binary", "__no_such_binary_abc__", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dep := checkDep(tc.binary)
			assert.Equal(t, tc.found, dep.Found)
		})
	}
}
