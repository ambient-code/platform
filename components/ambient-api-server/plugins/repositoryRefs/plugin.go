package repositoryRefs

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
)

type ServiceLocator func() RepositoryRefService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() RepositoryRefService {
		return NewRepositoryRefService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewRepositoryRefDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) RepositoryRefService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("RepositoryRefs"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("RepositoryRefs", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("repositoryRefs", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		repositoryRefHandler := NewRepositoryRefHandler(Service(envServices), generic.Service(envServices))

		repositoryRefsRouter := apiV1Router.PathPrefix("/repository_refs").Subrouter()
		repositoryRefsRouter.HandleFunc("", repositoryRefHandler.List).Methods(http.MethodGet)
		repositoryRefsRouter.HandleFunc("/{id}", repositoryRefHandler.Get).Methods(http.MethodGet)
		repositoryRefsRouter.HandleFunc("", repositoryRefHandler.Create).Methods(http.MethodPost)
		repositoryRefsRouter.HandleFunc("/{id}", repositoryRefHandler.Patch).Methods(http.MethodPatch)
		repositoryRefsRouter.HandleFunc("/{id}", repositoryRefHandler.Delete).Methods(http.MethodDelete)
		repositoryRefsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		repositoryRefsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("RepositoryRefs", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		repositoryRefServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "RepositoryRefs",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {repositoryRefServices.OnUpsert},
				api.UpdateEventType: {repositoryRefServices.OnUpsert},
				api.DeleteEventType: {repositoryRefServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(RepositoryRef{}, "repository_refs")
	presenters.RegisterPath(&RepositoryRef{}, "repository_refs")
	presenters.RegisterKind(RepositoryRef{}, "RepositoryRef")
	presenters.RegisterKind(&RepositoryRef{}, "RepositoryRef")

	db.RegisterMigration(migration())
}
