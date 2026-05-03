package sensitive

import "strings"

// Filter performs simple case-insensitive substring matching against a list of sensitive words.
type Filter struct {
	words []string
}

// New creates a Filter with the given word list. Words are lowercased for case-insensitive matching.
func New(words []string) *Filter {
	lower := make([]string, len(words))
	for i, w := range words {
		lower[i] = strings.ToLower(w)
	}
	return &Filter{words: lower}
}

// Contains returns true if the content contains any sensitive word (case-insensitive, after trimming).
func (f *Filter) Contains(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	for _, w := range f.words {
		if strings.Contains(lower, w) {
			return true
		}
	}
	return false
}
