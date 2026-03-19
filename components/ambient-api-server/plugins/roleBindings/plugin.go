package roleBindings

import (
	"net/http"

	pkgrbac "github.com/ambient-code/platform/components/ambient-api-server/plugins/rbac"
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

type ServiceLocator func() RoleBindingService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() RoleBindingService {
		return NewRoleBindingService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewRoleBindingDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) RoleBindingService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("RoleBindings"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("RoleBindings", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("roleBindings", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		roleBindingHandler := NewRoleBindingHandler(Service(envServices), generic.Service(envServices))

		roleBindingsRouter := apiV1Router.PathPrefix("/role_bindings").Subrouter()
		roleBindingsRouter.HandleFunc("", roleBindingHandler.List).Methods(http.MethodGet)
		roleBindingsRouter.HandleFunc("/{id}", roleBindingHandler.Get).Methods(http.MethodGet)
		roleBindingsRouter.HandleFunc("", roleBindingHandler.Create).Methods(http.MethodPost)
		roleBindingsRouter.HandleFunc("/{id}", roleBindingHandler.Patch).Methods(http.MethodPatch)
		roleBindingsRouter.HandleFunc("/{id}", roleBindingHandler.Delete).Methods(http.MethodDelete)
		roleBindingsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		roleBindingsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("RoleBindings", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		roleBindingServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "RoleBindings",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {roleBindingServices.OnUpsert},
				api.UpdateEventType: {roleBindingServices.OnUpsert},
				api.DeleteEventType: {roleBindingServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(RoleBinding{}, "role_bindings")
	presenters.RegisterPath(&RoleBinding{}, "role_bindings")
	presenters.RegisterKind(RoleBinding{}, "RoleBinding")
	presenters.RegisterKind(&RoleBinding{}, "RoleBinding")

	db.RegisterMigration(migration())
}
