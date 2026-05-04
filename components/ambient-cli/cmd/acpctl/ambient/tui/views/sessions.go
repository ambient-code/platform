package views

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// PhaseColor returns the display color for a session or agent phase.
//
//	pending              -> yellow (33)
//	running / active     -> orange (214)
//	succeeded / completed -> dim   (240)
//	failed               -> red    (31)
//	cancelled / idle     -> dim    (240)
func PhaseColor(phase string) lipgloss.Color {
	switch strings.ToLower(phase) {
	case "pending":
		return lipgloss.Color("33") // yellow
	case "running", "active":
		return lipgloss.Color("214") // orange
	case "succeeded", "completed":
		return lipgloss.Color("240") // dim
	case "failed":
		return lipgloss.Color("31") // red
	case "cancelled", "idle":
		return lipgloss.Color("240") // dim
	default:
		return lipgloss.Color("240") // dim
	}
}

// SessionColumns returns the column definitions for the session list view.
// Column order matches the TUI spec: ID, AGENT, PROJECT, PHASE, TRIGGERED BY, STARTED, DURATION.
func SessionColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 14},
		{Title: "NAME", Width: 15},
		{Title: "AGENT", Width: 12},
		{Title: "PROJECT", Width: 12},
		{Title: "PHASE", Width: 12},
		{Title: "STARTED", Width: 10},
		{Title: "DURATION", Width: 10},
	}
}

// SessionRow converts an SDK Session into a table row suitable for the session
// list view. The agentName parameter is the resolved display name for the
// session's AgentID — the caller is responsible for resolving agent ID to name
// (see Known N+1 Queries in the TUI spec). The now parameter is used to compute
// the relative STARTED column and running duration.
//
// ID is shown in short form (first 12 characters). DURATION is computed as
// CompletionTime - StartTime for completed sessions, now - StartTime for
// running sessions, or empty for pending sessions.
//
// The PHASE column value is rendered with lipgloss-embedded color so it
// displays correctly in the bubbles/table without conflicting with Cell style.
func SessionRow(s sdktypes.Session, agentName string, now time.Time) table.Row {
	// Short ID: first 12 characters.
	shortID := s.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}

	// STARTED: relative age since StartTime.
	started := ""
	if s.StartTime != nil {
		started = FormatAge(now.Sub(*s.StartTime))
	}

	// DURATION: completed = CompletionTime - StartTime,
	// running = now - StartTime, pending = empty.
	duration := ""
	if s.CompletionTime != nil && s.StartTime != nil {
		duration = FormatAge(s.CompletionTime.Sub(*s.StartTime))
	} else if s.StartTime != nil {
		// Session is still running — show elapsed time.
		duration = FormatAge(now.Sub(*s.StartTime))
	}

	return table.Row{
		shortID,
		s.Name,
		agentName,
		s.ProjectID,
		s.Phase,
		started,
		duration,
	}
}

// NewSessionTable creates a ResourceTable configured for the session list view.
// The scope parameter controls the title bar context — "all" for global view,
// an agent name for agent-scoped view, etc.
func NewSessionTable(scope string, style TableStyle) ResourceTable {
	return NewResourceTable("sessions", scope, SessionColumns(), style)
}
