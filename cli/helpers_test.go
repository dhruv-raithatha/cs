package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dhruv/cs/internal/session"
)

func TestVersionShort(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"tmux 3.4", "3.4"},
		{"fzf 0.71.0 (abc)", "0.71.0"},
		{"claude 1.2.3", "1.2.3"},
		{"v1.0.0", "v1.0.0"},
		{"", ""},
		{"some-name 2.0", "2.0"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := versionShort(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPrintJSON_Valid(t *testing.T) {
	sessions := []session.Session{
		{Name: "work", WorkingDir: "/tmp/work", Status: session.Active},
		{Name: "old", WorkingDir: "/tmp/old", Status: session.Dead},
	}
	var buf bytes.Buffer
	serr := printJSON(&buf, sessions)
	require.NoError(t, serr)

	out := buf.String()
	assert.Contains(t, out, `"name":"work"`)
	assert.Contains(t, out, `"status":"active"`)
	assert.Contains(t, out, `"name":"old"`)
	assert.Contains(t, out, `"status":"dead"`)
}

func TestPrintTable(t *testing.T) {
	sessions := []session.Session{
		{Name: "my-proj", WorkingDir: "/tmp/proj", Status: session.Active},
	}
	var buf bytes.Buffer
	printTable(&buf, sessions, false, false)
	out := buf.String()

	assert.True(t, strings.Contains(out, "my-proj"))
	assert.True(t, strings.Contains(out, "/tmp/proj"))
}

func TestExpandHome(t *testing.T) {
	home, _ := os.UserHomeDir()
	cases := []struct {
		input string
		want  string
	}{
		{"~/foo/bar", home + "/foo/bar"},
		{"/absolute/path", "/absolute/path"},
		{"relative", "relative"},
		{"~", "~"},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, expandHome(tc.input), tc.input)
	}
}
