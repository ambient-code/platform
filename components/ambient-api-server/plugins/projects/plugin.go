package projects

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

type ServiceLocator func() ProjectService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() ProjectService {
		return NewProjectService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewProjectDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) ProjectService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Projects"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Projects", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("projects", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		projectHandler := NewProjectHandler(Service(envServices), generic.Service(envServices))

		projectsRouter := apiV1Router.PathPrefix("/projects").Subrouter()
		projectsRouter.HandleFunc("", projectHandler.List).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}", projectHandler.Get).Methods(http.MethodGet)
		projectsRouter.HandleFunc("", projectHandler.Create).Methods(http.MethodPost)
		projectsRouter.HandleFunc("/{id}", projectHandler.Patch).Methods(http.MethodPatch)
		projectsRouter.HandleFunc("/{id}", projectHandler.Delete).Methods(http.MethodDelete)
		projectsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		projectsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Projects", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		projectServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Projects",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {projectServices.OnUpsert},
				api.UpdateEventType: {projectServices.OnUpsert},
				api.DeleteEventType: {projectServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Project{}, "projects")
	presenters.RegisterPath(&Project{}, "projects")
	presenters.RegisterKind(Project{}, "Project")
	presenters.RegisterKind(&Project{}, "Project")

	db.RegisterMigration(migration())
}
