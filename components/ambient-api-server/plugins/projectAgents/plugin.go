package projectAgents

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

	"github.com/ambient-code/platform/components/ambient-api-server/plugins/agents"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/inbox"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
)

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"code":"NOT_IMPLEMENTED","reason":"not yet implemented"}`))
}

type ServiceLocator func() ProjectAgentService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() ProjectAgentService {
		return NewProjectAgentService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewProjectAgentDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) ProjectAgentService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("ProjectAgents"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("ProjectAgents", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("projectAgents", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		projectAgentHandler := NewProjectAgentHandler(Service(envServices), generic.Service(envServices))
		paIgniteHandler := NewPAIgniteHandler(
			Service(envServices),
			agents.Service(envServices),
			inbox.Service(envServices),
			sessions.Service(envServices),
			sessions.MessageSvc(envServices),
		)

		projectsRouter := apiV1Router.PathPrefix("/projects").Subrouter()
		projectsRouter.HandleFunc("/{id}/agents", projectAgentHandler.List).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents", projectAgentHandler.Create).Methods(http.MethodPost)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}", projectAgentHandler.Get).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}", projectAgentHandler.Patch).Methods(http.MethodPatch)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}", projectAgentHandler.Delete).Methods(http.MethodDelete)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}/ignite", paIgniteHandler.Ignite).Methods(http.MethodPost)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}/ignition", notImplemented).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}/sessions", notImplemented).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/home", notImplemented).Methods(http.MethodGet)
		projectsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		projectsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("ProjectAgents", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		projectAgentServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "ProjectAgents",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {projectAgentServices.OnUpsert},
				api.UpdateEventType: {projectAgentServices.OnUpsert},
				api.DeleteEventType: {projectAgentServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(ProjectAgent{}, "project_agents")
	presenters.RegisterPath(&ProjectAgent{}, "project_agents")
	presenters.RegisterKind(ProjectAgent{}, "ProjectAgent")
	presenters.RegisterKind(&ProjectAgent{}, "ProjectAgent")

	db.RegisterMigration(migration())
}
