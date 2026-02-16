package tasks

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

type ServiceLocator func() TaskService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() TaskService {
		return NewTaskService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewTaskDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) TaskService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Tasks"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Tasks", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("tasks", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		taskHandler := NewTaskHandler(Service(envServices), generic.Service(envServices))

		tasksRouter := apiV1Router.PathPrefix("/tasks").Subrouter()
		tasksRouter.HandleFunc("", taskHandler.List).Methods(http.MethodGet)
		tasksRouter.HandleFunc("/{id}", taskHandler.Get).Methods(http.MethodGet)
		tasksRouter.HandleFunc("", taskHandler.Create).Methods(http.MethodPost)
		tasksRouter.HandleFunc("/{id}", taskHandler.Patch).Methods(http.MethodPatch)
		tasksRouter.HandleFunc("/{id}", taskHandler.Delete).Methods(http.MethodDelete)
		tasksRouter.Use(authMiddleware.AuthenticateAccountJWT)
		tasksRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Tasks", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		taskServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Tasks",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {taskServices.OnUpsert},
				api.UpdateEventType: {taskServices.OnUpsert},
				api.DeleteEventType: {taskServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Task{}, "tasks")
	presenters.RegisterPath(&Task{}, "tasks")
	presenters.RegisterKind(Task{}, "Task")
	presenters.RegisterKind(&Task{}, "Task")

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
	db.RegisterMigration(projectIdMigration())
}
