package rbac

import (
	"context"
	"net/http"

	"github.com/golang/glog"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/db"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/middleware"
)

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

		if middleware.IsServiceCaller(ctx) {
			next.ServeHTTP(w, r)
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

		allowed, evalErr := m.evaluator.Evaluate(ctx, username, resource, action, scope)
		if evalErr != nil {
			http.Error(w, `{"kind":"Error","reason":"Internal Server Error"}`, http.StatusInternalServerError)
			return
		}

		if !allowed {
			if isListEndpoint(r.Method, r.URL.Path) {
				projectIDs, isGlobal, _ := m.evaluator.AuthorizedProjectIDs(ctx, username)
				credentialIDs, credGlobal, _ := m.evaluator.AuthorizedCredentialIDs(ctx, username)

				if scope.ProjectID != "" && !isGlobal {
					found := false
					for _, pid := range projectIDs {
						if pid == scope.ProjectID {
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

		projectIDs, isGlobal, _ := m.evaluator.AuthorizedProjectIDs(ctx, username)
		credentialIDs, credGlobal, _ := m.evaluator.AuthorizedCredentialIDs(ctx, username)
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
	var emailPtr interface{} = nil
	if email != "" {
		emailPtr = email
	}
	result := g.Exec(
		`INSERT INTO users (id, username, name, email, created_at, updated_at)
		 VALUES (?, ?, ?, ?, NOW(), NOW())
		 ON CONFLICT (username) WHERE deleted_at IS NULL DO NOTHING`,
		api.NewID(), username, name, emailPtr,
	)
	if result.Error != nil {
		glog.Warningf("user auto-provision failed for %s: %v", username, result.Error)
	}
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
			return singularize(resource)
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
