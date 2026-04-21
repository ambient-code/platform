package repoIntelligences

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/common"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/repoFindings"
)

type repoIntelligenceHandler struct {
	service         RepoIntelligenceService
	generic         services.GenericService
	findingsGeneric services.GenericService
}

func NewRepoIntelligenceHandler(svc RepoIntelligenceService, generic services.GenericService, findingsGeneric services.GenericService) *repoIntelligenceHandler {
	return &repoIntelligenceHandler{
		service:         svc,
		generic:         generic,
		findingsGeneric: findingsGeneric,
	}
}

func (h repoIntelligenceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body RepoIntelligenceAPI
	cfg := &handlers.HandlerConfig{
		Body: &body,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&body, "ID", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			model := ConvertRepoIntelligence(body)
			model, err := h.service.Create(ctx, model)
			if err != nil {
				return nil, err
			}
			return PresentRepoIntelligence(model), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h repoIntelligenceHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch RepoIntelligencePatchRequest
	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			if patch.Summary != nil {
				found.Summary = *patch.Summary
			}
			if patch.Language != nil {
				found.Language = *patch.Language
			}
			if patch.Framework != nil {
				found.Framework = patch.Framework
			}
			if patch.BuildSystem != nil {
				found.BuildSystem = patch.BuildSystem
			}
			if patch.TestStrategy != nil {
				found.TestStrategy = patch.TestStrategy
			}
			if patch.Architecture != nil {
				found.Architecture = patch.Architecture
			}
			if patch.Conventions != nil {
				found.Conventions = patch.Conventions
			}
			if patch.Caveats != nil {
				found.Caveats = patch.Caveats
			}
			if patch.Confidence != nil {
				found.Confidence = patch.Confidence
			}
			if patch.RepoBranch != nil {
				found.RepoBranch = *patch.RepoBranch
			}

			found.Version++

			model, err := h.service.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentRepoIntelligence(model), nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h repoIntelligenceHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := services.NewListArguments(r.URL.Query())

			if serr := common.ApplyProjectScope(r, listArgs); serr != nil {
				return nil, serr
			}

			var items []RepoIntelligence
			paging, err := h.generic.List(ctx, "id", listArgs, &items)
			if err != nil {
				return nil, err
			}

			list := RepoIntelligenceListAPI{
				Kind:  "RepoIntelligenceList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []RepoIntelligenceAPI{},
			}
			for _, item := range items {
				list.Items = append(list.Items, PresentRepoIntelligence(&item))
			}
			return list, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

func (h repoIntelligenceHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			ri, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			return PresentRepoIntelligence(ri), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h repoIntelligenceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			err := h.service.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (h repoIntelligenceHandler) DeleteByLookup(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			projectID := r.URL.Query().Get("project_id")
			repoURL := r.URL.Query().Get("repo_url")
			if projectID == "" || repoURL == "" {
				return nil, errors.Validation("project_id and repo_url query parameters are required")
			}
			ctx := r.Context()
			ri, err := h.service.GetByProjectAndRepo(ctx, projectID, repoURL)
			if err != nil {
				return nil, err
			}
			deleteErr := h.service.Delete(ctx, ri.ID)
			if deleteErr != nil {
				return nil, deleteErr
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func (h repoIntelligenceHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			projectID := r.URL.Query().Get("project_id")
			repoURL := r.URL.Query().Get("repo_url")
			if projectID == "" || repoURL == "" {
				return nil, errors.Validation("project_id and repo_url query parameters are required")
			}
			ctx := r.Context()
			ri, err := h.service.GetByProjectAndRepo(ctx, projectID, repoURL)
			if err != nil {
				return nil, err
			}
			return PresentRepoIntelligence(ri), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h repoIntelligenceHandler) ListFindings(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()

			// Verify the intelligence record exists
			_, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			listArgs := services.NewListArguments(r.URL.Query())

			// Inject parent filter into search (follows agents/subresource_handler pattern)
			filter := fmt.Sprintf("intelligence_id = '%s'", id)
			if listArgs.Search != "" {
				listArgs.Search = filter + " and (" + listArgs.Search + ")"
			} else {
				listArgs.Search = filter
			}

			var items []repoFindings.RepoFinding
			paging, serr := h.findingsGeneric.List(ctx, "id", listArgs, &items)
			if serr != nil {
				return nil, serr
			}

			list := repoFindings.RepoFindingListAPI{
				Kind:  "RepoFindingList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []repoFindings.RepoFindingAPI{},
			}
			for _, item := range items {
				list.Items = append(list.Items, repoFindings.PresentRepoFinding(&item))
			}
			return list, nil
		},
	}
	handlers.HandleList(w, r, cfg)
}

func (h repoIntelligenceHandler) Context(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			projectID := r.URL.Query().Get("project_id")
			repoURLsParam := r.URL.Query().Get("repo_urls")
			if projectID == "" || repoURLsParam == "" {
				return nil, errors.Validation("project_id and repo_urls query parameters are required")
			}

			repoURLs := strings.Split(repoURLsParam, ",")
			maxEntries, _ := strconv.Atoi(r.URL.Query().Get("max_entries"))
			if maxEntries <= 0 {
				maxEntries = 20
			}

			ctx := r.Context()
			var allIntel []RepoIntelligenceAPI
			var allFindings []repoFindings.RepoFindingAPI

			for _, repoURL := range repoURLs {
				repoURL = strings.TrimSpace(repoURL)
				if repoURL == "" {
					continue
				}

				intel, err := h.service.GetByProjectAndRepo(ctx, projectID, repoURL)
				if err != nil {
					continue // skip repos without intelligence
				}
				allIntel = append(allIntel, PresentRepoIntelligence(intel))

				// Fetch active findings for this intelligence, capped at maxEntries
				listArgs := services.NewListArguments(r.URL.Query())
				listArgs.Search = fmt.Sprintf("intelligence_id = '%s' and status = 'active'", intel.ID)
				listArgs.Size = int64(maxEntries)

				var findings []repoFindings.RepoFinding
				_, serr := h.findingsGeneric.List(ctx, "id", listArgs, &findings)
				if serr != nil {
					continue
				}
				for _, f := range findings {
					allFindings = append(allFindings, repoFindings.PresentRepoFinding(&f))
				}
			}

			if allIntel == nil {
				allIntel = []RepoIntelligenceAPI{}
			}
			if allFindings == nil {
				allFindings = []repoFindings.RepoFindingAPI{}
			}

			return map[string]interface{}{
				"intelligences":   allIntel,
				"findings":        allFindings,
				"injected_context": buildInjectedContext(allIntel, allFindings),
			}, nil
		},
		ErrorHandler: handlers.HandleError,
	}
	handlers.HandleGet(w, r, cfg)
}

func buildInjectedContext(intels []RepoIntelligenceAPI, findings []repoFindings.RepoFindingAPI) string {
	if len(intels) == 0 && len(findings) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<!-- BEGIN PROJECT MEMORY (auto-injected) -->\n")
	sb.WriteString("## Project Intelligence\n\n")

	for _, intel := range intels {
		sb.WriteString(fmt.Sprintf("### %s (%s)\n", intel.RepoURL, intel.Language))
		sb.WriteString(fmt.Sprintf("%s\n", intel.Summary))
		if intel.Caveats != nil && *intel.Caveats != "" {
			sb.WriteString(fmt.Sprintf("**Caveats:** %s\n", *intel.Caveats))
		}
		sb.WriteString("\n")
	}

	if len(findings) > 0 {
		sb.WriteString("### Active Findings\n\n")
		for _, f := range findings {
			severity := "info"
			if f.Severity != nil {
				severity = *f.Severity
			}
			sb.WriteString(fmt.Sprintf("- **[%s]** %s (`%s`): %s\n", severity, f.Title, f.FilePath, f.Body))
		}
	}

	sb.WriteString("<!-- END PROJECT MEMORY -->\n")
	return sb.String()
}
