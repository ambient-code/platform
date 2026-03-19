package projectDocuments

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type ProjectDocumentDao interface {
	Get(ctx context.Context, id string) (*ProjectDocument, error)
	Create(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, error)
	Replace(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (ProjectDocumentList, error)
	All(ctx context.Context) (ProjectDocumentList, error)
}

var _ ProjectDocumentDao = &sqlProjectDocumentDao{}

type sqlProjectDocumentDao struct {
	sessionFactory *db.SessionFactory
}

func NewProjectDocumentDao(sessionFactory *db.SessionFactory) ProjectDocumentDao {
	return &sqlProjectDocumentDao{sessionFactory: sessionFactory}
}

func (d *sqlProjectDocumentDao) Get(ctx context.Context, id string) (*ProjectDocument, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var projectDocument ProjectDocument
	if err := g2.Take(&projectDocument, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &projectDocument, nil
}

func (d *sqlProjectDocumentDao) Create(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(projectDocument).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return projectDocument, nil
}

func (d *sqlProjectDocumentDao) Replace(ctx context.Context, projectDocument *ProjectDocument) (*ProjectDocument, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(projectDocument).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return projectDocument, nil
}

func (d *sqlProjectDocumentDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&ProjectDocument{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlProjectDocumentDao) FindByIDs(ctx context.Context, ids []string) (ProjectDocumentList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	projectDocuments := ProjectDocumentList{}
	if err := g2.Where("id in (?)", ids).Find(&projectDocuments).Error; err != nil {
		return nil, err
	}
	return projectDocuments, nil
}

func (d *sqlProjectDocumentDao) All(ctx context.Context) (ProjectDocumentList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	projectDocuments := ProjectDocumentList{}
	if err := g2.Find(&projectDocuments).Error; err != nil {
		return nil, err
	}
	return projectDocuments, nil
}
