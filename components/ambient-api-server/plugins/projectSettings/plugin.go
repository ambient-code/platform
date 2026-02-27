package projectSettings

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

const EventSource = "ProjectSettings"

type ServiceLocator func() ProjectSettingsService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() ProjectSettingsService {
		return NewProjectSettingsService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewProjectSettingsDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) ProjectSettingsService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("ProjectSettings"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("ProjectSettings", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("project_settings", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		handler := NewProjectSettingsHandler(Service(envServices), generic.Service(envServices))

		router := apiV1Router.PathPrefix("/project_settings").Subrouter()
		router.HandleFunc("", handler.List).Methods(http.MethodGet)
		router.HandleFunc("/{id}", handler.Get).Methods(http.MethodGet)
		router.HandleFunc("", handler.Create).Methods(http.MethodPost)
		router.HandleFunc("/{id}", handler.Patch).Methods(http.MethodPatch)
		router.HandleFunc("/{id}", handler.Delete).Methods(http.MethodDelete)
		router.Use(authMiddleware.AuthenticateAccountJWT)
		router.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController(EventSource, func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		psServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: EventSource,
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {psServices.OnUpsert},
				api.UpdateEventType: {psServices.OnUpsert},
				api.DeleteEventType: {psServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(ProjectSettings{}, "project_settings")
	presenters.RegisterPath(&ProjectSettings{}, "project_settings")
	presenters.RegisterKind(ProjectSettings{}, "ProjectSettings")
	presenters.RegisterKind(&ProjectSettings{}, "ProjectSettings")

	pkgserver.RegisterGRPCService("project_settings", func(grpcServer *grpc.Server, services pkgserver.ServicesInterface) {
		envServices := services.(*environments.Services)
		psService := Service(envServices)
		genericService := generic.Service(envServices)
		brokerFunc := func() *pkgserver.EventBroker {
			if obj := envServices.GetService("EventBroker"); obj != nil {
				return obj.(*pkgserver.EventBroker)
			}
			return nil
		}
		pb.RegisterProjectSettingsServiceServer(grpcServer, NewProjectSettingsGRPCHandler(psService, genericService, brokerFunc))
	})

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
}
