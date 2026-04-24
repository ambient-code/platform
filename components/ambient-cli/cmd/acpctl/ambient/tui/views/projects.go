package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// ProjectColumns returns the column definitions for the project list view.
// Column order matches the TUI spec: NAME, DESCRIPTION, STATUS, AGE.
func ProjectColumns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 25},
		{Title: "DESCRIPTION", Width: 40},
		{Title: "STATUS", Width: 12},
		{Title: "AGE", Width: 8},
	}
}

// ProjectRow converts an SDK Project into a table row suitable for the project
// list view. The now parameter is used to compute the relative AGE column.
// Truncation of long values is handled by the table widget.
func ProjectRow(p sdktypes.Project, now time.Time) table.Row {
	age := ""
	if p.CreatedAt != nil {
		age = FormatAge(now.Sub(*p.CreatedAt))
	}

	return table.Row{
		p.Name,
		p.Description,
		p.Status,
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
