package roleBindings

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type RoleBinding struct {
	api.Meta
	RoleId       string  `json:"role_id"       gorm:"not null;index"`
	Scope        string  `json:"scope"         gorm:"not null"`
	UserId       *string `json:"user_id"       gorm:"index"`
	ProjectId    *string `json:"project_id"    gorm:"index"`
	AgentId      *string `json:"agent_id"      gorm:"index"`
	SessionId    *string `json:"session_id"    gorm:"index"`
	CredentialId *string `json:"credential_id" gorm:"index"`
}

type RoleBindingList []*RoleBinding
type RoleBindingIndex map[string]*RoleBinding

func (l RoleBindingList) Index() RoleBindingIndex {
	index := RoleBindingIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *RoleBinding) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type RoleBindingPatchRequest struct {
	RoleId       *string `json:"role_id,omitempty"`
	Scope        *string `json:"scope,omitempty"`
	UserId       *string `json:"user_id,omitempty"`
	ProjectId    *string `json:"project_id,omitempty"`
	AgentId      *string `json:"agent_id,omitempty"`
	SessionId    *string `json:"session_id,omitempty"`
	CredentialId *string `json:"credential_id,omitempty"`
}
