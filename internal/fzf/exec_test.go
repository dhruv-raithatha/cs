//go:build integration

package fzf

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecFuzzySelector_Select_Cancel(t *testing.T) {
	s := &execFuzzySelector{}
	// fzf with no TTY input will exit non-zero — treated as cancel
	_, err := s.Select([]string{"a", "b"}, "> ", "")
	// Expect an error when not attached to a real TTY
	assert.Error(t, err)
}

func TestExecFuzzySelector_EmptyItems(t *testing.T) {
	s := &execFuzzySelector{}
	// fzf with empty input should not panic
	_, err := s.Select([]string{}, "> ", "select:")
	_ = err
	_ = require.New(t) // satisfies import
}
