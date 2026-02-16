package permissions

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

type ServiceLocator func() PermissionService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() PermissionService {
		return NewPermissionService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewPermissionDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) PermissionService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Permissions"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Permissions", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("permissions", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		permissionHandler := NewPermissionHandler(Service(envServices), generic.Service(envServices))

		permissionsRouter := apiV1Router.PathPrefix("/permissions").Subrouter()
		permissionsRouter.HandleFunc("", permissionHandler.List).Methods(http.MethodGet)
		permissionsRouter.HandleFunc("/{id}", permissionHandler.Get).Methods(http.MethodGet)
		permissionsRouter.HandleFunc("", permissionHandler.Create).Methods(http.MethodPost)
		permissionsRouter.HandleFunc("/{id}", permissionHandler.Patch).Methods(http.MethodPatch)
		permissionsRouter.HandleFunc("/{id}", permissionHandler.Delete).Methods(http.MethodDelete)
		permissionsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		permissionsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Permissions", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		permissionServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Permissions",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {permissionServices.OnUpsert},
				api.UpdateEventType: {permissionServices.OnUpsert},
				api.DeleteEventType: {permissionServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Permission{}, "permissions")
	presenters.RegisterPath(&Permission{}, "permissions")
	presenters.RegisterKind(Permission{}, "Permission")
	presenters.RegisterKind(&Permission{}, "Permission")

	db.RegisterMigration(migration())
}
