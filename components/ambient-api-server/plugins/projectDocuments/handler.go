package projectDocuments

import (
	"fmt"
	"net/http"
	"net/url"

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

func (h projectDocumentHandler) ListByProject(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["project_id"]

			q := r.URL.Query()
			existing := q.Get("search")
			projectFilter := fmt.Sprintf("project_id = '%s'", projectID)
			if existing != "" {
				q.Set("search", projectFilter+" and "+existing)
			} else {
				q.Set("search", projectFilter)
			}

			listArgs := services.NewListArguments(q)
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

func (h projectDocumentHandler) UpsertBySlug(w http.ResponseWriter, r *http.Request) {
	var body openapi.ProjectDocument

	cfg := &handlers.HandlerConfig{
		Body:       &body,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["project_id"]
			slug := mux.Vars(r)["slug"]

			searchExpr := fmt.Sprintf("project_id = '%s' and slug = '%s'",
				projectID, url.PathEscape(slug))
			q := url.Values{}
			q.Set("search", searchExpr)
			q.Set("size", "1")

			listArgs := services.NewListArguments(q)
			var existing []ProjectDocument
			_, listErr := h.generic.List(ctx, "id", listArgs, &existing)
			if listErr != nil {
				return nil, listErr
			}

			if len(existing) > 0 {
				doc := existing[0]
				if body.Title != nil {
					doc.Title = body.Title
				}
				if body.Content != nil {
					doc.Content = body.Content
				}
				updated, err := h.projectDocument.Replace(ctx, &doc)
				if err != nil {
					return nil, err
				}
				return PresentProjectDocument(updated), nil
			}

			doc := ConvertProjectDocument(body)
			doc.ProjectId = projectID
			doc.Slug = slug
			created, err := h.projectDocument.Create(ctx, doc)
			if err != nil {
				return nil, err
			}
			return PresentProjectDocument(created), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
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
