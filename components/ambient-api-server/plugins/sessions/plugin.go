package sessions

import (
	"net/http"

	pb "github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc/ambient/v1"
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

func init() {
	registry.RegisterService("Sessions", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("sessions", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		sessionHandler := NewSessionHandler(Service(envServices), generic.Service(envServices))

		sessionsRouter := apiV1Router.PathPrefix("/sessions").Subrouter()
		sessionsRouter.HandleFunc("", sessionHandler.List).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}", sessionHandler.Get).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("", sessionHandler.Create).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}", sessionHandler.Patch).Methods(http.MethodPatch)
		sessionsRouter.HandleFunc("/{id}/status", sessionHandler.PatchStatus).Methods(http.MethodPatch)
		sessionsRouter.HandleFunc("/{id}/start", sessionHandler.Start).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/stop", sessionHandler.Stop).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}", sessionHandler.Delete).Methods(http.MethodDelete)
		sessionsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		sessionsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Sessions", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		sessionServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Sessions",
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
		brokerFunc := func() *pkgserver.EventBroker {
			if obj := envServices.GetService("EventBroker"); obj != nil {
				return obj.(*pkgserver.EventBroker)
			}
			return nil
		}
		pb.RegisterSessionServiceServer(grpcServer, NewSessionGRPCHandler(sessionService, genericService, brokerFunc))
	})

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
	db.RegisterMigration(schemaExpansionMigration())
}
