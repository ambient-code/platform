package permissions

import (
	"fmt"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Permission struct {
	api.Meta
	SubjectType string  `json:"subject_type"`
	SubjectName string  `json:"subject_name"`
	Role        string  `json:"role"`
	ProjectId   *string `json:"project_id"`
}

type PermissionList []*Permission
type PermissionIndex map[string]*Permission

func (l PermissionList) Index() PermissionIndex {
	index := PermissionIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

var validSubjectTypes = map[string]bool{
	"user":  true,
	"group": true,
}

var validRoles = map[string]bool{
	"admin": true,
	"edit":  true,
	"view":  true,
}

func (d *Permission) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()

	if !validSubjectTypes[d.SubjectType] {
		return fmt.Errorf("invalid subject_type %q: must be 'user' or 'group'", d.SubjectType)
	}
	if !validRoles[d.Role] {
		return fmt.Errorf("invalid role %q: must be 'admin', 'edit', or 'view'", d.Role)
	}

	return nil
}

type PermissionPatchRequest struct {
	SubjectType *string `json:"subject_type,omitempty"`
	SubjectName *string `json:"subject_name,omitempty"`
	Role        *string `json:"role,omitempty"`
	ProjectId   *string `json:"project_id,omitempty"`
}
