package skills

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var _ handlers.RestHandler = skillHandler{}

type skillHandler struct {
	skill   SkillService
	generic services.GenericService
}

func NewSkillHandler(skill SkillService, generic services.GenericService) *skillHandler {
	return &skillHandler{
		skill:   skill,
		generic: generic,
	}
}

func (h skillHandler) Create(w http.ResponseWriter, r *http.Request) {
	var skill openapi.Skill
	cfg := &handlers.HandlerConfig{
		Body: &skill,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&skill, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			skillModel := ConvertSkill(skill)
			skillModel, err := h.skill.Create(ctx, skillModel)
			if err != nil {
				return nil, err
			}
			return PresentSkill(skillModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h skillHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.SkillPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.skill.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Name != nil {
				found.Name = *patch.Name
			}
			if patch.RepoUrl != nil {
				found.RepoUrl = patch.RepoUrl
			}
			if patch.Prompt != nil {
				found.Prompt = patch.Prompt
			}
			if patch.ProjectId != nil {
				found.ProjectId = patch.ProjectId
			}

			skillModel, err := h.skill.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentSkill(skillModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h skillHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			var skills []Skill
			paging, err := h.generic.List(ctx, "id", listArgs, &skills)
			if err != nil {
				return nil, err
			}
			skillList := openapi.SkillList{
				Kind:  "SkillList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Skill{},
			}

			for _, skill := range skills {
				converted := PresentSkill(&skill)
				skillList.Items = append(skillList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, skillList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return skillList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h skillHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			skill, err := h.skill.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentSkill(skill), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h skillHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.skill.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
