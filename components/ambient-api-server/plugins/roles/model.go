package roles

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Role struct {
	api.Meta
	Name        string   `json:"name"         gorm:"uniqueIndex;not null"`
	DisplayName *string  `json:"display_name"`
	Description *string  `json:"description"`
	Permissions []string `json:"permissions"  gorm:"type:text;serializer:json"`
	BuiltIn     bool     `json:"built_in"     gorm:"default:false"`
}

type RoleList []*Role
type RoleIndex map[string]*Role

func (l RoleList) Index() RoleIndex {
	index := RoleIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Role) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type RolePatchRequest struct {
	DisplayName *string  `json:"display_name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}
