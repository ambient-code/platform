package agents

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Agent struct {
	api.Meta
	ProjectId            string  `json:"project_id"             gorm:"not null;index"`
	ParentAgentId        *string `json:"parent_agent_id"        gorm:"index"`
	OwnerUserId          string  `json:"owner_user_id"          gorm:"not null"`
	Name                 string  `json:"name"                   gorm:"not null"`
	DisplayName          *string `json:"display_name"`
	Description          *string `json:"description"`
	Prompt               *string `json:"prompt"                 gorm:"type:text"`
	RepoUrl              *string `json:"repo_url"`
	WorkflowId           *string `json:"workflow_id"`
	LlmModel             string  `json:"llm_model"              gorm:"default:'claude-sonnet-4-6'"`
	LlmTemperature       float64 `json:"llm_temperature"        gorm:"default:0.7"`
	LlmMaxTokens         int32   `json:"llm_max_tokens"         gorm:"default:4000"`
	BotAccountName       *string `json:"bot_account_name"`
	ResourceOverrides    *string `json:"resource_overrides"`
	EnvironmentVariables *string `json:"environment_variables"`
	Labels               *string `json:"labels"`
	Annotations          *string `json:"annotations"`
	CurrentSessionId     *string `json:"current_session_id"`
}

type AgentList []*Agent
type AgentIndex map[string]*Agent

func (l AgentList) Index() AgentIndex {
	index := AgentIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Agent) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	if d.LlmModel == "" {
		d.LlmModel = "claude-sonnet-4-6"
	}
	if d.LlmTemperature == 0 {
		d.LlmTemperature = 0.7
	}
	if d.LlmMaxTokens == 0 {
		d.LlmMaxTokens = 4000
	}
	return nil
}

type AgentPatchRequest struct {
	DisplayName          *string  `json:"display_name,omitempty"`
	Description          *string  `json:"description,omitempty"`
	Prompt               *string  `json:"prompt,omitempty"`
	RepoUrl              *string  `json:"repo_url,omitempty"`
	WorkflowId           *string  `json:"workflow_id,omitempty"`
	LlmModel             *string  `json:"llm_model,omitempty"`
	LlmTemperature       *float64 `json:"llm_temperature,omitempty"`
	LlmMaxTokens         *int32   `json:"llm_max_tokens,omitempty"`
	BotAccountName       *string  `json:"bot_account_name,omitempty"`
	ResourceOverrides    *string  `json:"resource_overrides,omitempty"`
	EnvironmentVariables *string  `json:"environment_variables,omitempty"`
	Labels               *string  `json:"labels,omitempty"`
	Annotations          *string  `json:"annotations,omitempty"`
	CurrentSessionId     *string  `json:"current_session_id,omitempty"`
	ParentAgentId        *string  `json:"parent_agent_id,omitempty"`
}
