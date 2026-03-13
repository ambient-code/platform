package projectDocuments

import (
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"gorm.io/gorm"
)

type ProjectDocument struct {
	api.Meta
	ProjectId string  `json:"project_id" gorm:"not null;index"`
	Slug      string  `json:"slug"       gorm:"not null"`
	Title     *string `json:"title"`
	Content   *string `json:"content"    gorm:"type:text"`
}

type ProjectDocumentList []*ProjectDocument
type ProjectDocumentIndex map[string]*ProjectDocument

func (l ProjectDocumentList) Index() ProjectDocumentIndex {
	index := ProjectDocumentIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (d *ProjectDocument) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()
	return nil
}

type ProjectDocumentPatchRequest struct {
	Title   *string `json:"title,omitempty"`
	Content *string `json:"content,omitempty"`
}
