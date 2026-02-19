package skills

import (
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/util"
)

func ConvertSkill(skill openapi.Skill) *Skill {
	c := &Skill{
		Meta: api.Meta{
			ID: util.NilToEmptyString(skill.Id),
		},
	}
	c.Name = skill.Name
	c.RepoUrl = skill.RepoUrl
	c.Prompt = skill.Prompt
	c.ProjectId = skill.ProjectId

	if skill.CreatedAt != nil {
		c.CreatedAt = *skill.CreatedAt
		c.UpdatedAt = *skill.UpdatedAt
	}

	return c
}

func PresentSkill(skill *Skill) openapi.Skill {
	reference := presenters.PresentReference(skill.ID, skill)
	return openapi.Skill{
		Id:        reference.Id,
		Kind:      reference.Kind,
		Href:      reference.Href,
		CreatedAt: openapi.PtrTime(skill.CreatedAt),
		UpdatedAt: openapi.PtrTime(skill.UpdatedAt),
		Name:      skill.Name,
		RepoUrl:   skill.RepoUrl,
		Prompt:    skill.Prompt,
		ProjectId: skill.ProjectId,
	}
}
