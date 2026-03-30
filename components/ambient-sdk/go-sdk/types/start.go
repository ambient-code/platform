package types

type StartRequest struct {
	Prompt string `json:"prompt,omitempty"`
}

type StartResponse struct {
	Session        *Session `json:"session,omitempty"`
	StartingPrompt string   `json:"starting_prompt,omitempty"`
}

type ProjectHome struct {
	ProjectID string             `json:"project_id,omitempty"`
	Agents    []ProjectHomeAgent `json:"agents,omitempty"`
}

type ProjectHomeAgent struct {
	AgentID          string `json:"agent_id,omitempty"`
	AgentName        string `json:"agent_name,omitempty"`
	SessionPhase     string `json:"session_phase,omitempty"`
	InboxUnreadCount int    `json:"inbox_unread_count,omitempty"`
	Summary          string `json:"summary,omitempty"`
}
