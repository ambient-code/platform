package views

import (
	"time"

	"github.com/charmbracelet/bubbles/table"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// AgentColumns returns the column definitions for the agent list view.
// Column order matches the TUI spec: NAME, PROMPT, SESSION, PHASE, AGE.
func AgentColumns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "PROMPT", Width: 60},
		{Title: "SESSION", Width: 14},
		{Title: "PHASE", Width: 12},
		{Title: "AGE", Width: 8},
	}
}

// AgentRow converts an SDK Agent into a table row suitable for the agent list
// view. The now parameter is used to compute the relative AGE column.
//
// The PHASE column is left empty because populating it requires a secondary
// fetch of the agent's current session. The caller (model layer) is responsible
// for enriching rows with phase data after the fan-out fetch.
func AgentRow(a sdktypes.Agent, now time.Time) table.Row {
	age := ""
	if a.CreatedAt != nil {
		age = FormatAge(now.Sub(*a.CreatedAt))
	}

	session := "<none>"
	if a.CurrentSessionID != "" {
		session = a.CurrentSessionID
	}

	return table.Row{
		a.Name,
		TruncateString(a.Prompt, 60),
		session,
		"", // PHASE — requires secondary fetch; filled in by the model
		age,
	}
}

// NewAgentTable creates a ResourceTable configured for the agent list view.
// The scope parameter is the project name that the agent list is scoped to.
func NewAgentTable(scope string, style TableStyle) ResourceTable {
	return NewResourceTable("agents", scope, AgentColumns(), style)
}

// TruncateString truncates s to maxLen characters, appending an ellipsis if the
// string was shortened. If maxLen is less than 1, an empty string is returned.
// This helper is exported for reuse by other views that need column truncation.
func TruncateString(s string, maxLen int) string {
	if maxLen < 1 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}

	if maxLen <= 1 {
		return string(runes[:1])
	}

	return string(runes[:maxLen-1]) + "…"
}
