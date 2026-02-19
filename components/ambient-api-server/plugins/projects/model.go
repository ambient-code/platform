package projects

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Project struct {
	api.Meta
	Name        string  `json:"name" gorm:"uniqueIndex;not null"`
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`
	Labels      *string `json:"labels"`
	Annotations *string `json:"annotations"`
	Status      *string `json:"status"`
}

type ProjectList []*Project
type ProjectIndex map[string]*Project

func (l ProjectList) Index() ProjectIndex {
	index := ProjectIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Project) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type ProjectPatchRequest struct {
	Name        *string `json:"name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Description *string `json:"description,omitempty"`
	Labels      *string `json:"labels,omitempty"`
	Annotations *string `json:"annotations,omitempty"`
	Status      *string `json:"status,omitempty"`
}
