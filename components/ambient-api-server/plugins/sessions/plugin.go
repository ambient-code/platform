package sessions

import (
	"net/http"
	"sync"

	pb "github.com/ambient-code/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
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
	"google.golang.org/grpc"
)

const EventSource = "Sessions"

type ServiceLocator func() SessionService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() SessionService {
		return NewSessionService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewSessionDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) SessionService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Sessions"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

type MessageServiceLocator func() MessageService

func NewMessageServiceLocator(env *environments.Env) MessageServiceLocator {
	var (
		once    sync.Once
		svcInst MessageService
	)
	return func() MessageService {
		once.Do(func() {
			svcInst = NewMessageService(NewMessageDao(&env.Database.SessionFactory))
		})
		return svcInst
	}
}

func MessageSvc(s *environments.Services) MessageService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("SessionMessages"); obj != nil {
		locator := obj.(MessageServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Sessions", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	registry.RegisterService("SessionMessages", func(env interface{}) interface{} {
		return NewMessageServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("sessions", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		sessionHandler := NewSessionHandler(Service(envServices), generic.Service(envServices))
		msgHandler := NewMessageHandler(Service(envServices), MessageSvc(envServices))

		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}

		sessionsRouter := apiV1Router.PathPrefix("/sessions").Subrouter()
		sessionsRouter.HandleFunc("", sessionHandler.List).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}", sessionHandler.Get).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("", sessionHandler.Create).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}", sessionHandler.Patch).Methods(http.MethodPatch)
		sessionsRouter.HandleFunc("/{id}/status", sessionHandler.PatchStatus).Methods(http.MethodPatch)
		sessionsRouter.HandleFunc("/{id}/start", sessionHandler.Start).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/stop", sessionHandler.Stop).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}", sessionHandler.Delete).Methods(http.MethodDelete)
		sessionsRouter.HandleFunc("/{id}/messages", msgHandler.StreamMessages).Methods(http.MethodGet)
		sessionsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		sessionsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController(EventSource, func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		sessionServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: EventSource,
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {sessionServices.OnUpsert},
				api.UpdateEventType: {sessionServices.OnUpsert},
				api.DeleteEventType: {sessionServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Session{}, "sessions")
	presenters.RegisterPath(&Session{}, "sessions")
	presenters.RegisterKind(Session{}, "Session")
	presenters.RegisterKind(&Session{}, "Session")

	pkgserver.RegisterGRPCService("sessions", func(grpcServer *grpc.Server, services pkgserver.ServicesInterface) {
		envServices := services.(*environments.Services)
		sessionService := Service(envServices)
		genericService := generic.Service(envServices)
		msgService := MessageSvc(envServices)
		brokerFunc := func() *pkgserver.EventBroker {
			if obj := envServices.GetService("EventBroker"); obj != nil {
				return obj.(*pkgserver.EventBroker)
			}
			return nil
		}
		pb.RegisterSessionServiceServer(grpcServer, NewSessionGRPCHandler(sessionService, genericService, brokerFunc, msgService))
	})

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
	db.RegisterMigration(sessionMessagesMigration())
	db.RegisterMigration(schemaExpansionMigration())
}
