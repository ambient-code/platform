package projectDocuments_test

import (
	"context"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/projectDocuments"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func newProjectDocument(id string) (*projectDocuments.ProjectDocument, error) {
	projectDocumentService := projectDocuments.Service(&environments.Environment().Services)

	projectDocument := &projectDocuments.ProjectDocument{
		ProjectId: "test-project_id",
		Slug:      id,
		Title:     stringPtr("test-title"),
		Content:   stringPtr("test-content"),
	}

	sub, err := projectDocumentService.Create(context.Background(), projectDocument)
	if err != nil {
		return nil, err
	}

	return sub, nil
}

func newProjectDocumentList(namePrefix string, count int) ([]*projectDocuments.ProjectDocument, error) {
	var items []*projectDocuments.ProjectDocument
	for i := 1; i <= count; i++ {
		name := fmt.Sprintf("%s_%d", namePrefix, i)
		c, err := newProjectDocument(name)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, nil
}
func stringPtr(s string) *string { return &s }
