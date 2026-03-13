package agentMessages

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

type ServiceLocator func() AgentMessageService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() AgentMessageService {
		return NewAgentMessageService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewAgentMessageDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) AgentMessageService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("AgentMessages"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("AgentMessages", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("agentMessages", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		agentMessageHandler := NewAgentMessageHandler(Service(envServices), generic.Service(envServices))

		agentMessagesRouter := apiV1Router.PathPrefix("/agent_messages").Subrouter()
		agentMessagesRouter.HandleFunc("", agentMessageHandler.List).Methods(http.MethodGet)
		agentMessagesRouter.HandleFunc("/{id}", agentMessageHandler.Get).Methods(http.MethodGet)
		agentMessagesRouter.HandleFunc("", agentMessageHandler.Create).Methods(http.MethodPost)
		agentMessagesRouter.HandleFunc("/{id}", agentMessageHandler.Patch).Methods(http.MethodPatch)
		agentMessagesRouter.HandleFunc("/{id}", agentMessageHandler.Delete).Methods(http.MethodDelete)
		agentMessagesRouter.Use(authMiddleware.AuthenticateAccountJWT)
		agentMessagesRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("AgentMessages", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		agentMessageServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "AgentMessages",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {agentMessageServices.OnUpsert},
				api.UpdateEventType: {agentMessageServices.OnUpsert},
				api.DeleteEventType: {agentMessageServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(AgentMessage{}, "agent_messages")
	presenters.RegisterPath(&AgentMessage{}, "agent_messages")
	presenters.RegisterKind(AgentMessage{}, "AgentMessage")
	presenters.RegisterKind(&AgentMessage{}, "AgentMessage")

	db.RegisterMigration(migration())
}
