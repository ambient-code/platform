package types

type ScheduledSessionPatch struct {
	AgentID           *string `json:"agent_id,omitempty"`
	Description       *string `json:"description,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
	InactivityTimeout *int32  `json:"inactivity_timeout,omitempty"`
	Name              *string `json:"name,omitempty"`
	RunnerType        *string `json:"runner_type,omitempty"`
	Schedule          *string `json:"schedule,omitempty"`
	SessionPrompt     *string `json:"session_prompt,omitempty"`
	StopOnRunFinished *bool   `json:"stop_on_run_finished,omitempty"`
	Timeout           *int32  `json:"timeout,omitempty"`
	Timezone          *string `json:"timezone,omitempty"`
}
