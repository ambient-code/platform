package tui

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// LabelFilter represents a server-side label filter parsed from /-l key=val syntax.
type LabelFilter struct {
	Key   string
	Value string
}

// Filter represents a parsed filter expression from the TUI filter bar.
// Filters are entered via / in the TUI and support three syntaxes:
//   - /term        — case-insensitive regex match across all visible columns
//   - /!term       — inverse regex (hide matching rows)
//   - /-l key=val  — server-side label filter (@> containment)
type Filter struct {
	Raw     string         // original input string (without leading /)
	Pattern *regexp.Regexp // compiled regex (nil for label filters)
	Inverse bool           // true for /! filters
	Label   *LabelFilter   // non-nil for /-l filters
}

// ParseFilter parses the raw filter string (without the leading /).
// It returns a compiled Filter or an error if the regex is invalid.
//
// Examples:
//
//	ParseFilter("running")       → regex filter matching "running"
//	ParseFilter("!completed")    → inverse regex hiding "completed"
//	ParseFilter("-l env=prod")   → label filter {Key: "env", Value: "prod"}
func ParseFilter(input string) (*Filter, error) {
	f := &Filter{Raw: input}

	// Label filter: -l key=val
	if rest, ok := strings.CutPrefix(input, "-l "); ok {
		return parseLabelFilter(f, rest)
	}
	if rest, ok := strings.CutPrefix(input, "-l"); ok && len(rest) > 0 {
		return parseLabelFilter(f, rest)
	}

	// Inverse filter: !term
	if strings.HasPrefix(input, "!") {
		f.Inverse = true
		input = strings.TrimPrefix(input, "!")
	}

	// Empty pattern after stripping prefix
	if input == "" {
		// An empty regex matches everything, which is valid.
		// For inverse, this hides everything — unusual but not an error.
		f.Pattern = regexp.MustCompile("(?i)")
		return f, nil
	}

	// Compile as case-insensitive regex, falling back to literal match on invalid regex.
	re, err := regexp.Compile("(?i)" + input)
	if err != nil {
		re = regexp.MustCompile("(?i)" + regexp.QuoteMeta(input))
	}
	f.Pattern = re

	return f, nil
}

// parseLabelFilter parses the key=val portion of a -l filter.
func parseLabelFilter(f *Filter, kv string) (*Filter, error) {
	kv = strings.TrimSpace(kv)
	if kv == "" {
		return nil, fmt.Errorf("label filter requires key=value, got empty string")
	}

	eqIdx := strings.Index(kv, "=")
	if eqIdx < 0 {
		return nil, fmt.Errorf("label filter requires key=value format, got %q", kv)
	}

	key := kv[:eqIdx]
	value := kv[eqIdx+1:]

	if key == "" {
		return nil, fmt.Errorf("label filter key must not be empty")
	}

	f.Label = &LabelFilter{
		Key:   key,
		Value: value,
	}
	return f, nil
}

// MatchRow returns true if the row matches the filter.
//
// For regex filters, the row matches if ANY column contains a match.
// For inverse filters, the result is negated (rows that match are hidden).
// For label filters, MatchRow always returns true — label filtering is
// performed server-side, not client-side.
func (f *Filter) MatchRow(columns []string) bool {
	// Label filters are server-side only; all rows pass client-side filtering.
	if f.Label != nil {
		return true
	}

	if f.Pattern == nil {
		return true
	}

	matched := slices.ContainsFunc(columns, func(col string) bool {
		return f.Pattern.MatchString(col)
	})

	if f.Inverse {
		return !matched
	}
	return matched
}

// IsLabelFilter returns true if this filter is a server-side label filter.
func (f *Filter) IsLabelFilter() bool {
	return f.Label != nil
}

// String returns a human-readable representation of the filter for display
// in the TUI status line.
func (f *Filter) String() string {
	if f.Label != nil {
		return fmt.Sprintf("-l %s=%s", f.Label.Key, f.Label.Value)
	}
	if f.Inverse {
		return "!" + stripCaseInsensitivePrefix(f.Raw)
	}
	return f.Raw
}

// stripCaseInsensitivePrefix removes the leading "!" from the raw string
// if present, since String() adds it back explicitly for inverse filters.
// This avoids double-prefixing when Raw already starts with "!".
func stripCaseInsensitivePrefix(raw string) string {
	return strings.TrimPrefix(raw, "!")
}
