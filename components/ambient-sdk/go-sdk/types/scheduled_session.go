package types

import (
	"fmt"
	"time"
)

type ScheduledSession struct {
	ObjectReference

	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	ProjectID     string     `json:"project_id,omitempty"`
	AgentID       string     `json:"agent_id,omitempty"`
	Schedule      string     `json:"schedule"`
	Timezone      string     `json:"timezone,omitempty"`
	Enabled       bool       `json:"enabled"`
	SessionPrompt string     `json:"session_prompt,omitempty"`
	LastRunAt     *time.Time `json:"last_run_at,omitempty"`
	NextRunAt     *time.Time `json:"next_run_at,omitempty"`
}

type ScheduledSessionList struct {
	ListMeta
	Items []ScheduledSession `json:"items"`
}

func (l *ScheduledSessionList) GetItems() []ScheduledSession { return l.Items }
func (l *ScheduledSessionList) GetTotal() int                { return l.Total }
func (l *ScheduledSessionList) GetPage() int                 { return l.Page }
func (l *ScheduledSessionList) GetSize() int                 { return l.Size }

type ScheduledSessionPatch struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	Schedule      *string `json:"schedule,omitempty"`
	Timezone      *string `json:"timezone,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
	SessionPrompt *string `json:"session_prompt,omitempty"`
	AgentID       *string `json:"agent_id,omitempty"`
}

// ScheduledSessionBuilder provides a fluent API for constructing ScheduledSession values.
type ScheduledSessionBuilder struct {
	resource ScheduledSession
}

func NewScheduledSessionBuilder() *ScheduledSessionBuilder {
	return &ScheduledSessionBuilder{resource: ScheduledSession{Enabled: true}}
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
func (b *ScheduledSessionBuilder) Build() (*ScheduledSession, error) {
	if b.resource.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if b.resource.Schedule == "" {
		return nil, fmt.Errorf("schedule is required")
	}
	return &b.resource, nil
}

// ScheduledSessionPatchBuilder provides a fluent API for constructing ScheduledSessionPatch values.
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
func (b *ScheduledSessionPatchBuilder) Schedule(v string) *ScheduledSessionPatchBuilder {
	b.patch.Schedule = &v
	return b
}
func (b *ScheduledSessionPatchBuilder) Timezone(v string) *ScheduledSessionPatchBuilder {
	b.patch.Timezone = &v
	return b
}
func (b *ScheduledSessionPatchBuilder) SessionPrompt(v string) *ScheduledSessionPatchBuilder {
	b.patch.SessionPrompt = &v
	return b
}
func (b *ScheduledSessionPatchBuilder) Description(v string) *ScheduledSessionPatchBuilder {
	b.patch.Description = &v
	return b
}
func (b *ScheduledSessionPatchBuilder) Build() *ScheduledSessionPatch {
	return &b.patch
}
