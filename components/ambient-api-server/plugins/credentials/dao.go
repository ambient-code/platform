package credentials

import (
	"context"

	"gorm.io/gorm/clause"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
)

type CredentialDao interface {
	Get(ctx context.Context, id string) (*Credential, error)
	Create(ctx context.Context, credential *Credential) (*Credential, error)
	Replace(ctx context.Context, credential *Credential) (*Credential, error)
	Delete(ctx context.Context, id string) error
	FindByIDs(ctx context.Context, ids []string) (CredentialList, error)
	All(ctx context.Context) (CredentialList, error)
}

var _ CredentialDao = &sqlCredentialDao{}

type sqlCredentialDao struct {
	sessionFactory *db.SessionFactory
}

func NewCredentialDao(sessionFactory *db.SessionFactory) CredentialDao {
	return &sqlCredentialDao{sessionFactory: sessionFactory}
}

func (d *sqlCredentialDao) Get(ctx context.Context, id string) (*Credential, error) {
	g2 := (*d.sessionFactory).New(ctx)
	var credential Credential
	if err := g2.Take(&credential, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &credential, nil
}

func (d *sqlCredentialDao) Create(ctx context.Context, credential *Credential) (*Credential, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Create(credential).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return credential, nil
}

func (d *sqlCredentialDao) Replace(ctx context.Context, credential *Credential) (*Credential, error) {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Save(credential).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, err
	}
	return credential, nil
}

func (d *sqlCredentialDao) Delete(ctx context.Context, id string) error {
	g2 := (*d.sessionFactory).New(ctx)
	if err := g2.Omit(clause.Associations).Delete(&Credential{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return err
	}
	return nil
}

func (d *sqlCredentialDao) FindByIDs(ctx context.Context, ids []string) (CredentialList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	credentials := CredentialList{}
	if err := g2.Where("id in (?)", ids).Find(&credentials).Error; err != nil {
		return nil, err
	}
	return credentials, nil
}

func (d *sqlCredentialDao) All(ctx context.Context) (CredentialList, error) {
	g2 := (*d.sessionFactory).New(ctx)
	credentials := CredentialList{}
	if err := g2.Find(&credentials).Error; err != nil {
		return nil, err
	}
	return credentials, nil
}
