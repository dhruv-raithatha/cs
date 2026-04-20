// Package fzf provides the FuzzySelector interface and implementations.
package fzf

// FuzzySelector defines the contract for interactive fuzzy selection.
type FuzzySelector interface {
	Select(items []string, prompt, header string) (string, error)
}
