package workflows

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

type ServiceLocator func() WorkflowService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() WorkflowService {
		return NewWorkflowService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewWorkflowDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) WorkflowService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Workflows"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Workflows", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("workflows", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		workflowHandler := NewWorkflowHandler(Service(envServices), generic.Service(envServices))

		workflowsRouter := apiV1Router.PathPrefix("/workflows").Subrouter()
		workflowsRouter.HandleFunc("", workflowHandler.List).Methods(http.MethodGet)
		workflowsRouter.HandleFunc("/{id}", workflowHandler.Get).Methods(http.MethodGet)
		workflowsRouter.HandleFunc("", workflowHandler.Create).Methods(http.MethodPost)
		workflowsRouter.HandleFunc("/{id}", workflowHandler.Patch).Methods(http.MethodPatch)
		workflowsRouter.HandleFunc("/{id}", workflowHandler.Delete).Methods(http.MethodDelete)
		workflowsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		workflowsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Workflows", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		workflowServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Workflows",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {workflowServices.OnUpsert},
				api.UpdateEventType: {workflowServices.OnUpsert},
				api.DeleteEventType: {workflowServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Workflow{}, "workflows")
	presenters.RegisterPath(&Workflow{}, "workflows")
	presenters.RegisterKind(Workflow{}, "Workflow")
	presenters.RegisterKind(&Workflow{}, "Workflow")

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
	db.RegisterMigration(projectIdMigration())
}
