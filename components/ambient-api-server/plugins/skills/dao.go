package skills

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type SkillDao interface {
	Get(ctx context.Context, id string) (*Skill, error)
	Create(ctx context.Context, skill *Skill) (*Skill, error)
	Replace(ctx context.Context, skill *Skill) (*Skill, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (SkillList, error)
	All(ctx context.Context) (SkillList, error)
}

var _ SkillDao = &sqlSkillDao{}

type sqlSkillDao struct {
	sessionFactory *db.SessionFactory
}

func NewSkillDao(sessionFactory *db.SessionFactory) SkillDao {
	return &sqlSkillDao{sessionFactory: sessionFactory}
}

func (d *sqlSkillDao) Get(ctx context.Context, id string) (*Skill, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var skill Skill
	if err := g2.Take(&skill, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &skill, nil
}

func (d *sqlSkillDao) Create(ctx context.Context, skill *Skill) (*Skill, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(skill).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return skill, nil
}

func (d *sqlSkillDao) Replace(ctx context.Context, skill *Skill) (*Skill, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(skill).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return skill, nil
}

func (d *sqlSkillDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Skill{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlSkillDao) FindByIDs(ctx context.Context, ids []string) (SkillList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	skills := SkillList{}
	if err := g2.Where("id in (?)", ids).Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

func (d *sqlSkillDao) All(ctx context.Context) (SkillList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	skills := SkillList{}
	if err := g2.Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}
