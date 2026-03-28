package inbox

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

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"code":"NOT_IMPLEMENTED","reason":"not yet implemented"}`))
}

type ServiceLocator func() InboxMessageService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() InboxMessageService {
		return NewInboxMessageService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewInboxMessageDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) InboxMessageService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("InboxMessages"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("InboxMessages", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("inbox", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		inboxMessageHandler := NewInboxMessageHandler(Service(envServices), generic.Service(envServices))

		projectsRouter := apiV1Router.PathPrefix("/projects").Subrouter()
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}/inbox", inboxMessageHandler.List).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}/inbox", inboxMessageHandler.Create).Methods(http.MethodPost)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}/inbox/{msg_id}", inboxMessageHandler.Patch).Methods(http.MethodPatch)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}/inbox/{msg_id}", inboxMessageHandler.Delete).Methods(http.MethodDelete)
		projectsRouter.HandleFunc("/{id}/agents/{pa_id}/inbox/{msg_id}", notImplemented).Methods(http.MethodGet)
		projectsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		projectsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("InboxMessages", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		inboxMessageServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "InboxMessages",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {inboxMessageServices.OnUpsert},
				api.UpdateEventType: {inboxMessageServices.OnUpsert},
				api.DeleteEventType: {inboxMessageServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(InboxMessage{}, "inbox_messages")
	presenters.RegisterPath(&InboxMessage{}, "inbox_messages")
	presenters.RegisterKind(InboxMessage{}, "InboxMessage")
	presenters.RegisterKind(&InboxMessage{}, "InboxMessage")

	db.RegisterMigration(migration())
}
