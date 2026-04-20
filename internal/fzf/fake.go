package fzf

// FakeFuzzySelector is a configurable test double for FuzzySelector.
type FakeFuzzySelector struct {
	// Selections is consumed in order; each call pops the first element.
	Selections []string
	Err        error

	LastItems  []string
	LastPrompt string
	LastHeader string
}

func (f *FakeFuzzySelector) Select(items []string, prompt, header string) (string, error) {
	f.LastItems = items
	f.LastPrompt = prompt
	f.LastHeader = header
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
