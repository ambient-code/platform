package workflowTasks

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

type ServiceLocator func() WorkflowTaskService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() WorkflowTaskService {
		return NewWorkflowTaskService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewWorkflowTaskDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) WorkflowTaskService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("WorkflowTasks"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("WorkflowTasks", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("workflowTasks", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		workflowTaskHandler := NewWorkflowTaskHandler(Service(envServices), generic.Service(envServices))

		workflowTasksRouter := apiV1Router.PathPrefix("/workflow_tasks").Subrouter()
		workflowTasksRouter.HandleFunc("", workflowTaskHandler.List).Methods(http.MethodGet)
		workflowTasksRouter.HandleFunc("/{id}", workflowTaskHandler.Get).Methods(http.MethodGet)
		workflowTasksRouter.HandleFunc("", workflowTaskHandler.Create).Methods(http.MethodPost)
		workflowTasksRouter.HandleFunc("/{id}", workflowTaskHandler.Patch).Methods(http.MethodPatch)
		workflowTasksRouter.HandleFunc("/{id}", workflowTaskHandler.Delete).Methods(http.MethodDelete)
		workflowTasksRouter.Use(authMiddleware.AuthenticateAccountJWT)
		workflowTasksRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("WorkflowTasks", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		workflowTaskServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "WorkflowTasks",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {workflowTaskServices.OnUpsert},
				api.UpdateEventType: {workflowTaskServices.OnUpsert},
				api.DeleteEventType: {workflowTaskServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(WorkflowTask{}, "workflow_tasks")
	presenters.RegisterPath(&WorkflowTask{}, "workflow_tasks")
	presenters.RegisterKind(WorkflowTask{}, "WorkflowTask")
	presenters.RegisterKind(&WorkflowTask{}, "WorkflowTask")

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
}
