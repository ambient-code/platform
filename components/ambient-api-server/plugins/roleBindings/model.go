package roleBindings

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type RoleBinding struct {
	api.Meta
	UserId  string  `json:"user_id"  gorm:"not null;index"`
	RoleId  string  `json:"role_id"  gorm:"not null;index"`
	Scope   string  `json:"scope"    gorm:"not null"`
	ScopeId *string `json:"scope_id"`
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
	ScopeId *string `json:"scope_id,omitempty"`
}
