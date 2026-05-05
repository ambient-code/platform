package types

type ScheduledSessionPatch struct {
	Name          *string `json:"name,omitempty"`
	Description   *string `json:"description,omitempty"`
	Schedule      *string `json:"schedule,omitempty"`
	Timezone      *string `json:"timezone,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
	SessionPrompt *string `json:"session_prompt,omitempty"`
}
