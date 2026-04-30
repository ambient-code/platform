package views

import (
	"github.com/charmbracelet/bubbles/table"
)

// ContextColumns returns the column definitions for the context list view.
func ContextColumns() []table.Column {
	return []table.Column{
		{Title: "ACTIVE", Width: 6},
		{Title: "NAME", Width: 25},
		{Title: "SERVER", Width: 45},
		{Title: "PROJECT", Width: 20},
	}
}

// ContextRow converts a context entry into a table row.
func ContextRow(name, server, project string, active bool) table.Row {
	indicator := ""
	if active {
		indicator = "(*)"
	}
	return table.Row{indicator, name, server, project}
}

// NewContextTable creates a ResourceTable configured for the context list view.
func NewContextTable(style TableStyle) ResourceTable {
	return NewResourceTable("contexts", "all", ContextColumns(), style)
}
