package agents

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

type ServiceLocator func() AgentService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() AgentService {
		return NewAgentService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewAgentDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) AgentService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Agents"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Agents", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("agents", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		agentHandler := NewAgentHandler(Service(envServices), generic.Service(envServices))

		agentsRouter := apiV1Router.PathPrefix("/agents").Subrouter()
		agentsRouter.HandleFunc("", agentHandler.List).Methods(http.MethodGet)
		agentsRouter.HandleFunc("/{id}", agentHandler.Get).Methods(http.MethodGet)
		agentsRouter.HandleFunc("", agentHandler.Create).Methods(http.MethodPost)
		agentsRouter.HandleFunc("/{id}", agentHandler.Patch).Methods(http.MethodPatch)
		agentsRouter.HandleFunc("/{id}", agentHandler.Delete).Methods(http.MethodDelete)
		agentsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		agentsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Agents", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		agentServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Agents",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {agentServices.OnUpsert},
				api.UpdateEventType: {agentServices.OnUpsert},
				api.DeleteEventType: {agentServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Agent{}, "agents")
	presenters.RegisterPath(&Agent{}, "agents")
	presenters.RegisterKind(Agent{}, "Agent")
	presenters.RegisterKind(&Agent{}, "Agent")

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
	db.RegisterMigration(projectIdMigration())
}
