package skills

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ SkillDao = &skillDaoMock{}

type skillDaoMock struct {
	skills SkillList
}

func NewMockSkillDao() *skillDaoMock {
	return &skillDaoMock{}
}

func (d *skillDaoMock) Get(ctx context.Context, id string) (*Skill, error) {
	for _, skill := range d.skills {
		if skill.ID == id {
			return skill, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *skillDaoMock) Create(ctx context.Context, skill *Skill) (*Skill, error) {
	d.skills = append(d.skills, skill)
	return skill, nil
}

func (d *skillDaoMock) Replace(ctx context.Context, skill *Skill) (*Skill, error) {
	return nil, errors.NotImplemented("Skill").AsError()
}

func (d *skillDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Skill").AsError()
}

func (d *skillDaoMock) FindByIDs(ctx context.Context, ids []string) (SkillList, error) {
	return nil, errors.NotImplemented("Skill").AsError()
}

func (d *skillDaoMock) All(ctx context.Context) (SkillList, error) {
	return d.skills, nil
}
