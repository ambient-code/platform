package projectSettings

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type ProjectSettingsDao interface {
	Get(ctx context.Context, id string) (*ProjectSettings, error)
	Create(ctx context.Context, projectSettings *ProjectSettings) (*ProjectSettings, error)
	Replace(ctx context.Context, projectSettings *ProjectSettings) (*ProjectSettings, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (ProjectSettingsList, error)
	All(ctx context.Context) (ProjectSettingsList, error)
	AllByProjectId(ctx context.Context, projectId string) (ProjectSettingsList, error)
}

var _ ProjectSettingsDao = &sqlProjectSettingsDao{}

type sqlProjectSettingsDao struct {
	sessionFactory *db.SessionFactory
}

func NewProjectSettingsDao(sessionFactory *db.SessionFactory) ProjectSettingsDao {
	return &sqlProjectSettingsDao{sessionFactory: sessionFactory}
}

func (d *sqlProjectSettingsDao) Get(ctx context.Context, id string) (*ProjectSettings, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var ps ProjectSettings
	if err := g2.Take(&ps, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &ps, nil
}

func (d *sqlProjectSettingsDao) Create(ctx context.Context, projectSettings *ProjectSettings) (*ProjectSettings, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(projectSettings).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return projectSettings, nil
}

func (d *sqlProjectSettingsDao) Replace(ctx context.Context, projectSettings *ProjectSettings) (*ProjectSettings, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(projectSettings).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return projectSettings, nil
}

func (d *sqlProjectSettingsDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&ProjectSettings{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlProjectSettingsDao) FindByIDs(ctx context.Context, ids []string) (ProjectSettingsList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	list := ProjectSettingsList{}
	if err := g2.Where("id in (?)", ids).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (d *sqlProjectSettingsDao) All(ctx context.Context) (ProjectSettingsList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	list := ProjectSettingsList{}
	if err := g2.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (d *sqlProjectSettingsDao) AllByProjectId(ctx context.Context, projectId string) (ProjectSettingsList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	list := ProjectSettingsList{}
	if err := g2.Where("project_id = ?", projectId).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}
