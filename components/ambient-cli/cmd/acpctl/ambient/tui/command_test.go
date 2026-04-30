package tui

import (
	"slices"
	"testing"
)

func TestParseCommand_FullNames(t *testing.T) {
	tests := []struct {
		input string
		kind  CommandKind
		arg   string
	}{
		{"projects", CmdProjects, ""},
		{"agents", CmdAgents, ""},
		{"sessions", CmdSessions, ""},
		{"inbox", CmdInbox, ""},
		{"messages", CmdMessages, ""},
		{"context", CmdContext, ""},
		{"project my-proj", CmdProject, "my-proj"},
		{"aliases", CmdAliases, ""},
		{"q", CmdQuit, ""},
		{"quit", CmdQuit, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := ParseCommand(tt.input)
			if cmd.Kind != tt.kind {
				t.Errorf("ParseCommand(%q).Kind = %d, want %d", tt.input, cmd.Kind, tt.kind)
			}
			if cmd.Arg != tt.arg {
				t.Errorf("ParseCommand(%q).Arg = %q, want %q", tt.input, cmd.Arg, tt.arg)
			}
		})
	}
}

func TestParseCommand_Aliases(t *testing.T) {
	tests := []struct {
		input string
		kind  CommandKind
		arg   string
	}{
		{"ag", CmdAgents, ""},
		{"se", CmdSessions, ""},
		{"ib", CmdInbox, ""},
		{"msg", CmdMessages, ""},
		{"ctx", CmdContext, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := ParseCommand(tt.input)
			if cmd.Kind != tt.kind {
				t.Errorf("ParseCommand(%q).Kind = %d, want %d", tt.input, cmd.Kind, tt.kind)
			}
			if cmd.Arg != tt.arg {
				t.Errorf("ParseCommand(%q).Arg = %q, want %q", tt.input, cmd.Arg, tt.arg)
			}
		})
	}
}

func TestParseCommand_ProjOverload(t *testing.T) {
	// :proj with no arg → CmdProjects (list projects)
	cmd := ParseCommand("proj")
	if cmd.Kind != CmdProjects {
		t.Errorf("ParseCommand(\"proj\").Kind = %d, want CmdProjects (%d)", cmd.Kind, CmdProjects)
	}
	if cmd.Arg != "" {
		t.Errorf("ParseCommand(\"proj\").Arg = %q, want empty", cmd.Arg)
	}

	// :proj <name> → CmdProject (switch project)
	cmd = ParseCommand("proj my-project")
	if cmd.Kind != CmdProject {
		t.Errorf("ParseCommand(\"proj my-project\").Kind = %d, want CmdProject (%d)", cmd.Kind, CmdProject)
	}
	if cmd.Arg != "my-project" {
		t.Errorf("ParseCommand(\"proj my-project\").Arg = %q, want \"my-project\"", cmd.Arg)
	}
}

func TestParseCommand_WithArguments(t *testing.T) {
	tests := []struct {
		input string
		kind  CommandKind
		arg   string
	}{
		{"context staging", CmdContext, "staging"},
		{"ctx staging", CmdContext, "staging"},
		{"ctx local", CmdContext, "local"},
		{"project my-proj", CmdProject, "my-proj"},
		{"context staging.ambient.io", CmdContext, "staging.ambient.io"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := ParseCommand(tt.input)
			if cmd.Kind != tt.kind {
				t.Errorf("ParseCommand(%q).Kind = %d, want %d", tt.input, cmd.Kind, tt.kind)
			}
			if cmd.Arg != tt.arg {
				t.Errorf("ParseCommand(%q).Arg = %q, want %q", tt.input, cmd.Arg, tt.arg)
			}
		})
	}
}

func TestParseCommand_ContextNoArg(t *testing.T) {
	// :context with no arg lists contexts
	cmd := ParseCommand("context")
	if cmd.Kind != CmdContext {
		t.Errorf("ParseCommand(\"context\").Kind = %d, want CmdContext (%d)", cmd.Kind, CmdContext)
	}
	if cmd.Arg != "" {
		t.Errorf("ParseCommand(\"context\").Arg = %q, want empty", cmd.Arg)
	}
}

func TestParseCommand_Unknown(t *testing.T) {
	tests := []string{
		"foobar",
		"nonexistent",
		"sesions",  // misspelled
		"agentss",  // extra s
		"Projects", // verify case insensitivity works
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			cmd := ParseCommand(input)
			// "Projects" should be recognized (case insensitive)
			if input == "Projects" {
				if cmd.Kind != CmdProjects {
					t.Errorf("ParseCommand(%q).Kind = %d, want CmdProjects", input, cmd.Kind)
				}
				return
			}
			if cmd.Kind != CmdUnknown {
				t.Errorf("ParseCommand(%q).Kind = %d, want CmdUnknown (%d)", input, cmd.Kind, CmdUnknown)
			}
		})
	}
}

func TestParseCommand_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		kind  CommandKind
	}{
		{"PROJECTS", CmdProjects},
		{"Projects", CmdProjects},
		{"AG", CmdAgents},
		{"Ctx", CmdContext},
		{"QUIT", CmdQuit},
		{"Q", CmdQuit},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cmd := ParseCommand(tt.input)
			if cmd.Kind != tt.kind {
				t.Errorf("ParseCommand(%q).Kind = %d, want %d", tt.input, cmd.Kind, tt.kind)
			}
		})
	}
}

func TestParseCommand_EdgeCases(t *testing.T) {
	// Empty input
	cmd := ParseCommand("")
	if cmd.Kind != CmdUnknown {
		t.Errorf("ParseCommand(\"\").Kind = %d, want CmdUnknown", cmd.Kind)
	}

	// Whitespace only
	cmd = ParseCommand("   ")
	if cmd.Kind != CmdUnknown {
		t.Errorf("ParseCommand(\"   \").Kind = %d, want CmdUnknown", cmd.Kind)
	}

	// Leading whitespace
	cmd = ParseCommand("  projects")
	if cmd.Kind != CmdProjects {
		t.Errorf("ParseCommand(\"  projects\").Kind = %d, want CmdProjects", cmd.Kind)
	}

	// Trailing whitespace
	cmd = ParseCommand("agents  ")
	if cmd.Kind != CmdAgents {
		t.Errorf("ParseCommand(\"agents  \").Kind = %d, want CmdAgents", cmd.Kind)
	}

	// Extra spaces between command and arg
	cmd = ParseCommand("ctx   staging")
	if cmd.Kind != CmdContext {
		t.Errorf("ParseCommand(\"ctx   staging\").Kind = %d, want CmdContext", cmd.Kind)
	}
	if cmd.Arg != "staging" {
		t.Errorf("ParseCommand(\"ctx   staging\").Arg = %q, want \"staging\"", cmd.Arg)
	}
}

func TestTabComplete_CommandNames(t *testing.T) {
	tests := []struct {
		partial string
		want    []string
	}{
		// Partial "s" matches sessions and scheduledsessions
		{"s", []string{"scheduledsession", "scheduledsessions", "se", "sessions", "ss"}},
		// Partial "a" matches agents, ag, aliases
		{"a", []string{"ag", "agents", "aliases"}},
		// Partial "q" matches q, quit
		{"q", []string{"q", "quit"}},
		// Partial "in" matches inbox
		{"in", []string{"inbox"}},
		// Partial "con" matches context
		{"con", []string{"context"}},
		// Partial "pro" matches project, projects, proj
		{"pro", []string{"proj", "project", "projects"}},
		// Exact match still returned
		{"sessions", []string{"sessions"}},
		// No match
		{"xyz", nil},
	}

	for _, tt := range tests {
		t.Run(tt.partial, func(t *testing.T) {
			got := TabComplete(tt.partial, nil, nil)
			if !stringSliceEqual(got, tt.want) {
				t.Errorf("TabComplete(%q, nil, nil) = %v, want %v", tt.partial, got, tt.want)
			}
		})
	}
}

func TestTabComplete_EmptyInput(t *testing.T) {
	got := TabComplete("", nil, nil)
	// Should return all command names
	if len(got) == 0 {
		t.Error("TabComplete(\"\", nil, nil) returned empty, want all command names")
	}
	// Verify it contains known commands
	found := map[string]bool{}
	for _, name := range got {
		found[name] = true
	}
	for _, expected := range []string{"projects", "agents", "sessions", "inbox", "messages", "context", "ctx", "project", "proj", "aliases", "q", "quit", "ag", "se", "ib", "msg", "scheduledsessions", "scheduledsession", "ss"} {
		if !found[expected] {
			t.Errorf("TabComplete(\"\") missing %q", expected)
		}
	}
}

func TestTabComplete_ContextNames(t *testing.T) {
	contexts := []string{"local", "staging", "staging.ambient.io", "prod"}

	tests := []struct {
		partial string
		want    []string
	}{
		{"ctx ", []string{"local", "prod", "staging", "staging.ambient.io"}},
		{"ctx s", []string{"staging", "staging.ambient.io"}},
		{"ctx l", []string{"local"}},
		{"ctx p", []string{"prod"}},
		{"ctx x", nil},
		{"context ", []string{"local", "prod", "staging", "staging.ambient.io"}},
		{"context sta", []string{"staging", "staging.ambient.io"}},
	}

	for _, tt := range tests {
		t.Run(tt.partial, func(t *testing.T) {
			got := TabComplete(tt.partial, contexts, nil)
			if !stringSliceEqual(got, tt.want) {
				t.Errorf("TabComplete(%q, contexts, nil) = %v, want %v", tt.partial, got, tt.want)
			}
		})
	}
}

func TestTabComplete_ProjectNames(t *testing.T) {
	projects := []string{"ambient-platform", "my-proj", "demo"}

	tests := []struct {
		partial string
		want    []string
	}{
		{"project ", []string{"ambient-platform", "demo", "my-proj"}},
		{"project a", []string{"ambient-platform"}},
		{"project m", []string{"my-proj"}},
		{"proj ", []string{"ambient-platform", "demo", "my-proj"}},
		{"proj d", []string{"demo"}},
		{"project x", nil},
	}

	for _, tt := range tests {
		t.Run(tt.partial, func(t *testing.T) {
			got := TabComplete(tt.partial, nil, projects)
			if !stringSliceEqual(got, tt.want) {
				t.Errorf("TabComplete(%q, nil, projects) = %v, want %v", tt.partial, got, tt.want)
			}
		})
	}
}

func TestTabComplete_NonArgCommand(t *testing.T) {
	// Tab-completing after a non-arg command should return nothing
	got := TabComplete("agents ", nil, nil)
	if got != nil {
		t.Errorf("TabComplete(\"agents \", nil, nil) = %v, want nil", got)
	}

	got = TabComplete("q ", nil, nil)
	if got != nil {
		t.Errorf("TabComplete(\"q \", nil, nil) = %v, want nil", got)
	}
}

func TestTabComplete_CaseInsensitive(t *testing.T) {
	contexts := []string{"Local", "Staging"}

	got := TabComplete("CTX ", contexts, nil)
	if !stringSliceEqual(got, []string{"Local", "Staging"}) {
		t.Errorf("TabComplete(\"CTX \", contexts, nil) = %v, want [Local Staging]", got)
	}

	got = TabComplete("S", nil, nil)
	if !stringSliceEqual(got, []string{"scheduledsession", "scheduledsessions", "se", "sessions", "ss"}) {
		t.Errorf("TabComplete(\"S\", nil, nil) = %v, want [scheduledsession scheduledsessions se sessions ss]", got)
	}
}

func TestAliasTable(t *testing.T) {
	entries := AliasTable()

	if len(entries) == 0 {
		t.Fatal("AliasTable() returned empty")
	}

	// Verify expected commands are present
	found := map[string]bool{}
	for _, entry := range entries {
		found[entry.Command] = true

		// Every entry should have a description
		if entry.Description == "" {
			t.Errorf("AliasTable entry %q has empty description", entry.Command)
		}
	}

	expected := []string{"projects", "agents", "sessions", "inbox", "messages", "context", "project", "aliases", "q"}
	for _, cmd := range expected {
		if !found[cmd] {
			t.Errorf("AliasTable() missing command %q", cmd)
		}
	}

	// Verify specific alias mappings
	for _, entry := range entries {
		switch entry.Command {
		case "agents":
			if !containsString(entry.Aliases, "ag") {
				t.Errorf("agents entry missing alias \"ag\", got %v", entry.Aliases)
			}
		case "sessions":
			if !containsString(entry.Aliases, "se") {
				t.Errorf("sessions entry missing alias \"se\", got %v", entry.Aliases)
			}
		case "context":
			if !containsString(entry.Aliases, "ctx") {
				t.Errorf("context entry missing alias \"ctx\", got %v", entry.Aliases)
			}
		case "q":
			if !containsString(entry.Aliases, "quit") {
				t.Errorf("q entry missing alias \"quit\", got %v", entry.Aliases)
			}
		}
	}
}

func TestAliasTable_NoDuplicateCommands(t *testing.T) {
	entries := AliasTable()
	seen := map[string]bool{}
	for _, entry := range entries {
		if seen[entry.Command] {
			t.Errorf("AliasTable() has duplicate command %q", entry.Command)
		}
		seen[entry.Command] = true
	}
}

// stringSliceEqual compares two string slices for equality (nil and empty are different).
func stringSliceEqual(a, b []string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// containsString checks if a string slice contains a value.
func containsString(slice []string, val string) bool {
	return slices.Contains(slice, val)
}
