package projectDocuments

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = projectDocumentHandler{}

type projectDocumentHandler struct {
	projectDocument ProjectDocumentService
	generic         services.GenericService
}

func NewProjectDocumentHandler(projectDocument ProjectDocumentService, generic services.GenericService) *projectDocumentHandler {
	return &projectDocumentHandler{
		projectDocument: projectDocument,
		generic:         generic,
	}
}

func (h projectDocumentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var projectDocument openapi.ProjectDocument
	cfg := &handlers.HandlerConfig{
		Body: &projectDocument,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&projectDocument, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectDocumentModel := ConvertProjectDocument(projectDocument)
			projectDocumentModel, err := h.projectDocument.Create(ctx, projectDocumentModel)
			if err != nil {
				return nil, err
			}
			return PresentProjectDocument(projectDocumentModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h projectDocumentHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.ProjectDocumentPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.projectDocument.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.ProjectId != nil {
				found.ProjectId = *patch.ProjectId
			}
			if patch.Slug != nil {
				found.Slug = *patch.Slug
			}
			if patch.Title != nil {
				found.Title = patch.Title
			}
			if patch.Content != nil {
				found.Content = patch.Content
			}

			projectDocumentModel, err := h.projectDocument.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentProjectDocument(projectDocumentModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h projectDocumentHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var projectDocuments []ProjectDocument
			paging, err := h.generic.List(ctx, "id", listArgs, &projectDocuments)
			if err != nil {
				return nil, err
			}
			projectDocumentList := openapi.ProjectDocumentList{
				Kind:  "ProjectDocumentList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.ProjectDocument{},
			}

			for _, projectDocument := range projectDocuments {
				converted := PresentProjectDocument(&projectDocument)
				projectDocumentList.Items = append(projectDocumentList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, projectDocumentList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return projectDocumentList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h projectDocumentHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			projectDocument, err := h.projectDocument.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentProjectDocument(projectDocument), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h projectDocumentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.projectDocument.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
