package repoFindings

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/api/presenters"
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/controllers"
	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
	"github.com/openshift-online/rh-trex-ai/pkg/registry"
	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
	"github.com/openshift-online/rh-trex-ai/plugins/events"
	"github.com/openshift-online/rh-trex-ai/plugins/generic"

	pkgrbac "github.com/ambient-code/platform/components/ambient-api-server/plugins/rbac"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/repoEvents"
)

type ServiceLocator func() RepoFindingService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() RepoFindingService {
		return NewRepoFindingService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewRepoFindingDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
			repoEvents.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) RepoFindingService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("RepoFindings"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("RepoFindings", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("repo_findings", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		svc := Service(envServices)
		handler := NewRepoFindingHandler(svc, generic.Service(envServices))

		router := apiV1Router.PathPrefix("/repo_findings").Subrouter()
		router.HandleFunc("", handler.List).Methods(http.MethodGet)
		router.HandleFunc("", handler.Create).Methods(http.MethodPost)
		router.HandleFunc("/{id}", handler.Get).Methods(http.MethodGet)
		router.HandleFunc("/{id}", handler.Patch).Methods(http.MethodPatch)
		router.HandleFunc("/{id}", handler.Delete).Methods(http.MethodDelete)
		router.Use(authMiddleware.AuthenticateAccountJWT)
		router.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("RepoFindings", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		svc := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "RepoFindings",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {svc.OnUpsert},
				api.UpdateEventType: {svc.OnUpsert},
				api.DeleteEventType: {svc.OnDelete},
			},
		})
	})

	presenters.RegisterPath(RepoFinding{}, "repo_findings")
	presenters.RegisterPath(&RepoFinding{}, "repo_findings")
	presenters.RegisterKind(RepoFinding{}, "RepoFinding")
	presenters.RegisterKind(&RepoFinding{}, "RepoFinding")

	db.RegisterMigration(migration())
}
