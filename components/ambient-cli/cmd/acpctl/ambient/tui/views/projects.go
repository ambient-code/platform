package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// ProjectColumns returns the column definitions for the project list view.
// Column order: NAME, DESCRIPTION, STATUS, AGENTS, SESSIONS, AGE.
func ProjectColumns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 25},
		{Title: "DESCRIPTION", Width: 40},
		{Title: "STATUS", Width: 12},
		{Title: "AGENTS", Width: 8},
		{Title: "SESSIONS", Width: 8},
		{Title: "AGE", Width: 8},
	}
}

// ProjectRow converts an SDK Project into a table row suitable for the project
// list view. The now parameter is used to compute the relative AGE column.
// agentCount and sessionCount are displayed as integers; a value of -1 renders
// as "-" to indicate counts have not been loaded yet.
// Truncation of long values is handled by the table widget.
func ProjectRow(p sdktypes.Project, now time.Time, agentCount, sessionCount int) table.Row {
	age := ""
	if p.CreatedAt != nil {
		age = FormatAge(now.Sub(*p.CreatedAt))
	}

	agents := "-"
	if agentCount >= 0 {
		agents = fmt.Sprintf("%d", agentCount)
	}

	sessions := "-"
	if sessionCount >= 0 {
		sessions = fmt.Sprintf("%d", sessionCount)
	}

	return table.Row{
		p.Name,
		p.Description,
		p.Status,
		agents,
		sessions,
		age,
	}
}

// FormatAge formats a duration as a compact relative time string suitable for
// table display. It picks the largest meaningful unit:
//
//	>=24h  → "3d"
//	>=1h   → "2h"
//	>=1m   → "5m"
//	<1m    → "30s"
//
// Negative durations are clamped to "0s".
func FormatAge(d time.Duration) string {
	if d < 0 {
		return "0s"
	}

	days := int(d.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}

	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}

	minutes := int(d.Minutes())
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}

	seconds := int(d.Seconds())
	return fmt.Sprintf("%ds", seconds)
}

// NewProjectTable creates a ResourceTable configured for the project list view.
// The table uses kind="projects" and scope="all" since the project list is
// always global (not scoped to another resource).
func NewProjectTable(style TableStyle) ResourceTable {
	return NewResourceTable("projects", "all", ProjectColumns(), style)
}
