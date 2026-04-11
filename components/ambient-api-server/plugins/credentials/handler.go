package credentials

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var safeProjectIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

var _ handlers.RestHandler = credentialHandler{}

type credentialHandler struct {
	credential CredentialService
	generic    services.GenericService
}

func NewCredentialHandler(credential CredentialService, generic services.GenericService) *credentialHandler {
	return &credentialHandler{
		credential: credential,
		generic:    generic,
	}
}

func (h credentialHandler) Create(w http.ResponseWriter, r *http.Request) {
	var credential openapi.Credential
	cfg := &handlers.HandlerConfig{
		Body: &credential,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&credential, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]
			credentialModel := ConvertCredential(credential, projectID)
			credentialModel, err := h.credential.Create(ctx, credentialModel)
			if err != nil {
				return nil, err
			}
			return PresentCredential(credentialModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h credentialHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.CredentialPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]
			id := mux.Vars(r)["cred_id"]
			found, err := h.credential.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if found.ProjectID != projectID {
				return nil, errors.NotFound("credential with id='%s' not found", id)
			}

			if patch.Name != nil {
				found.Name = *patch.Name
			}
			if patch.Description != nil {
				found.Description = patch.Description
			}
			if patch.Provider != nil {
				found.Provider = *patch.Provider
			}
			if patch.Token != nil {
				found.Token = patch.Token
			}
			if patch.Url != nil {
				found.Url = patch.Url
			}
			if patch.Email != nil {
				found.Email = patch.Email
			}
			if patch.Labels != nil {
				found.Labels = patch.Labels
			}
			if patch.Annotations != nil {
				found.Annotations = patch.Annotations
			}

			credentialModel, err := h.credential.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentCredential(credentialModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h credentialHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			projectID := mux.Vars(r)["id"]
			if !safeProjectIDPattern.MatchString(projectID) {
				return nil, errors.Validation("invalid project ID format")
			}

			listArgs := services.NewListArguments(r.URL.Query())
			projectFilter := fmt.Sprintf("project_id = '%s'", projectID)
			if listArgs.Search != "" {
				listArgs.Search = fmt.Sprintf("(%s) and %s", listArgs.Search, projectFilter)
			} else {
				listArgs.Search = projectFilter
			}
			var credentials []Credential
			paging, err := h.generic.List(ctx, "id", listArgs, &credentials)
			if err != nil {
				return nil, err
			}
			credentialList := openapi.CredentialList{
				Kind:  "CredentialList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Credential{},
			}

			for _, credential := range credentials {
				converted := PresentCredential(&credential)
				credentialList.Items = append(credentialList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, credentialList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return credentialList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h credentialHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			projectID := mux.Vars(r)["id"]
			id := mux.Vars(r)["cred_id"]
			ctx := r.Context()
			credential, err := h.credential.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if credential.ProjectID != projectID {
				return nil, errors.NotFound("credential with id='%s' not found", id)
			}

			return PresentCredential(credential), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h credentialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			projectID := mux.Vars(r)["id"]
			id := mux.Vars(r)["cred_id"]
			ctx := r.Context()
			found, err := h.credential.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if found.ProjectID != projectID {
				return nil, errors.NotFound("credential with id='%s' not found", id)
			}
			err = h.credential.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (h credentialHandler) GetToken(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			projectID := mux.Vars(r)["id"]
			id := mux.Vars(r)["cred_id"]
			ctx := r.Context()
			credential, err := h.credential.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			if credential.ProjectID != projectID {
				return nil, errors.NotFound("credential with id='%s' not found", id)
			}

			return PresentCredentialToken(credential), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
