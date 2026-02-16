package skills_test

import (
	"context"
	"fmt"

	"github.com/ambient/platform/components/ambient-api-server/plugins/skills"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newSkill(id string) (*skills.Skill, error) {
	skillService := skills.Service(&environments.Environment().Services)

	skill := &skills.Skill{
		Name:    "test-name",
		RepoUrl: stringPtr("test-repo_url"),
		Prompt:  stringPtr("test-prompt"),
	}

	sub, err := skillService.Create(context.Background(), skill)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newSkillList(namePrefix string, count int) ([]*skills.Skill, error) {
	var items []*skills.Skill
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newSkill(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
