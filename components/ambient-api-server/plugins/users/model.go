package users

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type User struct {
	api.Meta
	Username string `json:"username"`
	Name     string `json:"name"`
	Groups   *string `json:"groups"`
}

type UserList []*User
type UserIndex map[string]*User

func (l UserList) Index() UserIndex {
	index := UserIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *User) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type UserPatchRequest struct {
	Username *string `json:"username,omitempty"`
	Name     *string `json:"name,omitempty"`
	Groups   *string `json:"groups,omitempty"`
}
