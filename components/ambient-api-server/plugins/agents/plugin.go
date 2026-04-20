package agents

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

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/inbox"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/roleBindings"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
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

func projectPromptFetcher(s *environments.Services) ProjectPromptFetcher {
	if s == nil {
		return nil
	}
	if obj := s.GetService("ProjectPromptFetcher"); obj != nil {
		locator := obj.(func() ProjectPromptFetcher)
		return locator()
	}
	return nil
}

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"code":"NOT_IMPLEMENTED","reason":"not yet implemented"}`))
}

func init() {
	registry.RegisterService("Agents", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	registry.RegisterService("AgentOwnershipChecker", func(env interface{}) interface{} {
		return func(s *environments.Services) interface{} {
			return &agentOwnershipAdapter{agentSvc: Service(s)}
		}
	})

	pkgserver.RegisterRoutes("agents", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		agentSvc := Service(envServices)
		agentHandler := NewAgentHandler(agentSvc, generic.Service(envServices))
		startHandler := NewStartHandler(agentSvc, inbox.Service(envServices), sessions.Service(envServices), sessions.MessageSvc(envServices), projectPromptFetcher(envServices))
		subHandler := NewAgentSubresourceHandler(agentSvc, sessions.Service(envServices), generic.Service(envServices), roleBindings.Service(envServices))

		projectsRouter := apiV1Router.PathPrefix("/projects").Subrouter()
		projectsRouter.HandleFunc("/{id}/agents", agentHandler.List).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents", agentHandler.Create).Methods(http.MethodPost)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}", agentHandler.Get).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}", agentHandler.Patch).Methods(http.MethodPatch)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}", agentHandler.Delete).Methods(http.MethodDelete)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/start", startHandler.Start).Methods(http.MethodPost)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/start", startHandler.StartPreview).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/sessions", subHandler.ListSessions).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/role_bindings", subHandler.ListRoleBindings).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/home", notImplemented).Methods(http.MethodGet)
		projectsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		projectsRouter.Use(authzMiddleware.AuthorizeApi)
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
	db.RegisterMigration(agentSchemaExpansionMigration())
}
