package workflowSkills

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = workflowSkillHandler{}

type workflowSkillHandler struct {
	workflowSkill WorkflowSkillService
	generic       services.GenericService
}

func NewWorkflowSkillHandler(workflowSkill WorkflowSkillService, generic services.GenericService) *workflowSkillHandler {
	return &workflowSkillHandler{
		workflowSkill: workflowSkill,
		generic:       generic,
	}
}

func (h workflowSkillHandler) Create(w http.ResponseWriter, r *http.Request) {
	var workflowSkill openapi.WorkflowSkill
	cfg := &handlers.HandlerConfig{
		Body: &workflowSkill,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&workflowSkill, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			workflowSkillModel := ConvertWorkflowSkill(workflowSkill)
			workflowSkillModel, err := h.workflowSkill.Create(ctx, workflowSkillModel)
			if err != nil {
				return nil, err
			}
			return PresentWorkflowSkill(workflowSkillModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h workflowSkillHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.WorkflowSkillPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.workflowSkill.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.WorkflowId != nil {
				found.WorkflowId = *patch.WorkflowId
			}
			if patch.SkillId != nil {
				found.SkillId = *patch.SkillId
			}
			if patch.Position != nil {
				found.Position = int(*patch.Position)
			}

			workflowSkillModel, err := h.workflowSkill.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentWorkflowSkill(workflowSkillModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h workflowSkillHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var workflowSkills []WorkflowSkill
			paging, err := h.generic.List(ctx, "id", listArgs, &workflowSkills)
			if err != nil {
				return nil, err
			}
			workflowSkillList := openapi.WorkflowSkillList{
				Kind:  "WorkflowSkillList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.WorkflowSkill{},
			}

			for _, workflowSkill := range workflowSkills {
				converted := PresentWorkflowSkill(&workflowSkill)
				workflowSkillList.Items = append(workflowSkillList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, workflowSkillList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return workflowSkillList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h workflowSkillHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			workflowSkill, err := h.workflowSkill.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentWorkflowSkill(workflowSkill), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h workflowSkillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.workflowSkill.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
