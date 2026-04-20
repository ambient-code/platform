package inbox

import (
	"net/http"
	"sync"

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
	"google.golang.org/grpc"

	pb "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
)

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"code":"NOT_IMPLEMENTED","reason":"not yet implemented"}`))
}

var (
	globalWatchSvc     InboxWatchService
	globalWatchSvcOnce sync.Once
)

func getGlobalWatchSvc() InboxWatchService {
	globalWatchSvcOnce.Do(func() {
		globalWatchSvc = NewInboxWatchService()
	})
	return globalWatchSvc
}

type ServiceLocator func() InboxMessageService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	watchSvc := getGlobalWatchSvc()
	return func() InboxMessageService {
		return NewInboxMessageService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewInboxMessageDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
			watchSvc,
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
		var ownershipChecker AgentOwnershipChecker
		if obj := envServices.GetService("AgentOwnershipChecker"); obj != nil {
			locator := obj.(func(s *environments.Services) interface{})
			ownershipChecker = locator(envServices).(AgentOwnershipChecker)
		}
		inboxMessageHandler := NewInboxMessageHandler(Service(envServices), generic.Service(envServices), ownershipChecker)

		projectsRouter := apiV1Router.PathPrefix("/projects").Subrouter()
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/inbox", inboxMessageHandler.List).Methods(http.MethodGet)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/inbox", inboxMessageHandler.Create).Methods(http.MethodPost)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/inbox/{msg_id}", inboxMessageHandler.Patch).Methods(http.MethodPatch)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/inbox/{msg_id}", inboxMessageHandler.Delete).Methods(http.MethodDelete)
		projectsRouter.HandleFunc("/{id}/agents/{agent_id}/inbox/{msg_id}", notImplemented).Methods(http.MethodGet)
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

	pkgserver.RegisterGRPCService("inbox", func(grpcServer *grpc.Server, services pkgserver.ServicesInterface) {
		pb.RegisterInboxServiceServer(grpcServer, NewInboxGRPCHandler(getGlobalWatchSvc()))
	})

	db.RegisterMigration(migration())
}
