package types

import (
	"errors"
	"fmt"
)

type Credential struct {
	ObjectReference

	ProjectID   string `json:"project_id"`
	Annotations string `json:"annotations,omitempty"`
	Description string `json:"description,omitempty"`
	Email       string `json:"email,omitempty"`
	Labels      string `json:"labels,omitempty"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	Token       string `json:"token,omitempty"`
	Url         string `json:"url,omitempty"`
}

type CredentialList struct {
	ListMeta
	Items []Credential `json:"items"`
}

func (l *CredentialList) GetItems() []Credential { return l.Items }
func (l *CredentialList) GetTotal() int          { return l.Total }
func (l *CredentialList) GetPage() int           { return l.Page }
func (l *CredentialList) GetSize() int           { return l.Size }

type CredentialTokenResponse struct {
	CredentialID string `json:"credential_id"`
	Provider     string `json:"provider"`
	Token        string `json:"token"`
}

type CredentialBuilder struct {
	resource Credential
	errors   []error
}

func NewCredentialBuilder() *CredentialBuilder {
	return &CredentialBuilder{}
}

func (b *CredentialBuilder) Name(v string) *CredentialBuilder {
	b.resource.Name = v
	return b
}

func (b *CredentialBuilder) Provider(v string) *CredentialBuilder {
	b.resource.Provider = v
	return b
}

func (b *CredentialBuilder) Token(v string) *CredentialBuilder {
	b.resource.Token = v
	return b
}

func (b *CredentialBuilder) Description(v string) *CredentialBuilder {
	b.resource.Description = v
	return b
}

func (b *CredentialBuilder) Url(v string) *CredentialBuilder {
	b.resource.Url = v
	return b
}

func (b *CredentialBuilder) Email(v string) *CredentialBuilder {
	b.resource.Email = v
	return b
}

func (b *CredentialBuilder) Labels(v string) *CredentialBuilder {
	b.resource.Labels = v
	return b
}

func (b *CredentialBuilder) Annotations(v string) *CredentialBuilder {
	b.resource.Annotations = v
	return b
}

func (b *CredentialBuilder) Build() (*Credential, error) {
	if b.resource.Name == "" {
		b.errors = append(b.errors, fmt.Errorf("name is required"))
	}
	if b.resource.Provider == "" {
		b.errors = append(b.errors, fmt.Errorf("provider is required"))
	}
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("validation failed: %w", errors.Join(b.errors...))
	}
	return &b.resource, nil
}

type CredentialPatchBuilder struct {
	patch map[string]any
}

func NewCredentialPatchBuilder() *CredentialPatchBuilder {
	return &CredentialPatchBuilder{patch: make(map[string]any)}
}

func (b *CredentialPatchBuilder) Name(v string) *CredentialPatchBuilder {
	b.patch["name"] = v
	return b
}

func (b *CredentialPatchBuilder) Token(v string) *CredentialPatchBuilder {
	b.patch["token"] = v
	return b
}

func (b *CredentialPatchBuilder) Description(v string) *CredentialPatchBuilder {
	b.patch["description"] = v
	return b
}

func (b *CredentialPatchBuilder) Url(v string) *CredentialPatchBuilder {
	b.patch["url"] = v
	return b
}

func (b *CredentialPatchBuilder) Email(v string) *CredentialPatchBuilder {
	b.patch["email"] = v
	return b
}

func (b *CredentialPatchBuilder) Labels(v string) *CredentialPatchBuilder {
	b.patch["labels"] = v
	return b
}

func (b *CredentialPatchBuilder) Annotations(v string) *CredentialPatchBuilder {
	b.patch["annotations"] = v
	return b
}

func (b *CredentialPatchBuilder) Build() map[string]any {
	return b.patch
}
