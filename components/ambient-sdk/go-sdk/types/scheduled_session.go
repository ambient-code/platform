package types

import (
	"errors"
	"fmt"
	"time"
)

type ScheduledSession struct {
	ObjectReference

	AgentID           string     `json:"agent_id,omitempty"`
	Description       string     `json:"description,omitempty"`
	Enabled           bool       `json:"enabled,omitempty"`
	InactivityTimeout *int32     `json:"inactivity_timeout,omitempty"`
	LastRunAt         *time.Time `json:"last_run_at,omitempty"`
	Name              string     `json:"name"`
	NextRunAt         *time.Time `json:"next_run_at,omitempty"`
	ProjectID         string     `json:"project_id,omitempty"`
	RunnerType        string     `json:"runner_type,omitempty"`
	Schedule          string     `json:"schedule,omitempty"`
	SessionPrompt     string     `json:"session_prompt,omitempty"`
	StopOnRunFinished *bool      `json:"stop_on_run_finished,omitempty"`
	Timeout           *int32     `json:"timeout,omitempty"`
	Timezone          string     `json:"timezone,omitempty"`
}

type ScheduledSessionList struct {
	ListMeta
	Items []ScheduledSession `json:"items"`
}

func (l *ScheduledSessionList) GetItems() []ScheduledSession { return l.Items }
func (l *ScheduledSessionList) GetTotal() int                { return l.Total }
func (l *ScheduledSessionList) GetPage() int                 { return l.Page }
func (l *ScheduledSessionList) GetSize() int                 { return l.Size }

// ScheduledSessionPatch is the request body for a PATCH operation.
// Only set fields that should be changed; omitted (nil) fields are left unchanged.
type ScheduledSessionPatch struct {
	Name              *string `json:"name,omitempty"`
	Description       *string `json:"description,omitempty"`
	AgentID           *string `json:"agent_id,omitempty"`
	Schedule          *string `json:"schedule,omitempty"`
	Timezone          *string `json:"timezone,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
	SessionPrompt     *string `json:"session_prompt,omitempty"`
	Timeout           *int32  `json:"timeout,omitempty"`
	InactivityTimeout *int32  `json:"inactivity_timeout,omitempty"`
	StopOnRunFinished *bool   `json:"stop_on_run_finished,omitempty"`
	RunnerType        *string `json:"runner_type,omitempty"`
}

// ScheduledSessionBuilder builds a ScheduledSession for creation.
type ScheduledSessionBuilder struct {
	resource ScheduledSession
	errs     []error
}

func NewScheduledSessionBuilder() *ScheduledSessionBuilder {
	return &ScheduledSessionBuilder{}
}

func (b *ScheduledSessionBuilder) Name(v string) *ScheduledSessionBuilder {
	b.resource.Name = v
	return b
}

func (b *ScheduledSessionBuilder) ProjectID(v string) *ScheduledSessionBuilder {
	b.resource.ProjectID = v
	return b
}

func (b *ScheduledSessionBuilder) AgentID(v string) *ScheduledSessionBuilder {
	b.resource.AgentID = v
	return b
}

func (b *ScheduledSessionBuilder) Schedule(v string) *ScheduledSessionBuilder {
	b.resource.Schedule = v
	return b
}

func (b *ScheduledSessionBuilder) Timezone(v string) *ScheduledSessionBuilder {
	b.resource.Timezone = v
	return b
}

func (b *ScheduledSessionBuilder) SessionPrompt(v string) *ScheduledSessionBuilder {
	b.resource.SessionPrompt = v
	return b
}

func (b *ScheduledSessionBuilder) Description(v string) *ScheduledSessionBuilder {
	b.resource.Description = v
	return b
}

func (b *ScheduledSessionBuilder) Timeout(v int32) *ScheduledSessionBuilder {
	b.resource.Timeout = &v
	return b
}

func (b *ScheduledSessionBuilder) InactivityTimeout(v int32) *ScheduledSessionBuilder {
	b.resource.InactivityTimeout = &v
	return b
}

func (b *ScheduledSessionBuilder) StopOnRunFinished(v bool) *ScheduledSessionBuilder {
	b.resource.StopOnRunFinished = &v
	return b
}

func (b *ScheduledSessionBuilder) RunnerType(v string) *ScheduledSessionBuilder {
	b.resource.RunnerType = v
	return b
}

func (b *ScheduledSessionBuilder) Build() (*ScheduledSession, error) {
	if b.resource.Name == "" {
		b.errs = append(b.errs, fmt.Errorf("name is required"))
	}
	if b.resource.Schedule == "" {
		b.errs = append(b.errs, fmt.Errorf("schedule is required"))
	}
	if len(b.errs) > 0 {
		return nil, fmt.Errorf("validation failed: %w", errors.Join(b.errs...))
	}
	return &b.resource, nil
}

// ScheduledSessionPatchBuilder builds a ScheduledSessionPatch for update operations.
type ScheduledSessionPatchBuilder struct {
	patch ScheduledSessionPatch
}

func NewScheduledSessionPatchBuilder() *ScheduledSessionPatchBuilder {
	return &ScheduledSessionPatchBuilder{}
}

func (b *ScheduledSessionPatchBuilder) Name(v string) *ScheduledSessionPatchBuilder {
	b.patch.Name = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) Description(v string) *ScheduledSessionPatchBuilder {
	b.patch.Description = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) Schedule(v string) *ScheduledSessionPatchBuilder {
	b.patch.Schedule = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) Timezone(v string) *ScheduledSessionPatchBuilder {
	b.patch.Timezone = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) Enabled(v bool) *ScheduledSessionPatchBuilder {
	b.patch.Enabled = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) SessionPrompt(v string) *ScheduledSessionPatchBuilder {
	b.patch.SessionPrompt = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) AgentID(v string) *ScheduledSessionPatchBuilder {
	b.patch.AgentID = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) Timeout(v int32) *ScheduledSessionPatchBuilder {
	b.patch.Timeout = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) InactivityTimeout(v int32) *ScheduledSessionPatchBuilder {
	b.patch.InactivityTimeout = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) StopOnRunFinished(v bool) *ScheduledSessionPatchBuilder {
	b.patch.StopOnRunFinished = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) RunnerType(v string) *ScheduledSessionPatchBuilder {
	b.patch.RunnerType = &v
	return b
}

func (b *ScheduledSessionPatchBuilder) Build() *ScheduledSessionPatch {
	return &b.patch
}
