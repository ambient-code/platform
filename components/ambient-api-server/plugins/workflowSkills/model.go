package workflowSkills

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type WorkflowSkill struct {
	api.Meta
	WorkflowId string `json:"workflow_id"`
	SkillId    string `json:"skill_id"`
	Position   int    `json:"position"`
}

type WorkflowSkillList []*WorkflowSkill
type WorkflowSkillIndex map[string]*WorkflowSkill

func (l WorkflowSkillList) Index() WorkflowSkillIndex {
	index := WorkflowSkillIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *WorkflowSkill) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type WorkflowSkillPatchRequest struct {
	WorkflowId *string `json:"workflow_id,omitempty"`
	SkillId    *string `json:"skill_id,omitempty"`
	Position   *int    `json:"position,omitempty"`
}
