package roles

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

type ServiceLocator func() RoleService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() RoleService {
		return NewRoleService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewRoleDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) RoleService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Roles"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Roles", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("roles", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		roleHandler := NewRoleHandler(Service(envServices), generic.Service(envServices))

		rolesRouter := apiV1Router.PathPrefix("/roles").Subrouter()
		rolesRouter.HandleFunc("", roleHandler.List).Methods(http.MethodGet)
		rolesRouter.HandleFunc("/{id}", roleHandler.Get).Methods(http.MethodGet)
		rolesRouter.HandleFunc("", roleHandler.Create).Methods(http.MethodPost)
		rolesRouter.HandleFunc("/{id}", roleHandler.Patch).Methods(http.MethodPatch)
		rolesRouter.HandleFunc("/{id}", roleHandler.Delete).Methods(http.MethodDelete)
		rolesRouter.Use(authMiddleware.AuthenticateAccountJWT)
		rolesRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Roles", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		roleServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Roles",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {roleServices.OnUpsert},
				api.UpdateEventType: {roleServices.OnUpsert},
				api.DeleteEventType: {roleServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Role{}, "roles")
	presenters.RegisterPath(&Role{}, "roles")
	presenters.RegisterKind(Role{}, "Role")
	presenters.RegisterKind(&Role{}, "Role")

	db.RegisterMigration(migration())
}
