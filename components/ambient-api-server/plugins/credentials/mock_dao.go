package credentials

import (
	"context"

	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
)

var _ CredentialDao = &credentialDaoMock{}

type credentialDaoMock struct {
	credentials CredentialList
}

func NewMockCredentialDao() *credentialDaoMock {
	return &credentialDaoMock{}
}

func (d *credentialDaoMock) Get(ctx context.Context, id string) (*Credential, error) {
	for _, credential := range d.credentials {
		if credential.ID == id {
			return credential, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (d *credentialDaoMock) Create(ctx context.Context, credential *Credential) (*Credential, error) {
	if err := credential.BeforeCreate(nil); err != nil {
		return nil, err
	}
	d.credentials = append(d.credentials, credential)
	return credential, nil
}

func (d *credentialDaoMock) Replace(ctx context.Context, credential *Credential) (*Credential, error) {
	return nil, errors.NotImplemented("Credential").AsError()
}

func (d *credentialDaoMock) Delete(ctx context.Context, id string) error {
	return errors.NotImplemented("Credential").AsError()
}

func (d *credentialDaoMock) FindByIDs(ctx context.Context, ids []string) (CredentialList, error) {
	return nil, errors.NotImplemented("Credential").AsError()
}

func (d *credentialDaoMock) All(ctx context.Context) (CredentialList, error) {
	return d.credentials, nil
}
