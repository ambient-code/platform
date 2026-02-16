package projectKeys

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

type ServiceLocator func() ProjectKeyService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() ProjectKeyService {
		return NewProjectKeyService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewProjectKeyDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) ProjectKeyService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("ProjectKeys"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("ProjectKeys", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("projectKeys", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		projectKeyHandler := NewProjectKeyHandler(Service(envServices), generic.Service(envServices))

		projectKeysRouter := apiV1Router.PathPrefix("/project_keys").Subrouter()
		projectKeysRouter.HandleFunc("", projectKeyHandler.List).Methods(http.MethodGet)
		projectKeysRouter.HandleFunc("/{id}", projectKeyHandler.Get).Methods(http.MethodGet)
		projectKeysRouter.HandleFunc("", projectKeyHandler.Create).Methods(http.MethodPost)
		projectKeysRouter.HandleFunc("/{id}", projectKeyHandler.Delete).Methods(http.MethodDelete)
		projectKeysRouter.Use(authMiddleware.AuthenticateAccountJWT)
		projectKeysRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("ProjectKeys", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		projectKeyServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "ProjectKeys",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {projectKeyServices.OnUpsert},
				api.DeleteEventType: {projectKeyServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(ProjectKey{}, "project_keys")
	presenters.RegisterPath(&ProjectKey{}, "project_keys")
	presenters.RegisterKind(ProjectKey{}, "ProjectKey")
	presenters.RegisterKind(&ProjectKey{}, "ProjectKey")

	db.RegisterMigration(migration())
}
