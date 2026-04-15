package repoEvents

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
	"github.com/openshift-online/rh-trex-ai/pkg/registry"
	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
	"github.com/openshift-online/rh-trex-ai/plugins/generic"

	pkgrbac "github.com/ambient-code/platform/components/ambient-api-server/plugins/rbac"
)

type ServiceLocator func() RepoEventService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() RepoEventService {
		return NewRepoEventService(
			NewRepoEventDao(&env.Database.SessionFactory),
		)
	}
}

func Service(s *environments.Services) RepoEventService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("RepoEvents"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("RepoEvents", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("repo_events", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		svc := Service(envServices)
		handler := NewRepoEventHandler(svc, generic.Service(envServices))

		router := apiV1Router.PathPrefix("/repo_events").Subrouter()
		router.HandleFunc("", handler.List).Methods(http.MethodGet)
		router.HandleFunc("/{id}", handler.Get).Methods(http.MethodGet)
		router.Use(authMiddleware.AuthenticateAccountJWT)
		router.Use(authzMiddleware.AuthorizeApi)
	})

	presenters.RegisterPath(RepoEvent{}, "repo_events")
	presenters.RegisterPath(&RepoEvent{}, "repo_events")
	presenters.RegisterKind(RepoEvent{}, "RepoEvent")
	presenters.RegisterKind(&RepoEvent{}, "RepoEvent")

	db.RegisterMigration(migration())
}
