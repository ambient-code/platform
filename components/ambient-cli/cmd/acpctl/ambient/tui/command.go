package tui

import (
	"sort"
	"strings"
)

// CommandKind identifies the type of command entered in command mode.
type CommandKind int

const (
	CmdProjects CommandKind = iota
	CmdAgents
	CmdSessions
	CmdInbox
	CmdMessages
	CmdContext
	CmdProject
	CmdAliases
	CmdScheduledSessions
	CmdQuit
	CmdUnknown
)

// Command represents a parsed command-mode input.
type Command struct {
	Kind CommandKind
	Arg  string // optional argument (context name, project name)
}

// AliasEntry describes a command and its aliases for the :aliases listing.
type AliasEntry struct {
	Command     string
	Aliases     []string
	Description string
}

// commandDef maps a canonical command name to its kind, aliases, and description.
type commandDef struct {
	kind        CommandKind
	aliases     []string
	description string
	// takesArg indicates the command accepts an optional argument that changes
	// its behavior (e.g. :ctx vs :ctx <name>).
	takesArg bool
}

// commandDefs is the authoritative list of commands. Order determines AliasTable output.
var commandDefs = []commandDef{
	{
		kind:        CmdProjects,
		aliases:     []string{"projects", "proj"},
		description: "Switch to project list",
	},
	{
		kind:        CmdAgents,
		aliases:     []string{"agents", "ag"},
		description: "Switch to agent list (current project)",
	},
	{
		kind:        CmdSessions,
		aliases:     []string{"sessions", "se"},
		description: "Switch to session list",
	},
	{
		kind:        CmdInbox,
		aliases:     []string{"inbox", "ib"},
		description: "Switch to inbox (requires agent context)",
	},
	{
		kind:        CmdMessages,
		aliases:     []string{"messages", "msg"},
		description: "Switch to message stream (requires session context)",
	},
	{
		kind:        CmdContext,
		aliases:     []string{"context", "ctx"},
		description: "List contexts (no arg) or switch context (with arg)",
		takesArg:    true,
	},
	{
		kind:        CmdProject,
		aliases:     []string{"project"},
		description: "Switch project within current context",
		takesArg:    true,
	},
	{
		kind:        CmdScheduledSessions,
		aliases:     []string{"scheduledsessions", "scheduledsession", "ss"},
		description: "Switch to scheduled sessions list (current project)",
	},
	{
		kind:        CmdAliases,
		aliases:     []string{"aliases"},
		description: "List all commands and aliases",
	},
	{
		kind:        CmdQuit,
		aliases:     []string{"q", "quit"},
		description: "Exit",
	},
}

// aliasToCommand maps every alias (including canonical names) to a commandDef.
var aliasToCommand map[string]*commandDef

func init() {
	aliasToCommand = make(map[string]*commandDef, len(commandDefs)*2)
	for i := range commandDefs {
		for _, alias := range commandDefs[i].aliases {
			aliasToCommand[alias] = &commandDefs[i]
		}
	}
}

// ParseCommand parses raw command-mode input (without the leading ':') and
// returns the parsed Command. Unrecognized input returns CmdUnknown.
//
// Special case: "proj <name>" is parsed as CmdProject (switch project),
// while "proj" alone is CmdProjects (list projects).
func ParseCommand(input string) Command {
	input = strings.TrimSpace(input)
	if input == "" {
		return Command{Kind: CmdUnknown}
	}

	// Split into command name and optional argument.
	parts := strings.SplitN(input, " ", 2)
	name := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(parts[1])
	}

	// Special case: "proj" is overloaded.
	// - "proj" with no arg → CmdProjects (list projects)
	// - "proj <name>" → CmdProject (switch project)
	if name == "proj" {
		if arg != "" {
			return Command{Kind: CmdProject, Arg: arg}
		}
		return Command{Kind: CmdProjects}
	}

	def, ok := aliasToCommand[name]
	if !ok {
		return Command{Kind: CmdUnknown}
	}

	// If the command takes an arg, pass it through. If it doesn't take an arg,
	// the arg is silently ignored (consistent with k9s behavior).
	if def.takesArg {
		return Command{Kind: def.kind, Arg: arg}
	}
	return Command{Kind: def.kind}
}

// allCommandNames returns a deduplicated, sorted list of all command aliases.
func allCommandNames() []string {
	names := make([]string, 0, len(aliasToCommand))
	for name := range aliasToCommand {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TabComplete returns completion suggestions for partial command-mode input.
// The partial string should not include the leading ':'.
//
// Completion behavior:
//   - If partial has no space, complete command names.
//   - If partial starts with "ctx" or "context" and has a space, complete context names.
//   - If partial starts with "project" or "proj" and has a space, complete project names.
//
// contexts and projects are the available names for argument completion.
// Returns suggestions sorted lexicographically.
func TabComplete(partial string, contexts []string, projects []string) []string {
	trimmed := strings.TrimSpace(partial)
	if trimmed == "" {
		// Show all command names when nothing typed yet.
		return allCommandNames()
	}

	// Check if we're completing an argument. A space anywhere in the input
	// (including trailing, e.g. "ctx ") means the user has moved past the
	// command name and is now entering an argument.
	spaceIdx := strings.IndexByte(partial, ' ')
	if spaceIdx >= 0 {
		cmdName := strings.ToLower(strings.TrimSpace(partial[:spaceIdx]))
		argPart := partial[spaceIdx+1:]
		argPartial := strings.TrimSpace(argPart)
		return completeArg(cmdName, argPartial, contexts, projects)
	}

	// Complete command name.
	lower := strings.ToLower(trimmed)
	var matches []string
	for _, name := range allCommandNames() {
		if strings.HasPrefix(name, lower) {
			matches = append(matches, name)
		}
	}
	return matches
}

// completeArg returns argument completions for the given command name.
func completeArg(cmdName, argPartial string, contexts, projects []string) []string {
	lower := strings.ToLower(argPartial)

	switch cmdName {
	case "context", "ctx":
		return filterPrefix(contexts, lower)
	case "project", "proj":
		return filterPrefix(projects, lower)
	default:
		return nil
	}
}

// filterPrefix returns items that have a case-insensitive prefix match with prefix.
// Results are sorted.
func filterPrefix(items []string, prefix string) []string {
	var matches []string
	for _, item := range items {
		if strings.HasPrefix(strings.ToLower(item), prefix) {
			matches = append(matches, item)
		}
	}
	sort.Strings(matches)
	return matches
}

// AliasTable returns the list of commands with their aliases and descriptions,
// suitable for rendering the :aliases output.
func AliasTable() []AliasEntry {
	entries := make([]AliasEntry, 0, len(commandDefs))
	for _, def := range commandDefs {
		canonical := def.aliases[0]
		var aliases []string
		if len(def.aliases) > 1 {
			aliases = make([]string, len(def.aliases)-1)
			copy(aliases, def.aliases[1:])
		}
		entries = append(entries, AliasEntry{
			Command:     canonical,
			Aliases:     aliases,
			Description: def.description,
		})
	}
	return entries
}
