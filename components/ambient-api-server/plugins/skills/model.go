package skills

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Skill struct {
	api.Meta
	Name      string  `json:"name"`
	RepoUrl   *string `json:"repo_url"`
	Prompt    *string `json:"prompt"`
	ProjectId *string `json:"project_id"`
}

type SkillList []*Skill
type SkillIndex map[string]*Skill

func (l SkillList) Index() SkillIndex {
	index := SkillIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Skill) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type SkillPatchRequest struct {
	Name      *string `json:"name,omitempty"`
	RepoUrl   *string `json:"repo_url,omitempty"`
	Prompt    *string `json:"prompt,omitempty"`
	ProjectId *string `json:"project_id,omitempty"`
}
