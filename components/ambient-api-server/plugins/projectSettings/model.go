package projectSettings

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type ProjectSettings struct {
	api.Meta
	ProjectId    string  `json:"project_id" gorm:"uniqueIndex;not null"`
	GroupAccess  *string `json:"group_access"`
	Repositories *string `json:"repositories"`
}

type ProjectSettingsList []*ProjectSettings
type ProjectSettingsIndex map[string]*ProjectSettings

func (l ProjectSettingsList) Index() ProjectSettingsIndex {
	index := ProjectSettingsIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *ProjectSettings) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type ProjectSettingsPatchRequest struct {
	ProjectId    *string `json:"project_id,omitempty"`
	GroupAccess  *string `json:"group_access,omitempty"`
	Repositories *string `json:"repositories,omitempty"`
}
