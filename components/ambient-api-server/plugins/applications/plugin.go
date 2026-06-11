package applications

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

type ServiceLocator func() ApplicationService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() ApplicationService {
		return NewApplicationService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewApplicationDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) ApplicationService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Applications"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Applications", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("applications", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		applicationHandler := NewApplicationHandler(Service(envServices), generic.Service(envServices))

		applicationsRouter := apiV1Router.PathPrefix("/applications").Subrouter()
		applicationsRouter.HandleFunc("", applicationHandler.List).Methods(http.MethodGet)
		applicationsRouter.HandleFunc("", applicationHandler.Create).Methods(http.MethodPost)
		applicationsRouter.HandleFunc("/{id}", applicationHandler.Get).Methods(http.MethodGet)
		applicationsRouter.HandleFunc("/{id}", applicationHandler.Patch).Methods(http.MethodPatch)
		applicationsRouter.HandleFunc("/{id}", applicationHandler.Delete).Methods(http.MethodDelete)
		applicationsRouter.HandleFunc("/{id}/sync", applicationHandler.Sync).Methods(http.MethodPost)
		applicationsRouter.HandleFunc("/{id}/refresh", applicationHandler.Refresh).Methods(http.MethodPost)
		applicationsRouter.HandleFunc("/{id}/status", applicationHandler.Status).Methods(http.MethodGet)
		applicationsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		applicationsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Applications", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		applicationServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Applications",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {applicationServices.OnUpsert},
				api.UpdateEventType: {applicationServices.OnUpsert},
				api.DeleteEventType: {applicationServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Application{}, "applications")
	presenters.RegisterPath(&Application{}, "applications")
	presenters.RegisterKind(Application{}, "Application")
	presenters.RegisterKind(&Application{}, "Application")

	db.RegisterMigration(migration())
	db.RegisterMigration(gitopsRolesMigration())
}
