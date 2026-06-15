package rbac

import (
	"context"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"gorm.io/gorm/clause"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/middleware"
)

// userRow is a local struct for GORM auto-provision inserts, avoiding
// circular imports with the users plugin package.
type userRow struct {
	ID        string `gorm:"primaryKey"`
	Username  string `gorm:"uniqueIndex:idx_users_username_active"`
	Name      string
	Email     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (userRow) TableName() string { return "users" }

type DBAuthorizationMiddleware struct {
	evaluator      *Evaluator
	sessionFactory *db.SessionFactory
	enableAuthz    bool
}

func NewDBAuthorizationMiddleware(sessionFactory *db.SessionFactory, enableAuthz bool) *DBAuthorizationMiddleware {
	return &DBAuthorizationMiddleware{
		evaluator:      NewEvaluator(sessionFactory),
		sessionFactory: sessionFactory,
		enableAuthz:    enableAuthz,
	}
}

func (m *DBAuthorizationMiddleware) AuthorizeApi(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Detect service caller from verified JWT username on the HTTP path.
		// On HTTP, GetAuthPayloadFromContext only succeeds for signature-verified
		// JWTs (upstream JWTHandler validates before we run). This prevents
		// forged JWTs from triggering auto-provisioning.
		if !middleware.IsServiceCaller(ctx) {
			payload, payloadErr := auth.GetAuthPayloadFromContext(ctx)
			if payloadErr == nil && payload != nil && payload.Username != "" &&
				middleware.IsConfiguredServiceAccount(payload.Username) {
				ctx = middleware.WithCallerType(ctx, middleware.CallerTypeService)
			}
		}

		// Service callers go through normal RBAC via their platform:admin
		// binding instead of bypassing. Auto-provision the user record
		// and the binding, then populate AuthResult from the evaluator.
		if middleware.IsServiceCaller(ctx) {
			username := auth.GetUsernameFromContext(ctx)
			if username == "" {
				// Raw service token (no JWT) — preserve legacy bypass
				// for backward compatibility with AMBIENT_API_TOKEN callers.
				// Set a well-defined AuthResult so handlers never see nil.
				ctx = SetAuthResult(ctx, &AuthResult{
					Username:      "service-token",
					IsGlobalAdmin: true,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			m.autoProvisionServiceAccount(ctx, username)
			projectIDs, isGlobal, projErr := m.evaluator.AuthorizedProjectIDs(ctx, username)
			if projErr != nil {
				http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
				return
			}
			credentialIDs, credGlobal, credErr := m.evaluator.AuthorizedCredentialIDs(ctx, username)
			if credErr != nil {
				http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
				return
			}
			ctx = SetAuthResult(ctx, &AuthResult{
				Username:      username,
				IsGlobalAdmin: isGlobal && credGlobal,
				ProjectIDs:    projectIDs,
				CredentialIDs: credentialIDs,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		m.autoProvisionUser(ctx)

		if isAuthExempt(r.Method, r.URL.Path) {
			username := auth.GetUsernameFromContext(ctx)
			ctx = SetAuthResult(ctx, &AuthResult{Username: username})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		if !m.enableAuthz {
			username := auth.GetUsernameFromContext(ctx)
			ctx = SetAuthResult(ctx, &AuthResult{
				Username:      username,
				IsGlobalAdmin: true,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		payload, err := auth.GetAuthPayloadFromContext(ctx)
		if err != nil || payload == nil || payload.Username == "" {
			http.Error(w, `{"kind":"Error","reason":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		username := payload.Username

		scope := ExtractRequestScope(r)
		resource := Resource(pathToResource(r.URL.Path))
		action := Action(pathToAction(r.Method, r.URL.Path))

		// Resolve scope for role_binding singleton operations.
		// The URL /role_bindings/{id} carries no project or credential context,
		// so we look up the binding's FK to populate the request scope.
		if resource == ResourceRoleBinding && scope.ProjectID == "" && scope.CredentialID == "" {
			if bindingID := mux.Vars(r)["id"]; bindingID != "" {
				m.resolveRoleBindingScope(ctx, bindingID, &scope)
			}
		}

		allowed, evalErr := m.evaluator.Evaluate(ctx, username, resource, action, scope)
		if evalErr != nil {
			http.Error(w, `{"kind":"Error","reason":"Internal Server Error"}`, http.StatusInternalServerError)
			return
		}

		if !allowed {
			if isListEndpoint(r.Method, r.URL.Path) {
				projectIDs, isGlobal, projErr := m.evaluator.AuthorizedProjectIDs(ctx, username)
				if projErr != nil {
					http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
					return
				}
				credentialIDs, credGlobal, credErr := m.evaluator.AuthorizedCredentialIDs(ctx, username)
				if credErr != nil {
					http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
					return
				}

				resolvedProjectID := scope.ProjectID
				if resolvedProjectID == "" && scope.SessionID != "" {
					if pid, err := m.evaluator.resolveSessionProject((*m.sessionFactory).New(ctx), scope.SessionID); err == nil {
						resolvedProjectID = pid
					}
				}

				if resolvedProjectID != "" && !isGlobal {
					found := false
					for _, pid := range projectIDs {
						if pid == resolvedProjectID {
							found = true
							break
						}
					}
					if !found {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"kind":"Error","reason":"Not Found"}`))
						return
					}
				}

				ctx = SetAuthResult(ctx, &AuthResult{
					Username:      username,
					IsGlobalAdmin: isGlobal && credGlobal,
					ProjectIDs:    projectIDs,
					CredentialIDs: credentialIDs,
				})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if isSingletonGet(r.Method, r.URL.Path) {
				// For session sub-resource GETs, resolve the session's project
				// and check whether the caller has a binding covering that
				// project.  This avoids relying solely on Evaluate() which
				// may fail when the request scope lacks a project_id.
				if scope.SessionID != "" {
					resolvedProjectID := scope.ProjectID
					if resolvedProjectID == "" {
						if pid, err := m.evaluator.resolveSessionProject((*m.sessionFactory).New(ctx), scope.SessionID); err == nil {
							resolvedProjectID = pid
						}
					}
					if resolvedProjectID != "" {
						projectIDs, isGlobal, projErr := m.evaluator.AuthorizedProjectIDs(ctx, username)
						if projErr != nil {
							http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
							return
						}
						credentialIDs, credGlobal, credErr := m.evaluator.AuthorizedCredentialIDs(ctx, username)
						if credErr != nil {
							http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
							return
						}
						hasAccess := isGlobal
						if !hasAccess {
							for _, pid := range projectIDs {
								if pid == resolvedProjectID {
									hasAccess = true
									break
								}
							}
						}
						if hasAccess {
							ctx = SetAuthResult(ctx, &AuthResult{
								Username:      username,
								IsGlobalAdmin: isGlobal && credGlobal,
								ProjectIDs:    projectIDs,
								CredentialIDs: credentialIDs,
							})
							next.ServeHTTP(w, r.WithContext(ctx))
							return
						}
					}
				}

				// For credential sub-resource GETs (e.g. /credentials/{id}/token),
				// check whether the caller has a binding covering the credential.
				if scope.CredentialID != "" {
					credentialIDs, credGlobal, credErr := m.evaluator.AuthorizedCredentialIDs(ctx, username)
					if credErr != nil {
						http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
						return
					}
					projectIDs, isGlobal, projErr := m.evaluator.AuthorizedProjectIDs(ctx, username)
					if projErr != nil {
						http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
						return
					}
					hasAccess := credGlobal
					if !hasAccess {
						for _, cid := range credentialIDs {
							if cid == scope.CredentialID {
								hasAccess = true
								break
							}
						}
					}
					if hasAccess {
						ctx = SetAuthResult(ctx, &AuthResult{
							Username:      username,
							IsGlobalAdmin: isGlobal && credGlobal,
							ProjectIDs:    projectIDs,
							CredentialIDs: credentialIDs,
						})
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"kind":"Error","reason":"Not Found"}`))
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"kind":"Error","reason":"Forbidden"}`))
			return
		}

		projectIDs, isGlobal, projErr := m.evaluator.AuthorizedProjectIDs(ctx, username)
		if projErr != nil {
			http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		credentialIDs, credGlobal, credErr := m.evaluator.AuthorizedCredentialIDs(ctx, username)
		if credErr != nil {
			http.Error(w, `{"kind":"Error","reason":"Service Unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		ctx = SetAuthResult(ctx, &AuthResult{
			Username:      username,
			IsGlobalAdmin: isGlobal && credGlobal,
			ProjectIDs:    projectIDs,
			CredentialIDs: credentialIDs,
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *DBAuthorizationMiddleware) autoProvisionUser(ctx context.Context) {
	var username, name, email string

	payload, err := auth.GetAuthPayloadFromContext(ctx)
	if err == nil && payload != nil && payload.Username != "" {
		username = payload.Username
		name = payload.FirstName
		if payload.LastName != "" {
			name = payload.FirstName + " " + payload.LastName
		}
		email = payload.Email
	} else {
		username = auth.GetUsernameFromContext(ctx)
		if username == "" {
			return
		}
		name = username
	}

	g := (*m.sessionFactory).New(ctx)
	now := time.Now()
	var emailPtr *string
	if email != "" {
		emailPtr = &email
	}
	row := userRow{
		ID:        api.NewID(),
		Username:  username,
		Name:      name,
		Email:     emailPtr,
		CreatedAt: now,
		UpdatedAt: now,
	}
	result := g.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if result.Error != nil {
		glog.Warningf("user auto-provision failed for %s: %v", username, result.Error)
	}
}

// autoProvisionServiceAccount upserts a User record for the service account
// and ensures it has a platform:admin global RoleBinding. This is the
// bootstrap mechanism that replaces the old RBAC bypass: the service account
// gains access through a real binding rather than skipping evaluation.
//
// The method is idempotent — concurrent requests will not duplicate records.
func (m *DBAuthorizationMiddleware) autoProvisionServiceAccount(ctx context.Context, username string) {
	g := (*m.sessionFactory).New(ctx)
	now := time.Now()

	// Upsert the user record.
	row := userRow{
		ID:        api.NewID(),
		Username:  username,
		Name:      username,
		CreatedAt: now,
		UpdatedAt: now,
	}
	result := g.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if result.Error != nil {
		glog.Warningf("service account user auto-provision failed for %s: %v", username, result.Error)
		return
	}

	// Check whether a platform:admin binding already exists.
	var count int64
	err := g.Table("role_bindings").
		Joins("JOIN roles ON roles.id = role_bindings.role_id").
		Where("role_bindings.user_id = ? AND role_bindings.deleted_at IS NULL", username).
		Where("roles.name = ? AND roles.deleted_at IS NULL", RolePlatformAdmin).
		Count(&count).Error
	if err != nil {
		glog.Warningf("service account binding check failed for %s: %v", username, err)
		return
	}
	if count > 0 {
		return // binding already exists
	}

	// Look up the platform:admin role ID.
	var roleID string
	err = g.Table("roles").
		Select("id").
		Where("name = ? AND deleted_at IS NULL", RolePlatformAdmin).
		Scan(&roleID).Error
	if err != nil || roleID == "" {
		glog.Warningf("service account binding: platform:admin role not found for %s", username)
		return
	}

	// Create the global binding.
	type bindingInsert struct {
		ID        string     `gorm:"primaryKey"`
		RoleID    string     `gorm:"column:role_id"`
		Scope     string     `gorm:"column:scope"`
		UserID    string     `gorm:"column:user_id"`
		CreatedAt time.Time  `gorm:"column:created_at"`
		UpdatedAt time.Time  `gorm:"column:updated_at"`
		DeletedAt *time.Time `gorm:"column:deleted_at"`
	}
	newBinding := bindingInsert{
		ID:        api.NewID(),
		RoleID:    roleID,
		Scope:     string(ScopeGlobal),
		UserID:    username,
		CreatedAt: now,
		UpdatedAt: now,
	}
	insertResult := g.Table("role_bindings").Create(&newBinding)
	if insertResult.Error != nil {
		glog.Warningf("service account binding creation failed for %s: %v", username, insertResult.Error)
		return
	}
	glog.Infof("auto-provisioned platform:admin binding for service account %s", username)
}

func httpMethodToAction(method string) string {
	switch method {
	case http.MethodGet:
		return "read"
	case http.MethodPost:
		return "create"
	case http.MethodPut, http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return "read"
	}
}

func pathToAction(method, path string) string {
	segments := splitPath(path)
	for i, seg := range segments {
		if seg == "v1" && i+2 < len(segments) {
			last := segments[len(segments)-1]
			switch last {
			case "token":
				return "fetch_token"
			case "start", "stop":
				return last
			case "sync", "refresh":
				return "update"
			}
		}
	}
	return httpMethodToAction(method)
}

func pathToResource(path string) string {
	segments := splitPath(path)
	for i, seg := range segments {
		if seg == "v1" && i+1 < len(segments) {
			resource := segments[i+1]
			if resource == "projects" && i+3 < len(segments) {
				resource = segments[i+3]
			}
			resource = singularize(resource)
			// Map sub-resource names to their parent permission resource.
			switch resource {
			case "scheduled-session":
				resource = "session"
			}
			return resource
		}
	}
	return "unknown"
}

func splitPath(path string) []string {
	trimmed := path
	if len(trimmed) > 0 && trimmed[0] == '/' {
		trimmed = trimmed[1:]
	}
	if trimmed == "" {
		return nil
	}
	parts := make([]string, 0, 8)
	for trimmed != "" {
		idx := 0
		for idx < len(trimmed) && trimmed[idx] != '/' {
			idx++
		}
		parts = append(parts, trimmed[:idx])
		if idx < len(trimmed) {
			trimmed = trimmed[idx+1:]
		} else {
			break
		}
	}
	return parts
}

func singularize(s string) string {
	if len(s) > 1 && s[len(s)-1] == 's' && s != "status" {
		return s[:len(s)-1]
	}
	return s
}

// resolveRoleBindingScope looks up a role_binding's project_id and credential_id
// from the database and populates the request scope so that bindingCoversScope
// can match the caller's binding against the correct project/credential.
func (m *DBAuthorizationMiddleware) resolveRoleBindingScope(ctx context.Context, bindingID string, scope *RequestScope) {
	g := (*m.sessionFactory).New(ctx)
	var result struct {
		ProjectID    *string
		CredentialID *string
	}
	err := g.Table("role_bindings").
		Select("project_id, credential_id").
		Where("id = ? AND deleted_at IS NULL", bindingID).
		Scan(&result).Error
	if err != nil {
		return
	}
	if result.ProjectID != nil && *result.ProjectID != "" {
		scope.ProjectID = *result.ProjectID
	}
	if result.CredentialID != nil && *result.CredentialID != "" {
		scope.CredentialID = *result.CredentialID
	}
}
