package sessions

import (
	"time"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Session struct {
	api.Meta
	Name            string  `json:"name"`
	RepoUrl         *string `json:"repo_url"`
	Prompt          *string `json:"prompt"`
	CreatedByUserId *string `json:"created_by_user_id"`
	AssignedUserId  *string `json:"assigned_user_id"`
	WorkflowId      *string `json:"workflow_id"`

	Repos                *string  `json:"repos"`
	Interactive          *bool    `json:"interactive"`
	Timeout              *int32   `json:"timeout"`
	LlmModel             *string  `json:"llm_model"`
	LlmTemperature       *float64 `json:"llm_temperature"`
	LlmMaxTokens         *int32   `json:"llm_max_tokens"`
	ParentSessionId      *string  `json:"parent_session_id"`
	BotAccountName       *string  `json:"bot_account_name"`
	ResourceOverrides    *string  `json:"resource_overrides"`
	EnvironmentVariables *string  `json:"environment_variables"`
	SessionLabels        *string  `json:"labels" gorm:"column:labels"`
	SessionAnnotations   *string  `json:"annotations" gorm:"column:annotations"`
	ProjectId            *string  `json:"project_id"`

	Phase              *string    `json:"phase"`
	StartTime          *time.Time `json:"start_time"`
	CompletionTime     *time.Time `json:"completion_time"`
	SdkSessionId       *string    `json:"sdk_session_id"`
	SdkRestartCount    *int32     `json:"sdk_restart_count"`
	Conditions         *string    `json:"conditions"`
	ReconciledRepos    *string    `json:"reconciled_repos"`
	ReconciledWorkflow *string    `json:"reconciled_workflow"`
	KubeCrName         *string    `json:"kube_cr_name"`
	KubeCrUid          *string    `json:"kube_cr_uid"`
	KubeNamespace      *string    `json:"kube_namespace"`
}

type SessionList []*Session
type SessionIndex map[string]*Session

func (l SessionList) Index() SessionIndex {
	index := SessionIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Session) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	d.KubeCrName = &d.ID

	if d.LlmModel == nil || *d.LlmModel == "" {
		defaultModel := "sonnet"
		d.LlmModel = &defaultModel
	}
	if d.LlmTemperature == nil {
		defaultTemp := 0.7
		d.LlmTemperature = &defaultTemp
	}
	if d.LlmMaxTokens == nil {
		defaultTokens := int32(4000)
		d.LlmMaxTokens = &defaultTokens
	}

	return nil
}

type SessionPatchRequest struct {
	Name                 *string  `json:"name,omitempty"`
	RepoUrl              *string  `json:"repo_url,omitempty"`
	Prompt               *string  `json:"prompt,omitempty"`
	AssignedUserId       *string  `json:"assigned_user_id,omitempty"`
	WorkflowId           *string  `json:"workflow_id,omitempty"`
	Repos                *string  `json:"repos,omitempty"`
	Interactive          *bool    `json:"interactive,omitempty"`
	Timeout              *int32   `json:"timeout,omitempty"`
	LlmModel             *string  `json:"llm_model,omitempty"`
	LlmTemperature       *float64 `json:"llm_temperature,omitempty"`
	LlmMaxTokens         *int32   `json:"llm_max_tokens,omitempty"`
	ParentSessionId      *string  `json:"parent_session_id,omitempty"`
	BotAccountName       *string  `json:"bot_account_name,omitempty"`
	ResourceOverrides    *string  `json:"resource_overrides,omitempty"`
	EnvironmentVariables *string  `json:"environment_variables,omitempty"`
	SessionLabels        *string  `json:"labels,omitempty"`
	SessionAnnotations   *string  `json:"annotations,omitempty"`
}

type SessionStatusPatchRequest struct {
	Phase              *string    `json:"phase,omitempty"`
	StartTime          *time.Time `json:"start_time,omitempty"`
	CompletionTime     *time.Time `json:"completion_time,omitempty"`
	SdkSessionId       *string    `json:"sdk_session_id,omitempty"`
	SdkRestartCount    *int32     `json:"sdk_restart_count,omitempty"`
	Conditions         *string    `json:"conditions,omitempty"`
	ReconciledRepos    *string    `json:"reconciled_repos,omitempty"`
	ReconciledWorkflow *string    `json:"reconciled_workflow,omitempty"`
	KubeCrUid          *string    `json:"kube_cr_uid,omitempty"`
	KubeNamespace      *string    `json:"kube_namespace,omitempty"`
}
