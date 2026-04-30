package tui

import (
	"testing"
)

func TestParseFilter_BasicRegex(t *testing.T) {
	f, err := mustParse(t, "running")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Pattern == nil {
		t.Fatal("expected non-nil Pattern")
	}
	if f.Inverse {
		t.Error("expected Inverse=false")
	}
	if f.Label != nil {
		t.Error("expected Label=nil")
	}
	if f.Raw != "running" {
		t.Errorf("expected Raw=%q, got %q", "running", f.Raw)
	}
}

func TestParseFilter_CaseInsensitive(t *testing.T) {
	f, err := mustParse(t, "Running")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should match lowercase
	if !f.Pattern.MatchString("running") {
		t.Error("expected case-insensitive match for 'running'")
	}
	// Should match uppercase
	if !f.Pattern.MatchString("RUNNING") {
		t.Error("expected case-insensitive match for 'RUNNING'")
	}
	// Should match mixed case
	if !f.Pattern.MatchString("RuNnInG") {
		t.Error("expected case-insensitive match for 'RuNnInG'")
	}
}

func TestParseFilter_InverseRegex(t *testing.T) {
	f, err := mustParse(t, "!completed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Inverse {
		t.Error("expected Inverse=true")
	}
	if f.Pattern == nil {
		t.Fatal("expected non-nil Pattern")
	}
	if f.Label != nil {
		t.Error("expected Label=nil")
	}
}

func TestParseFilter_LabelFilter(t *testing.T) {
	f, err := mustParse(t, "-l env=prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Label == nil {
		t.Fatal("expected non-nil Label")
	}
	if f.Label.Key != "env" {
		t.Errorf("expected Key=%q, got %q", "env", f.Label.Key)
	}
	if f.Label.Value != "prod" {
		t.Errorf("expected Value=%q, got %q", "prod", f.Label.Value)
	}
	if f.Pattern != nil {
		t.Error("expected Pattern=nil for label filter")
	}
}

func TestParseFilter_LabelFilterNoSpace(t *testing.T) {
	f, err := mustParse(t, "-lenv=prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Label == nil {
		t.Fatal("expected non-nil Label")
	}
	if f.Label.Key != "env" {
		t.Errorf("expected Key=%q, got %q", "env", f.Label.Key)
	}
	if f.Label.Value != "prod" {
		t.Errorf("expected Value=%q, got %q", "prod", f.Label.Value)
	}
}

func TestParseFilter_LabelFilterEmptyValue(t *testing.T) {
	// -l key= is valid (empty value)
	f, err := mustParse(t, "-l key=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Label == nil {
		t.Fatal("expected non-nil Label")
	}
	if f.Label.Key != "key" {
		t.Errorf("expected Key=%q, got %q", "key", f.Label.Key)
	}
	if f.Label.Value != "" {
		t.Errorf("expected Value=%q, got %q", "", f.Label.Value)
	}
}

func TestParseFilter_LabelFilterMultipleEquals(t *testing.T) {
	// -l key=val=ue should split on first = only
	f, err := mustParse(t, "-l key=val=ue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Label == nil {
		t.Fatal("expected non-nil Label")
	}
	if f.Label.Key != "key" {
		t.Errorf("expected Key=%q, got %q", "key", f.Label.Key)
	}
	if f.Label.Value != "val=ue" {
		t.Errorf("expected Value=%q, got %q", "val=ue", f.Label.Value)
	}
}

func TestParseFilter_EmptyString(t *testing.T) {
	f, err := mustParse(t, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Pattern == nil {
		t.Fatal("expected non-nil Pattern for empty string")
	}
	// Empty regex matches everything
	if !f.Pattern.MatchString("anything") {
		t.Error("empty regex should match everything")
	}
}

func TestParseFilter_InverseEmptyString(t *testing.T) {
	f, err := mustParse(t, "!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Inverse {
		t.Error("expected Inverse=true")
	}
	if f.Pattern == nil {
		t.Fatal("expected non-nil Pattern")
	}
}

func TestParseFilter_InvalidRegex(t *testing.T) {
	// Invalid regex falls back to literal match via QuoteMeta.
	f, err := ParseFilter("[invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Pattern == nil {
		t.Fatal("expected non-nil pattern")
	}
}

func TestParseFilter_InvalidRegexInverse(t *testing.T) {
	// Invalid regex falls back to literal match via QuoteMeta.
	f, err := ParseFilter("![invalid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Inverse {
		t.Fatal("expected inverse flag")
	}
}

func TestParseFilter_SpecialRegexChars(t *testing.T) {
	// Valid regex with special characters
	f, err := mustParse(t, "be-agent\\.v[12]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Pattern.MatchString("be-agent.v1") {
		t.Error("expected match for 'be-agent.v1'")
	}
	if !f.Pattern.MatchString("be-agent.v2") {
		t.Error("expected match for 'be-agent.v2'")
	}
	if f.Pattern.MatchString("be-agentXv3") {
		t.Error("expected no match for 'be-agentXv3'")
	}
}

func TestParseFilter_LabelFilterErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty after -l", "-l "},
		{"no equals sign", "-l envprod"},
		{"empty key", "-l =prod"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilter(tt.input)
			if err == nil {
				t.Errorf("expected error for input %q", tt.input)
			}
		})
	}
}

func TestMatchRow_BasicRegex(t *testing.T) {
	f, _ := ParseFilter("running")

	tests := []struct {
		name    string
		columns []string
		want    bool
	}{
		{
			name:    "match in first column",
			columns: []string{"running", "agent-1", "proj-a"},
			want:    true,
		},
		{
			name:    "match in middle column",
			columns: []string{"agent-1", "running", "proj-a"},
			want:    true,
		},
		{
			name:    "match in last column",
			columns: []string{"agent-1", "proj-a", "running"},
			want:    true,
		},
		{
			name:    "no match",
			columns: []string{"agent-1", "completed", "proj-a"},
			want:    false,
		},
		{
			name:    "partial match",
			columns: []string{"agent-running-1", "proj-a"},
			want:    true,
		},
		{
			name:    "empty columns",
			columns: []string{},
			want:    false,
		},
		{
			name:    "case insensitive match",
			columns: []string{"RUNNING", "agent-1"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.MatchRow(tt.columns)
			if got != tt.want {
				t.Errorf("MatchRow(%v) = %v, want %v", tt.columns, got, tt.want)
			}
		})
	}
}

func TestMatchRow_InverseRegex(t *testing.T) {
	f, _ := ParseFilter("!completed")

	tests := []struct {
		name    string
		columns []string
		want    bool
	}{
		{
			name:    "hide matching row",
			columns: []string{"agent-1", "completed", "proj-a"},
			want:    false,
		},
		{
			name:    "show non-matching row",
			columns: []string{"agent-1", "running", "proj-a"},
			want:    true,
		},
		{
			name:    "hide partial match",
			columns: []string{"completed-yesterday", "proj-a"},
			want:    false,
		},
		{
			name:    "empty columns (no match, so not hidden)",
			columns: []string{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.MatchRow(tt.columns)
			if got != tt.want {
				t.Errorf("MatchRow(%v) = %v, want %v", tt.columns, got, tt.want)
			}
		})
	}
}

func TestMatchRow_LabelFilter(t *testing.T) {
	f, _ := ParseFilter("-l env=prod")

	// Label filters always return true — filtering is server-side
	tests := []struct {
		name    string
		columns []string
		want    bool
	}{
		{
			name:    "any row passes",
			columns: []string{"agent-1", "running", "proj-a"},
			want:    true,
		},
		{
			name:    "empty columns pass",
			columns: []string{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.MatchRow(tt.columns)
			if got != tt.want {
				t.Errorf("MatchRow(%v) = %v, want %v", tt.columns, got, tt.want)
			}
		})
	}
}

func TestMatchRow_NilPattern(t *testing.T) {
	// A filter with nil Pattern (shouldn't normally happen, but defensively) returns true
	f := &Filter{Raw: "test"}
	if !f.MatchRow([]string{"anything"}) {
		t.Error("nil Pattern should match everything")
	}
}

func TestIsLabelFilter(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"regex filter", "running", false},
		{"inverse filter", "!completed", false},
		{"label filter", "-l env=prod", true},
		{"empty filter", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseFilter(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := f.IsLabelFilter()
			if got != tt.want {
				t.Errorf("IsLabelFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"basic regex", "running", "running"},
		{"inverse regex", "!completed", "!completed"},
		{"label filter", "-l env=prod", "-l env=prod"},
		{"empty", "", ""},
		{"special chars", "be-agent\\.v1", "be-agent\\.v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseFilter(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := f.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMatchRow_MultipleColumns(t *testing.T) {
	f, _ := ParseFilter("prod")

	// Match should check across ALL columns
	row := []string{"agent-prod-1", "running", "production", "2h"}
	if !f.MatchRow(row) {
		t.Error("expected match when multiple columns contain the pattern")
	}

	// No match in any column
	row = []string{"agent-dev-1", "running", "development", "2h"}
	if f.MatchRow(row) {
		t.Error("expected no match when no column contains the pattern")
	}
}

func TestMatchRow_RegexPattern(t *testing.T) {
	f, _ := ParseFilter("^agent-[0-9]+$")

	tests := []struct {
		name    string
		columns []string
		want    bool
	}{
		{
			name:    "full match",
			columns: []string{"agent-123"},
			want:    true,
		},
		{
			name:    "no match — letters",
			columns: []string{"agent-abc"},
			want:    false,
		},
		{
			name:    "no match — prefix",
			columns: []string{"my-agent-123"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.MatchRow(tt.columns)
			if got != tt.want {
				t.Errorf("MatchRow(%v) = %v, want %v", tt.columns, got, tt.want)
			}
		})
	}
}

// mustParse is a test helper that calls ParseFilter and returns the result.
func mustParse(t *testing.T, input string) (*Filter, error) {
	t.Helper()
	return ParseFilter(input)
}
