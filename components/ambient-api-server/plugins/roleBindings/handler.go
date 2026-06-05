package roleBindings

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	pkgrbac "github.com/ambient-code/platform/components/ambient-api-server/pkg/rbac"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/handlers"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
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
			if h.sessionFactory != nil {
				g := (*h.sessionFactory).New(ctx)

				// a) Look up target role name and reject internal roles
				var targetRoleName string
				if err := g.Raw("SELECT name FROM roles WHERE id = ? AND deleted_at IS NULL", roleBinding.RoleId).Scan(&targetRoleName).Error; err != nil || targetRoleName == "" {
					return nil, errors.Forbidden("target role not found")
				}
				if pkgrbac.InternalRoles[targetRoleName] {
					return nil, errors.Forbidden("cannot assign internal role")
				}

				// b) Level hierarchy check — scoped to the target resource
				username := auth.GetUsernameFromContext(ctx)
				var callerRoleNames []string
				if roleBinding.Scope == "project" && roleBinding.ProjectId != nil {
					g.Raw(`SELECT r.name FROM role_bindings rb
						   JOIN roles r ON r.id = rb.role_id
						   WHERE rb.user_id = ? AND (rb.project_id = ? OR rb.scope = 'global')
						   AND r.deleted_at IS NULL AND rb.deleted_at IS NULL`,
						username, *roleBinding.ProjectId).Scan(&callerRoleNames)
				} else if roleBinding.Scope == "credential" && roleBinding.CredentialId != nil {
					g.Raw(`SELECT r.name FROM role_bindings rb
						   JOIN roles r ON r.id = rb.role_id
						   WHERE rb.user_id = ? AND (rb.credential_id = ? OR rb.scope = 'global')
						   AND r.deleted_at IS NULL AND rb.deleted_at IS NULL`,
						username, *roleBinding.CredentialId).Scan(&callerRoleNames)
				} else {
					g.Raw(`SELECT r.name FROM role_bindings rb
						   JOIN roles r ON r.id = rb.role_id
						   WHERE rb.user_id = ? AND r.deleted_at IS NULL AND rb.deleted_at IS NULL`,
						username).Scan(&callerRoleNames)
				}
				callerLevel := pkgrbac.HighestLevel(callerRoleNames)
				if !pkgrbac.CanGrant(callerLevel, targetRoleName) {
					return nil, errors.Forbidden("insufficient privileges to grant this role")
				}

				// b2) Global scope: only platform:admin can create global bindings
				if roleBinding.Scope == "global" && callerLevel != 0 {
					return nil, errors.Forbidden("only platform admins can create global bindings")
				}

				// b3) Project scope: caller must have a binding covering the target project
				if roleBinding.Scope == "project" && roleBinding.ProjectId != nil {
					var projCount int64
					g.Raw(`SELECT COUNT(*) FROM role_bindings rb
						   WHERE rb.user_id = ?
						   AND (rb.project_id = ? OR rb.scope = 'global')
						   AND rb.deleted_at IS NULL`,
						username, *roleBinding.ProjectId).Scan(&projCount)
					if projCount == 0 {
						return nil, errors.Forbidden("caller has no access to this project")
					}
				}

				// c) Credential scope: caller must be credential:owner AND project:owner
				if roleBinding.Scope == "credential" && roleBinding.CredentialId != nil {
					var credOwnerCount int64
					g.Raw(`SELECT COUNT(*) FROM role_bindings rb
						   JOIN roles r ON r.id = rb.role_id
						   WHERE rb.user_id = ? AND r.name = ?
						   AND rb.credential_id = ? AND rb.deleted_at IS NULL AND r.deleted_at IS NULL`,
						username, pkgrbac.RoleCredentialOwner, *roleBinding.CredentialId).Scan(&credOwnerCount)
					if credOwnerCount == 0 {
						return nil, errors.Forbidden("caller must be credential owner to grant credential-scoped bindings")
					}
					if roleBinding.ProjectId != nil {
						var projOwnerCount int64
						g.Raw(`SELECT COUNT(*) FROM role_bindings rb
							   JOIN roles r ON r.id = rb.role_id
							   WHERE rb.user_id = ? AND r.name = 'project:owner'
							   AND rb.project_id = ? AND rb.deleted_at IS NULL AND r.deleted_at IS NULL`,
							username, *roleBinding.ProjectId).Scan(&projOwnerCount)
						if projOwnerCount == 0 {
							return nil, errors.Forbidden("caller must be project owner to bind credentials to a project")
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

			if h.sessionFactory != nil {
				g := (*h.sessionFactory).New(ctx)

				var callerRoleNames []string
				g.Raw(`SELECT r.name FROM role_bindings rb
					   JOIN roles r ON r.id = rb.role_id
					   WHERE rb.user_id = ? AND r.deleted_at IS NULL AND rb.deleted_at IS NULL`,
					username).Scan(&callerRoleNames)
				callerLevel := pkgrbac.HighestLevel(callerRoleNames)

				// Non-admin callers can only PATCH their own bindings.
				isOwner := found.UserId != nil && *found.UserId == username
				if callerLevel != 0 && !isOwner {
					return nil, errors.Forbidden("Forbidden")
				}

				// Prevent changing role_id to a role the caller cannot grant.
				if patch.RoleId != nil && *patch.RoleId != found.RoleId {
					var targetRoleName string
					if dbErr := g.Raw("SELECT name FROM roles WHERE id = ? AND deleted_at IS NULL", *patch.RoleId).Scan(&targetRoleName).Error; dbErr != nil || targetRoleName == "" {
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
				if patch.UserId != nil && (found.UserId == nil || *patch.UserId != *found.UserId) {
					if callerLevel != 0 {
						return nil, errors.Forbidden("Forbidden")
					}
				}
			}

			if patch.RoleId != nil {
				found.RoleId = *patch.RoleId
			}
			if patch.Scope != nil {
				found.Scope = *patch.Scope
			}
			if patch.UserId != nil {
				found.UserId = patch.UserId
			}
			if patch.ProjectId != nil {
				found.ProjectId = patch.ProjectId
			}
			if patch.AgentId != nil {
				found.AgentId = patch.AgentId
			}
			if patch.SessionId != nil {
				found.SessionId = patch.SessionId
			}
			if patch.CredentialId != nil {
				found.CredentialId = patch.CredentialId
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
				var conditions []string
				conditions = append(conditions, fmt.Sprintf("user_id = '%s'", strings.ReplaceAll(username, "'", "''")))

				if len(authResult.ProjectIDs) > 0 {
					quoted := make([]string, len(authResult.ProjectIDs))
					for i, id := range authResult.ProjectIDs {
						quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(id, "'", "''"))
					}
					conditions = append(conditions, fmt.Sprintf("project_id in (%s)", strings.Join(quoted, ",")))
				}

				if len(authResult.CredentialIDs) > 0 {
					quoted := make([]string, len(authResult.CredentialIDs))
					for i, id := range authResult.CredentialIDs {
						quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(id, "'", "''"))
					}
					conditions = append(conditions, fmt.Sprintf("credential_id in (%s)", strings.Join(quoted, ",")))
				}

				scopeFilter := strings.Join(conditions, " or ")
				if listArgs.Search != "" {
					listArgs.Search = fmt.Sprintf("(%s) and (%s)", listArgs.Search, scopeFilter)
				} else {
					listArgs.Search = scopeFilter
				}
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
			if h.sessionFactory != nil {
				binding, getErr := h.roleBinding.Get(ctx, id)
				if getErr != nil {
					return nil, getErr
				}

				var roleName string
				g := (*h.sessionFactory).New(ctx)
				g.Raw("SELECT name FROM roles WHERE id = ? AND deleted_at IS NULL", binding.RoleId).Scan(&roleName)

				if roleName == pkgrbac.RoleProjectOwner && binding.ProjectId != nil {
					var count int64
					g.Raw(`SELECT COUNT(*) FROM role_bindings
						   WHERE role_id = ? AND project_id = ? AND deleted_at IS NULL`,
						binding.RoleId, *binding.ProjectId).Scan(&count)
					if count <= 1 {
						return nil, errors.New(errors.ErrorConflict, "cannot delete the last owner binding")
					}
				}
				if roleName == pkgrbac.RoleCredentialOwner && binding.CredentialId != nil {
					var count int64
					g.Raw(`SELECT COUNT(*) FROM role_bindings
						   WHERE role_id = ? AND credential_id = ? AND deleted_at IS NULL`,
						binding.RoleId, *binding.CredentialId).Scan(&count)
					if count <= 1 {
						return nil, errors.New(errors.ErrorConflict, "cannot delete the last owner binding")
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
