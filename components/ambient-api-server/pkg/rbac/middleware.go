package rbac

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"gorm.io/gorm"
)

type roleRow struct {
	Permissions string
}

type roleBindingRow struct {
	RoleId  string
	Scope   string
	ScopeId *string
}

type DBAuthorizationMiddleware struct {
	sessionFactory *db.SessionFactory
	enableAuthz    bool
}

func NewDBAuthorizationMiddleware(sessionFactory *db.SessionFactory, enableAuthz bool) *DBAuthorizationMiddleware {
	return &DBAuthorizationMiddleware{sessionFactory: sessionFactory, enableAuthz: enableAuthz}
}

func (m *DBAuthorizationMiddleware) AuthorizeApi(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enableAuthz {
			next.ServeHTTP(w, r)
			return
		}

		ctx := r.Context()

		payload, err := auth.GetAuthPayloadFromContext(ctx)
		if err != nil || payload == nil || payload.Username == "" {
			http.Error(w, `{"kind":"Error","reason":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}

		g := (*m.sessionFactory).New(ctx)

		allowed, err := m.isAllowed(g, payload.Username, r.Method, r.URL.Path)
		if err != nil || !allowed {
			http.Error(w, `{"kind":"Error","reason":"Forbidden"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *DBAuthorizationMiddleware) isAllowed(g *gorm.DB, username, method, path string) (bool, error) {
	action := pathToAction(method, path)
	resource := pathToResource(path)

	var bindings []roleBindingRow
	if err := g.Raw(`
		SELECT rb.role_id, rb.scope, rb.scope_id
		FROM role_bindings rb
		WHERE rb.user_id = ?
	`, username).Scan(&bindings).Error; err != nil {
		return false, err
	}

	if len(bindings) == 0 {
		return false, nil
	}

	roleIDs := make([]string, 0, len(bindings))
	for _, b := range bindings {
		roleIDs = append(roleIDs, b.RoleId)
	}

	var rows []roleRow
	if err := g.Raw(`SELECT permissions FROM roles WHERE id IN (?)`, roleIDs).Scan(&rows).Error; err != nil {
		return false, err
	}

	for _, row := range rows {
		var perms []string
		if err := json.Unmarshal([]byte(row.Permissions), &perms); err != nil {
			continue
		}
		if matchesAny(perms, resource, action) {
			return true, nil
		}
	}

	return false, nil
}

func matchesAny(perms []string, resource, action string) bool {
	for _, perm := range perms {
		if perm == "*:*" {
			return true
		}
		parts := strings.SplitN(perm, ":", 2)
		if len(parts) != 2 {
			continue
		}
		r, a := parts[0], parts[1]
		resourceMatch := r == "*" || r == resource
		actionMatch := a == "*" || a == action
		if resourceMatch && actionMatch {
			return true
		}
	}
	return false
}

func httpMethodToAction(method string) string {
	switch strings.ToUpper(method) {
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
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, p := range parts {
		if p == "v1" && i+2 < len(parts) {
			last := parts[len(parts)-1]
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
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	for i, p := range parts {
		if p == "v1" && i+1 < len(parts) {
			seg := parts[i+1]
			return strings.ReplaceAll(strings.TrimSuffix(seg, "s"), "_", "_")
		}
	}
	return "unknown"
}
