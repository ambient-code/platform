package roleBindings

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	pkgrbac "github.com/ambient-code/platform/components/ambient-api-server/pkg/rbac"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
	"gorm.io/gorm"
)

var _ handlers.RestHandler = roleBindingHandler{}

type roleBindingHandler struct {
	roleBinding    RoleBindingService
	generic        services.GenericService
	sessionFactory *db.SessionFactory
}

func NewRoleBindingHandler(roleBinding RoleBindingService, generic services.GenericService, sessionFactory *db.SessionFactory) *roleBindingHandler {
	return &roleBindingHandler{
		roleBinding:    roleBinding,
		generic:        generic,
		sessionFactory: sessionFactory,
	}
}

func (h roleBindingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var roleBinding openapi.RoleBinding
	cfg := &handlers.HandlerConfig{
		Body: &roleBinding,
		Validators: []handlers.Validate{
			handlers.ValidateEmpty(&roleBinding, "Id", "id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			// --- Escalation prevention ---
			if h.sessionFactory == nil {
				return nil, errors.Forbidden("authorization not available")
			}

			validScopes := map[string]bool{"global": true, "project": true, "agent": true, "session": true, "credential": true}
			if !validScopes[roleBinding.Scope] {
				return nil, errors.BadRequest("invalid scope")
			}

			{
				g := (*h.sessionFactory).New(ctx)

				// a) Look up target role name
				var targetRoleName string
				if err := g.Table("roles").Select("name").Where("id = ? AND deleted_at IS NULL", roleBinding.RoleId).Scan(&targetRoleName).Error; err != nil || targetRoleName == "" {
					return nil, errors.Forbidden("target role not found")
				}

				// b) Level hierarchy check — scoped to the target resource
				username := auth.GetUsernameFromContext(ctx)
				var callerRoleNames []string
				baseQuery := func(g *gorm.DB) *gorm.DB {
					return g.Table("role_bindings rb").
						Select("r.name").
						Joins("JOIN roles r ON r.id = rb.role_id").
						Where("rb.user_id = ? AND r.deleted_at IS NULL AND rb.deleted_at IS NULL", username)
				}
				var scanErr error
				if roleBinding.Scope == "project" && roleBinding.ProjectId.IsSet() {
					scanErr = baseQuery(g).Where("rb.project_id = ? OR rb.scope = 'global'", *roleBinding.ProjectId.Get()).Scan(&callerRoleNames).Error
				} else if roleBinding.Scope == "credential" && roleBinding.CredentialId.IsSet() {
					scanErr = baseQuery(g).Where("rb.credential_id = ? OR rb.scope = 'global'", *roleBinding.CredentialId.Get()).Scan(&callerRoleNames).Error
				} else {
					scanErr = baseQuery(g).Scan(&callerRoleNames).Error
				}
				if scanErr != nil {
					return nil, errors.GeneralError("authorization check failed")
				}
				callerLevel := pkgrbac.HighestLevel(callerRoleNames)
				if pkgrbac.InternalRoles[targetRoleName] && callerLevel != 0 {
					return nil, errors.Forbidden("cannot assign internal role")
				}
				if !pkgrbac.CanGrant(callerLevel, targetRoleName) {
					return nil, errors.Forbidden("insufficient privileges to grant this role")
				}

				// b2) Global scope: only platform:admin can create global bindings
				if roleBinding.Scope == "global" && callerLevel != 0 {
					return nil, errors.Forbidden("only platform admins can create global bindings")
				}

				// b3) Project scope: caller must have a binding covering the target project
				if roleBinding.Scope == "project" && roleBinding.ProjectId.IsSet() {
					var projCount int64
					if dbErr := g.Table("role_bindings").
						Where("user_id = ? AND (project_id = ? OR scope = 'global') AND deleted_at IS NULL",
							username, *roleBinding.ProjectId.Get()).
						Count(&projCount).Error; dbErr != nil {
						return nil, errors.GeneralError("authorization check failed")
					}
					if projCount == 0 {
						return nil, errors.Forbidden("caller has no access to this project")
					}
				}

				// c) Credential scope authorization
				if roleBinding.Scope == "credential" && roleBinding.CredentialId.IsSet() {
					hasProjectID := roleBinding.ProjectId.IsSet() && roleBinding.ProjectId.Get() != nil
					hasAgentID := roleBinding.AgentId.IsSet() && roleBinding.AgentId.Get() != nil

					if hasProjectID && *roleBinding.ProjectId.Get() == "" {
						return nil, errors.BadRequest("project_id must not be empty")
					}
					if hasAgentID && *roleBinding.AgentId.Get() == "" {
						return nil, errors.BadRequest("agent_id must not be empty")
					}

					// c1) agent_id requires project_id
					if hasAgentID && !hasProjectID {
						return nil, errors.BadRequest("agent-scoped credential bindings require a project_id")
					}

					// c2) Validate agent belongs to the specified project
					if hasAgentID && hasProjectID {
						var agentProjectID string
						if dbErr := g.Table("agents").Select("project_id").
							Where("id = ? AND deleted_at IS NULL", *roleBinding.AgentId.Get()).
							Scan(&agentProjectID).Error; dbErr != nil || agentProjectID == "" {
							return nil, errors.BadRequest("agent not found")
						}
						if agentProjectID != *roleBinding.ProjectId.Get() {
							return nil, errors.BadRequest("agent does not belong to the specified project")
						}
					}

					// c3) Caller must be credential:owner
					var credOwnerCount int64
					if dbErr := g.Table("role_bindings").
						Joins("JOIN roles ON roles.id = role_bindings.role_id").
						Where("role_bindings.user_id = ? AND roles.name = ? AND role_bindings.credential_id = ? AND role_bindings.deleted_at IS NULL AND roles.deleted_at IS NULL",
							username, pkgrbac.RoleCredentialOwner, *roleBinding.CredentialId.Get()).
						Count(&credOwnerCount).Error; dbErr != nil {
						return nil, errors.GeneralError("authorization check failed")
					}
					if credOwnerCount == 0 {
						return nil, errors.Forbidden("caller must be credential owner to grant credential-scoped bindings")
					}

					// c4) Project-level or agent-level: caller needs project:editor or higher
					if hasProjectID && callerLevel != 0 {
						var projEditorCount int64
						if dbErr := g.Table("role_bindings").
							Joins("JOIN roles ON roles.id = role_bindings.role_id").
							Where("role_bindings.user_id = ? AND role_bindings.project_id = ? AND role_bindings.deleted_at IS NULL AND roles.deleted_at IS NULL",
								username, *roleBinding.ProjectId.Get()).
							Where("roles.name IN ?", []string{pkgrbac.RoleProjectOwner, pkgrbac.RoleProjectEditor}).
							Count(&projEditorCount).Error; dbErr != nil {
							return nil, errors.GeneralError("authorization check failed")
						}
						if projEditorCount == 0 {
							return nil, errors.Forbidden("caller must be project editor or higher to bind credentials to a project")
						}
					}

					// c5) Global credential binding: requires platform:admin
					if !hasProjectID && !hasAgentID {
						if callerLevel != 0 {
							return nil, errors.Forbidden("only platform admins can create global credential bindings")
						}
					}
				}
			}

			roleBindingModel := ConvertRoleBinding(roleBinding)
			roleBindingModel, err := h.roleBinding.Create(ctx, roleBindingModel)
			if err != nil {
				return nil, err
			}
			return PresentRoleBinding(roleBindingModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusCreated)
}

func (h roleBindingHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch openapi.RoleBindingPatchRequest

	cfg := &handlers.HandlerConfig{
		Body:       &patch,
		Validators: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.roleBinding.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			// --- Escalation prevention ---
			username := auth.GetUsernameFromContext(ctx)

			if h.sessionFactory == nil {
				return nil, errors.Forbidden("authorization not available")
			}
			{
				g := (*h.sessionFactory).New(ctx)

				var callerRoleNames []string
				if dbErr := g.Table("role_bindings rb").
					Select("r.name").
					Joins("JOIN roles r ON r.id = rb.role_id").
					Where("rb.user_id = ? AND r.deleted_at IS NULL AND rb.deleted_at IS NULL", username).
					Scan(&callerRoleNames).Error; dbErr != nil {
					return nil, errors.GeneralError("authorization check failed")
				}
				callerLevel := pkgrbac.HighestLevel(callerRoleNames)

				// Non-admin callers can only PATCH their own bindings.
				isOwner := found.UserId != nil && *found.UserId == username
				if callerLevel != 0 && !isOwner {
					return nil, errors.Forbidden("Forbidden")
				}

				// Prevent changing role_id to a role the caller cannot grant.
				if patch.RoleId != nil && *patch.RoleId != found.RoleId {
					var targetRoleName string
					if dbErr := g.Table("roles").Select("name").Where("id = ? AND deleted_at IS NULL", *patch.RoleId).Scan(&targetRoleName).Error; dbErr != nil || targetRoleName == "" {
						return nil, errors.Forbidden("target role not found")
					}
					if pkgrbac.InternalRoles[targetRoleName] {
						return nil, errors.Forbidden("cannot assign internal role")
					}
					if !pkgrbac.CanGrant(callerLevel, targetRoleName) {
						return nil, errors.Forbidden("insufficient privileges to change role")
					}
				}

				// Prevent changing user_id (ownership transfer).
				if patch.UserId.IsSet() {
					patchVal := patch.UserId.Get()
					if found.UserId == nil || patchVal == nil || *patchVal != *found.UserId {
						if callerLevel != 0 {
							return nil, errors.Forbidden("Forbidden")
						}
					}
				}

				// Prevent scope widening — non-admins cannot change scope FKs.
				if callerLevel != 0 {
					if patch.Scope != nil && *patch.Scope != found.Scope {
						return nil, errors.Forbidden("Forbidden")
					}
					if patch.ProjectId.IsSet() {
						patchVal := patch.ProjectId.Get()
						if found.ProjectId == nil || patchVal == nil || *patchVal != *found.ProjectId {
							return nil, errors.Forbidden("Forbidden")
						}
					}
					if patch.AgentId.IsSet() {
						patchVal := patch.AgentId.Get()
						if found.AgentId == nil || patchVal == nil || *patchVal != *found.AgentId {
							return nil, errors.Forbidden("Forbidden")
						}
					}
					if patch.SessionId.IsSet() {
						patchVal := patch.SessionId.Get()
						if found.SessionId == nil || patchVal == nil || *patchVal != *found.SessionId {
							return nil, errors.Forbidden("Forbidden")
						}
					}
					if patch.CredentialId.IsSet() {
						patchVal := patch.CredentialId.Get()
						if found.CredentialId == nil || patchVal == nil || *patchVal != *found.CredentialId {
							return nil, errors.Forbidden("Forbidden")
						}
					}
				}
			}

			if patch.RoleId != nil {
				found.RoleId = *patch.RoleId
			}
			if patch.Scope != nil {
				found.Scope = *patch.Scope
			}
			if patch.UserId.IsSet() {
				found.UserId = patch.UserId.Get()
			}
			if patch.ProjectId.IsSet() {
				found.ProjectId = patch.ProjectId.Get()
			}
			if patch.AgentId.IsSet() {
				found.AgentId = patch.AgentId.Get()
			}
			if patch.SessionId.IsSet() {
				found.SessionId = patch.SessionId.Get()
			}
			if patch.CredentialId.IsSet() {
				found.CredentialId = patch.CredentialId.Get()
			}

			roleBindingModel, err := h.roleBinding.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return PresentRoleBinding(roleBindingModel), nil
		},
		ErrorHandler: handlers.HandleError,
	}

	handlers.Handle(w, r, cfg, http.StatusOK)
}

func (h roleBindingHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())

			authResult := pkgrbac.GetAuthResult(ctx)
			if authResult != nil && !authResult.IsGlobalAdmin {
				username := auth.GetUsernameFromContext(ctx)
				// Show bindings where:
				// 1. user_id matches caller (own bindings), OR
				// 2. project_id is in caller's authorized projects (team bindings), OR
				// 3. credential_id is in caller's authorized credentials
				userFilter, err := pkgrbac.TSLEqual("user_id", username)
				if err != nil {
					return nil, errors.Forbidden("invalid username")
				}
				scopeFilter := userFilter

				if len(authResult.ProjectIDs) > 0 {
					projFilter, err := pkgrbac.TSLIn("project_id", authResult.ProjectIDs)
					if err != nil {
						return nil, errors.Forbidden("invalid project id")
					}
					scopeFilter = pkgrbac.TSLOr(scopeFilter, projFilter)
				}

				if len(authResult.CredentialIDs) > 0 {
					credFilter, err := pkgrbac.TSLIn("credential_id", authResult.CredentialIDs)
					if err != nil {
						return nil, errors.Forbidden("invalid credential id")
					}
					scopeFilter = pkgrbac.TSLOr(scopeFilter, credFilter)
				}

				pkgrbac.AppendTSLFilter(listArgs, scopeFilter)
			}

			var roleBindings []RoleBinding
			paging, err := h.generic.List(ctx, "id", listArgs, &roleBindings)
			if err != nil {
				return nil, err
			}
			roleBindingList := openapi.RoleBindingList{
				Kind:  "RoleBindingList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.RoleBinding{},
			}

			for _, roleBinding := range roleBindings {
				converted := PresentRoleBinding(&roleBinding)
				roleBindingList.Items = append(roleBindingList.Items, converted)
			}
			if listArgs.Fields != nil {
				filteredItems, err := presenters.SliceFilter(listArgs.Fields, roleBindingList.Items)
				if err != nil {
					return nil, err
				}
				return filteredItems, nil
			}
			return roleBindingList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h roleBindingHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			roleBinding, err := h.roleBinding.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return PresentRoleBinding(roleBinding), nil
		},
	}

	handlers.HandleGet(w, r, cfg)
}

func (h roleBindingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()

			// --- Last-owner protection ---
			if h.sessionFactory == nil {
				return nil, errors.Forbidden("authorization not available")
			}
			{
				binding, getErr := h.roleBinding.Get(ctx, id)
				if getErr != nil {
					return nil, getErr
				}

				var roleName string
				g := (*h.sessionFactory).New(ctx)
				if dbErr := g.Table("roles").Select("name").Where("id = ? AND deleted_at IS NULL", binding.RoleId).Scan(&roleName).Error; dbErr != nil {
					return nil, errors.GeneralError("authorization check failed")
				}

				if roleName == pkgrbac.RoleProjectOwner && binding.ProjectId != nil {
					var count int64
					if dbErr := g.Table("role_bindings").
						Where("role_id = ? AND project_id = ? AND deleted_at IS NULL",
							binding.RoleId, *binding.ProjectId).
						Count(&count).Error; dbErr != nil {
						return nil, errors.GeneralError("authorization check failed")
					}
					if count <= 1 {
						return nil, errors.New(errors.ErrorConflict, "cannot delete the last owner binding")
					}
				}
				if roleName == pkgrbac.RoleCredentialOwner && binding.CredentialId != nil {
					var count int64
					if dbErr := g.Table("role_bindings").
						Where("role_id = ? AND credential_id = ? AND deleted_at IS NULL",
							binding.RoleId, *binding.CredentialId).
						Count(&count).Error; dbErr != nil {
						return nil, errors.GeneralError("authorization check failed")
					}
					if count <= 1 {
						return nil, errors.New(errors.ErrorConflict, "cannot delete the last owner binding")
					}
				}

				// --- Authorization check ---
				username := auth.GetUsernameFromContext(ctx)

				if binding.Scope == "credential" {
					// Asymmetric unbind: project:editor+ can remove credential bindings
					// from their project without needing credential:owner.
					// platform:admin can always unbind.
					var callerAllRoles []string
					if dbErr := g.Table("role_bindings rb").
						Select("r.name").
						Joins("JOIN roles r ON r.id = rb.role_id").
						Where("rb.user_id = ? AND r.deleted_at IS NULL AND rb.deleted_at IS NULL", username).
						Scan(&callerAllRoles).Error; dbErr != nil {
						return nil, errors.GeneralError("authorization check failed")
					}
					callerLevel := pkgrbac.HighestLevel(callerAllRoles)

					if callerLevel == 0 {
						// platform:admin can always unbind
					} else if binding.ProjectId != nil {
						var projEditorCount int64
						if dbErr := g.Table("role_bindings").
							Joins("JOIN roles ON roles.id = role_bindings.role_id").
							Where("role_bindings.user_id = ? AND role_bindings.project_id = ? AND role_bindings.deleted_at IS NULL AND roles.deleted_at IS NULL",
								username, *binding.ProjectId).
							Where("roles.name IN ?", []string{pkgrbac.RoleProjectOwner, pkgrbac.RoleProjectEditor}).
							Count(&projEditorCount).Error; dbErr != nil {
							return nil, errors.GeneralError("authorization check failed")
						}
						if projEditorCount == 0 {
							return nil, errors.Forbidden("insufficient privileges to delete this binding")
						}
					} else {
						// Global credential binding: requires platform:admin (already checked above)
						return nil, errors.Forbidden("insufficient privileges to delete this binding")
					}
				} else {
					// Non-credential scopes: caller must outrank the binding's role
					// AND be at least project:owner (level 1)
					var callerRoleNames []string
					baseQuery := g.Table("role_bindings rb").
						Select("r.name").
						Joins("JOIN roles r ON r.id = rb.role_id").
						Where("rb.user_id = ? AND r.deleted_at IS NULL AND rb.deleted_at IS NULL", username)
					if binding.Scope == "project" && binding.ProjectId != nil {
						baseQuery = baseQuery.Where("rb.project_id = ? OR rb.scope = 'global'", *binding.ProjectId)
					}
					if dbErr := baseQuery.Scan(&callerRoleNames).Error; dbErr != nil {
						return nil, errors.GeneralError("authorization check failed")
					}
					callerLevel := pkgrbac.HighestLevel(callerRoleNames)
					if callerLevel > 1 || !pkgrbac.CanGrant(callerLevel, roleName) {
						return nil, errors.Forbidden("insufficient privileges to delete this binding")
					}
				}
			}

			err := h.roleBinding.Delete(ctx, id)
			if err != nil {
				return nil, err
			}
			return nil, nil
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}
