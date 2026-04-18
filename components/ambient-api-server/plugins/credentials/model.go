package credentials

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type Credential struct {
	api.Meta
	ProjectID   string  `json:"project_id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Provider    string  `json:"provider"`
	Token       *string `json:"token"`
	Url         *string `json:"url"`
	Email       *string `json:"email"`
	Labels      *string `json:"labels"`
	Annotations *string `json:"annotations"`
}

type CredentialList []*Credential
type CredentialIndex map[string]*Credential

func (l CredentialList) Index() CredentialIndex {
	index := CredentialIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *Credential) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type CredentialPatchRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Provider    *string `json:"provider,omitempty"`
	Token       *string `json:"token,omitempty"`
	Url         *string `json:"url,omitempty"`
	Email       *string `json:"email,omitempty"`
	Labels      *string `json:"labels,omitempty"`
	Annotations *string `json:"annotations,omitempty"`
}
