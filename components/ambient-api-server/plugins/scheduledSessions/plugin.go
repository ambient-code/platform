package scheduledSessions

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
	"github.com/openshift-online/rh-trex-ai/pkg/registry"
	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"

	pkgrbac "github.com/ambient-code/platform/components/ambient-api-server/plugins/rbac"
)

func init() {
	pkgserver.RegisterRoutes("scheduledSessions", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)

		var svc ScheduledSessionService
		if obj := envServices.GetService("ScheduledSessions"); obj != nil {
			svc = obj.(func() ScheduledSessionService)()
		} else {
			svc = NewInMemoryService()
		}

		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}

		h := NewScheduledSessionHandler(svc)

		projectRouter := apiV1Router.PathPrefix("/projects/{project_id}").Subrouter()
		schedRouter := projectRouter.PathPrefix("/scheduled-sessions").Subrouter()
		schedRouter.HandleFunc("", h.List).Methods(http.MethodGet)
		schedRouter.HandleFunc("", h.Create).Methods(http.MethodPost)
		schedRouter.HandleFunc("/{id}", h.Get).Methods(http.MethodGet)
		schedRouter.HandleFunc("/{id}", h.Patch).Methods(http.MethodPatch)
		schedRouter.HandleFunc("/{id}", h.Delete).Methods(http.MethodDelete)
		schedRouter.HandleFunc("/{id}/suspend", h.Suspend).Methods(http.MethodPost)
		schedRouter.HandleFunc("/{id}/resume", h.Resume).Methods(http.MethodPost)
		schedRouter.HandleFunc("/{id}/trigger", h.Trigger).Methods(http.MethodPost)
		schedRouter.HandleFunc("/{id}/runs", h.Runs).Methods(http.MethodGet)
		schedRouter.Use(authMiddleware.AuthenticateAccountJWT)
		schedRouter.Use(authzMiddleware.AuthorizeApi)
	})

	// SQL-backed service registered for production.
	// In unit_testing / dev the in-memory fallback in RegisterRoutes is used.
	registry.RegisterService("ScheduledSessionsSQL", func(env interface{}) interface{} {
		e := env.(*environments.Env)
		return func() ScheduledSessionService {
			return NewScheduledSessionService(
				NewScheduledSessionDao(&e.Database.SessionFactory),
			)
		}
	})

	db.RegisterMigration(migration())
	db.RegisterMigration(indexMigration())
}
