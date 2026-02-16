package users

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

type ServiceLocator func() UserService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() UserService {
		return NewUserService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewUserDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) UserService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Users"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Users", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("users", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		userHandler := NewUserHandler(Service(envServices), generic.Service(envServices))

		usersRouter := apiV1Router.PathPrefix("/users").Subrouter()
		usersRouter.HandleFunc("", userHandler.List).Methods(http.MethodGet)
		usersRouter.HandleFunc("/{id}", userHandler.Get).Methods(http.MethodGet)
		usersRouter.HandleFunc("", userHandler.Create).Methods(http.MethodPost)
		usersRouter.HandleFunc("/{id}", userHandler.Patch).Methods(http.MethodPatch)
		usersRouter.HandleFunc("/{id}", userHandler.Delete).Methods(http.MethodDelete)
		usersRouter.Use(authMiddleware.AuthenticateAccountJWT)
		usersRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Users", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		userServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Users",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {userServices.OnUpsert},
				api.UpdateEventType: {userServices.OnUpsert},
				api.DeleteEventType: {userServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(User{}, "users")
	presenters.RegisterPath(&User{}, "users")
	presenters.RegisterKind(User{}, "User")
	presenters.RegisterKind(&User{}, "User")

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
	db.RegisterMigration(groupsMigration())
}
