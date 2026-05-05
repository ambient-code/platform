package views

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/huh"

	sdktypes "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

// ScheduledSessionColumns returns the column definitions for the scheduled
// session list view.
func ScheduledSessionColumns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "SCHEDULE", Width: 16},
		{Title: "PROJECT", Width: 15},
		{Title: "SUSPENDED", Width: 10},
		{Title: "LAST RUN", Width: 12},
		{Title: "AGE", Width: 8},
	}
}

// ScheduledSessionRow converts a ScheduledSession into a table row suitable for
// the scheduled session list view.
func ScheduledSessionRow(ss sdktypes.ScheduledSession, now time.Time) table.Row {
	name := ss.Name

	suspended := "No"
	if !ss.Enabled {
		suspended = "Yes"
	}

	lastRun := ""
	if ss.LastRunAt != nil {
		lastRun = FormatAge(now.Sub(*ss.LastRunAt))
	}

	age := ""
	if ss.CreatedAt != nil {
		age = FormatAge(now.Sub(*ss.CreatedAt))
	}

	return table.Row{
		name,
		ss.Schedule,
		ss.ProjectID,
		suspended,
		lastRun,
		age,
	}
}

// NewScheduledSessionTable creates a ResourceTable configured for the scheduled
// session list view. The scope parameter controls the title bar context.
func NewScheduledSessionTable(scope string, style TableStyle) ResourceTable {
	return NewResourceTable("scheduledsessions", scope, ScheduledSessionColumns(), style)
}

// ScheduledSessionDetail returns detail lines for all fields of a
// ScheduledSession resource.
func ScheduledSessionDetail(ss sdktypes.ScheduledSession) []DetailLine {
	suspended := "No"
	if !ss.Enabled {
		suspended = "Yes"
	}

	lastRun := ""
	if ss.LastRunAt != nil {
		lastRun = ss.LastRunAt.Format(time.RFC3339)
	}

	nextRun := ""
	if ss.NextRunAt != nil {
		nextRun = ss.NextRunAt.Format(time.RFC3339)
	}

	createdAt := ""
	if ss.CreatedAt != nil {
		createdAt = ss.CreatedAt.Format(time.RFC3339)
	}

	updatedAt := ""
	if ss.UpdatedAt != nil {
		updatedAt = ss.UpdatedAt.Format(time.RFC3339)
	}

	timeout := ""
	if ss.Timeout != nil {
		timeout = fmt.Sprintf("%d", *ss.Timeout)
	}

	inactivityTimeout := ""
	if ss.InactivityTimeout != nil {
		inactivityTimeout = fmt.Sprintf("%d", *ss.InactivityTimeout)
	}

	stopOnRunFinished := ""
	if ss.StopOnRunFinished != nil {
		stopOnRunFinished = fmt.Sprintf("%v", *ss.StopOnRunFinished)
	}

	return []DetailLine{
		{Key: "ID", Value: ss.ID},
		{Key: "Name", Value: ss.Name},
		{Key: "Description", Value: ss.Description},
		{Key: "Project ID", Value: ss.ProjectID},
		{Key: "Agent ID", Value: ss.AgentID},
		{Key: "Schedule", Value: ss.Schedule},
		{Key: "Timezone", Value: ss.Timezone},
		{Key: "Suspended", Value: suspended},
		{Key: "Session Prompt", Value: ss.SessionPrompt},
		{Key: "Timeout", Value: timeout},
		{Key: "Inactivity Timeout", Value: inactivityTimeout},
		{Key: "Stop On Run Finished", Value: stopOnRunFinished},
		{Key: "Runner Type", Value: ss.RunnerType},
		{Key: "Last Run At", Value: lastRun},
		{Key: "Next Run At", Value: nextRun},
		{Key: "Created At", Value: createdAt},
		{Key: "Updated At", Value: updatedAt},
	}
}

// NewScheduledSessionForm creates a huh form for creating a new scheduled
// session. Agent selection is optional — a "None" option is prepended.
func NewScheduledSessionForm(
	displayName, schedule, description, sessionPrompt, timezone, agentID *string,
	agentOptions []huh.Option[string],
) *huh.Form {
	allAgentOpts := append([]huh.Option[string]{huh.NewOption("(none)", "")}, agentOptions...)
	fields := []huh.Field{
		huh.NewInput().
			Key("displayName").
			Title("Name").
			Placeholder("my-scheduled-session").
			Validate(huh.ValidateNotEmpty()).
			Value(displayName),
		huh.NewSelect[string]().
			Key("agent").
			Title("Agent (optional)").
			Options(allAgentOpts...).
			Value(agentID),
		huh.NewInput().
			Key("schedule").
			Title("Schedule (cron)").
			Placeholder("*/30 * * * *").
			Validate(huh.ValidateNotEmpty()).
			Value(schedule),
		huh.NewInput().
			Key("timezone").
			Title("Timezone").
			Placeholder("UTC").
			Value(timezone),
		huh.NewInput().
			Key("sessionPrompt").
			Title("Session Prompt").
			Placeholder("(optional)").
			Value(sessionPrompt),
		huh.NewInput().
			Key("description").
			Title("Description").
			Placeholder("(optional)").
			Value(description),
	}
	return huh.NewForm(
		huh.NewGroup(fields...),
	).WithTheme(ACPTheme()).WithShowHelp(false)
}
