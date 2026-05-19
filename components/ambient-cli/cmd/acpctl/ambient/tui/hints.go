package tui

import (
	"github.com/ambient-code/platform/components/ambient-cli/cmd/acpctl/ambient/tui/views"
)

// ViewHints holds the keyboard shortcut definitions for a single view.
// This is the single source of truth for both header hints and the help overlay.
type ViewHints struct {
	Resource   []views.HelpEntry
	General    []views.HelpEntry
	Navigation []views.HelpEntry
}

// defaultGeneral returns the general hints shared by most table views.
func defaultGeneral() []views.HelpEntry {
	return []views.HelpEntry{
		{Key: ":", Action: "Command"},
		{Key: "/", Action: "Filter"},
		{Key: "?", Action: "Help"},
		{Key: "c", Action: "Copy ID"},
		{Key: "j/k", Action: "Up/Down"},
		{Key: "shift-n", Action: "Sort Name"},
		{Key: "shift-a", Action: "Sort Age"},
	}
}

// viewHintRegistry maps view names to their hint definitions.
var viewHintRegistry = map[string]ViewHints{
	"projects": {
		Resource: []views.HelpEntry{
			{Key: "d", Action: "Describe"},
			{Key: "e", Action: "Edit"},
			{Key: "n", Action: "New"},
			{Key: "ctrl-d", Action: "Delete"},
		},
		Navigation: []views.HelpEntry{
			{Key: "Enter", Action: "Drill into agents"},
			{Key: "q", Action: "Quit"},
		},
	},
	"agents": {
		Resource: []views.HelpEntry{
			{Key: "s", Action: "Start"},
			{Key: "x", Action: "Stop"},
			{Key: "i", Action: "Inbox"},
			{Key: "d", Action: "Describe"},
			{Key: "e", Action: "Edit"},
			{Key: "l", Action: "Logs"},
			{Key: "n", Action: "New"},
			{Key: "y", Action: "JSON"},
			{Key: "ctrl-d", Action: "Delete"},
		},
		Navigation: []views.HelpEntry{
			{Key: "Enter", Action: "Drill into sessions"},
			{Key: "Esc", Action: "Back to projects"},
			{Key: "q", Action: "Back"},
			{Key: "0-9", Action: "Switch project"},
		},
	},
	"sessions": {
		Resource: []views.HelpEntry{
			{Key: "d", Action: "Describe"},
			{Key: "e", Action: "Edit"},
			{Key: "l", Action: "Logs"},
			{Key: "m", Action: "Send (via msgs)"},
			{Key: "n", Action: "New"},
			{Key: "x", Action: "Interrupt"},
			{Key: "y", Action: "JSON"},
			{Key: "ctrl-d", Action: "Delete"},
		},
		Navigation: []views.HelpEntry{
			{Key: "Enter", Action: "Drill into messages"},
			{Key: "Esc", Action: "Back to agents"},
			{Key: "q", Action: "Back"},
			{Key: "0-9", Action: "Switch project"},
		},
	},
	"inbox": {
		Resource: []views.HelpEntry{
			{Key: "r", Action: "Mark Read"},
			{Key: "ctrl-d", Action: "Delete"},
		},
		Navigation: []views.HelpEntry{
			{Key: "Enter", Action: "View body"},
			{Key: "Esc", Action: "Back to agents"},
			{Key: "q", Action: "Back"},
		},
	},
	"messages": {
		Resource: []views.HelpEntry{
			{Key: "s", Action: "Autoscroll"},
			{Key: "r", Action: "Raw"},
			{Key: "p", Action: "Pretty"},
			{Key: "t", Action: "Timestamps"},
			{Key: "m", Action: "Compose"},
			{Key: "c", Action: "Copy"},
			{Key: "x", Action: "Interrupt"},
			{Key: "shift-g", Action: "Bottom"},
			{Key: "g", Action: "Top"},
		},
		General: []views.HelpEntry{
			{Key: ":", Action: "Command"},
			{Key: "?", Action: "Help"},
		},
		Navigation: []views.HelpEntry{
			{Key: "Esc", Action: "Back to sessions"},
			{Key: "q", Action: "Back"},
		},
	},
	"scheduledsessions": {
		Resource: []views.HelpEntry{
			{Key: "d", Action: "Describe"},
			{Key: "e", Action: "Edit"},
			{Key: "n", Action: "New"},
			{Key: "s", Action: "Suspend/Resume"},
			{Key: "t", Action: "Trigger"},
			{Key: "y", Action: "JSON"},
			{Key: "ctrl-d", Action: "Delete"},
		},
		General: defaultGeneral(),
		Navigation: []views.HelpEntry{
			{Key: "Enter", Action: "Show detail"},
			{Key: "Esc", Action: "Back"},
			{Key: "q", Action: "Back"},
		},
	},
	"contexts": {
		Resource: []views.HelpEntry{},
		Navigation: []views.HelpEntry{
			{Key: "Enter", Action: "Switch context"},
			{Key: "Esc", Action: "Back"},
			{Key: "q", Action: "Back"},
		},
	},
	"detail": {
		Resource: []views.HelpEntry{
			{Key: "c", Action: "Copy value"},
			{Key: "j/k", Action: "Scroll"},
		},
		General: []views.HelpEntry{
			{Key: "?", Action: "Help"},
		},
		Navigation: []views.HelpEntry{
			{Key: "Esc", Action: "Back"},
			{Key: "q", Action: "Back"},
		},
	},
}

// hintsForView returns the ViewHints for a given view name.
// Views that don't override General get the default table-view general hints.
func hintsForView(viewName string) ViewHints {
	h, ok := viewHintRegistry[viewName]
	if !ok {
		return ViewHints{}
	}
	if h.General == nil {
		h.General = defaultGeneral()
	}
	return h
}
