package fzf

import "slices"

// SelectCall records the arguments of a single Select invocation.
type SelectCall struct {
	Items  []string
	Prompt string
	Header string
}

// FakeFuzzySelector is a configurable test double for FuzzySelector.
type FakeFuzzySelector struct {
	// Selections is consumed in order; each call pops the first element.
	Selections []string
	Err        error

	LastItems  []string
	LastPrompt string
	LastHeader string
	Calls      []SelectCall // records every Select call in order
}

func (f *FakeFuzzySelector) Select(items []string, prompt, header string) (string, error) {
	f.LastItems = items
	f.LastPrompt = prompt
	f.LastHeader = header
	f.Calls = append(f.Calls, SelectCall{Items: slices.Clone(items), Prompt: prompt, Header: header})
	if f.Err != nil {
		return "", f.Err
	}
	if len(f.Selections) == 0 {
		return "", nil
	}
	sel := f.Selections[0]
	f.Selections = f.Selections[1:]
	return sel, nil
}
