package credentials

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

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
			credentialModel := ConvertCredential(credential)
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
			id := mux.Vars(r)["id"]
			found, err := h.credential.Get(ctx, id)
			if err != nil {
				return nil, err
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

			listArgs := services.NewListArguments(r.URL.Query())
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
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			credential, err := h.credential.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentCredential(credential), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h credentialHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.credential.Delete(ctx, id)
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
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			credential, err := h.credential.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentCredentialToken(credential), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}
