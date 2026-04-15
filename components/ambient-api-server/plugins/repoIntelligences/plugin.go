package repoIntelligences

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

type ServiceLocator func() RepoIntelligenceService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() RepoIntelligenceService {
		return NewRepoIntelligenceService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewRepoIntelligenceDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
			repoEvents.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) RepoIntelligenceService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("RepoIntelligences"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("RepoIntelligences", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("repo_intelligences", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		svc := Service(envServices)
		genericSvc := generic.Service(envServices)
		handler := NewRepoIntelligenceHandler(svc, genericSvc, genericSvc)

		router := apiV1Router.PathPrefix("/repo_intelligences").Subrouter()
		router.HandleFunc("", handler.List).Methods(http.MethodGet)
		router.HandleFunc("", handler.Create).Methods(http.MethodPost)
		router.HandleFunc("/lookup", handler.Lookup).Methods(http.MethodGet)
		router.HandleFunc("/lookup", handler.DeleteByLookup).Methods(http.MethodDelete)
		router.HandleFunc("/context", handler.Context).Methods(http.MethodGet)
		router.HandleFunc("/{id}", handler.Get).Methods(http.MethodGet)
		router.HandleFunc("/{id}", handler.Patch).Methods(http.MethodPatch)
		router.HandleFunc("/{id}", handler.Delete).Methods(http.MethodDelete)
		router.HandleFunc("/{id}/findings", handler.ListFindings).Methods(http.MethodGet)
		router.Use(authMiddleware.AuthenticateAccountJWT)
		router.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("RepoIntelligences", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		svc := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "RepoIntelligences",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {svc.OnUpsert},
				api.UpdateEventType: {svc.OnUpsert},
				api.DeleteEventType: {svc.OnDelete},
			},
		})
	})

	presenters.RegisterPath(RepoIntelligence{}, "repo_intelligences")
	presenters.RegisterPath(&RepoIntelligence{}, "repo_intelligences")
	presenters.RegisterKind(RepoIntelligence{}, "RepoIntelligence")
	presenters.RegisterKind(&RepoIntelligence{}, "RepoIntelligence")

	db.RegisterMigration(migration())
	db.RegisterMigration(migrationFixUniqueIndex())
}
