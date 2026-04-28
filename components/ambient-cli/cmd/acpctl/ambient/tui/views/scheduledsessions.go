package views

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/huh"
)

// ScheduledSession mirrors the backend ScheduledSession response type.
// Defined locally because the backend types are not importable from the CLI module.
type ScheduledSession struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	CreationTimestamp string            `json:"creationTimestamp"`
	Schedule          string            `json:"schedule"`
	Suspend           bool              `json:"suspend"`
	DisplayName       string            `json:"displayName"`
	SessionTemplate   json.RawMessage   `json:"sessionTemplate"`
	LastScheduleTime  *string           `json:"lastScheduleTime,omitempty"`
	ActiveCount       int               `json:"activeCount"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	ReuseLastSession  bool              `json:"reuseLastSession"`
}

// CreateScheduledSessionRequest is the request body for creating a scheduled session.
type CreateScheduledSessionRequest struct {
	Schedule        string                 `json:"schedule"`
	DisplayName     string                 `json:"displayName"`
	SessionTemplate map[string]interface{} `json:"sessionTemplate"`
	Suspend         bool                   `json:"suspend,omitempty"`
}

// ScheduledSessionColumns returns the column definitions for the scheduled
// session list view.
func ScheduledSessionColumns() []table.Column {
	return []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "SCHEDULE", Width: 16},
		{Title: "PROJECT", Width: 15},
		{Title: "SUSPENDED", Width: 10},
		{Title: "ACTIVE", Width: 7},
		{Title: "LAST RUN", Width: 10},
		{Title: "AGE", Width: 8},
	}
}

// ScheduledSessionRow converts a ScheduledSession into a table row suitable for
// the scheduled session list view.
func ScheduledSessionRow(ss ScheduledSession, now time.Time) table.Row {
	name := ss.DisplayName
	if name == "" {
		name = ss.Name
	}

	suspended := "No"
	if ss.Suspend {
		suspended = "Yes"
	}

	lastRun := ""
	if ss.LastScheduleTime != nil && *ss.LastScheduleTime != "" {
		if t, err := time.Parse(time.RFC3339, *ss.LastScheduleTime); err == nil {
			lastRun = FormatAge(now.Sub(t))
		}
	}

	age := ""
	if ss.CreationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, ss.CreationTimestamp); err == nil {
			age = FormatAge(now.Sub(t))
		}
	}

	return table.Row{
		name,
		ss.Schedule,
		ss.Namespace,
		suspended,
		fmt.Sprintf("%d", ss.ActiveCount),
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
func ScheduledSessionDetail(ss ScheduledSession) []DetailLine {
	suspended := "No"
	if ss.Suspend {
		suspended = "Yes"
	}
	reuseLastSession := "No"
	if ss.ReuseLastSession {
		reuseLastSession = "Yes"
	}

	lastRun := ""
	if ss.LastScheduleTime != nil {
		lastRun = *ss.LastScheduleTime
	}

	templateJSON := ""
	if len(ss.SessionTemplate) > 0 {
		var obj interface{}
		if err := json.Unmarshal(ss.SessionTemplate, &obj); err == nil {
			if data, err := json.MarshalIndent(obj, "", "  "); err == nil {
				templateJSON = string(data)
			}
		}
	}

	labelsJSON := ""
	if len(ss.Labels) > 0 {
		if data, err := json.MarshalIndent(ss.Labels, "", "  "); err == nil {
			labelsJSON = string(data)
		}
	}

	annotationsJSON := ""
	if len(ss.Annotations) > 0 {
		if data, err := json.MarshalIndent(ss.Annotations, "", "  "); err == nil {
			annotationsJSON = string(data)
		}
	}

	return []DetailLine{
		{Key: "Name", Value: ss.Name},
		{Key: "Display Name", Value: ss.DisplayName},
		{Key: "Namespace", Value: ss.Namespace},
		{Key: "Schedule", Value: ss.Schedule},
		{Key: "Suspended", Value: suspended},
		{Key: "Reuse Last Session", Value: reuseLastSession},
		{Key: "Active Count", Value: fmt.Sprintf("%d", ss.ActiveCount)},
		{Key: "Last Schedule Time", Value: lastRun},
		{Key: "Created At", Value: ss.CreationTimestamp},
		{Key: "Labels", Value: labelsJSON},
		{Key: "Annotations", Value: annotationsJSON},
		{Key: "Session Template", Value: templateJSON},
	}
}

// NewScheduledSessionForm creates a huh form for creating a new scheduled session.
func NewScheduledSessionForm(displayName, schedule *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("displayName").
				Title("Display Name").
				Placeholder("my-scheduled-session").
				Validate(huh.ValidateNotEmpty()).
				Value(displayName),
			huh.NewInput().
				Key("schedule").
				Title("Schedule (cron)").
				Placeholder("*/30 * * * *").
				Validate(huh.ValidateNotEmpty()).
				Value(schedule),
		),
	).WithTheme(ACPTheme()).WithShowHelp(false)
}
