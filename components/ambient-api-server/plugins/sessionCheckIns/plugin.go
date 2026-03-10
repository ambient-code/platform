package sessionCheckIns

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

type ServiceLocator func() SessionCheckInService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() SessionCheckInService {
		return NewSessionCheckInService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewSessionCheckInDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) SessionCheckInService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("SessionCheckIns"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("SessionCheckIns", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("sessionCheckIns", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		sessionCheckInHandler := NewSessionCheckInHandler(Service(envServices), generic.Service(envServices))

		sessionCheckInsRouter := apiV1Router.PathPrefix("/session_check_ins").Subrouter()
		sessionCheckInsRouter.HandleFunc("", sessionCheckInHandler.List).Methods(http.MethodGet)
		sessionCheckInsRouter.HandleFunc("/{id}", sessionCheckInHandler.Get).Methods(http.MethodGet)
		sessionCheckInsRouter.HandleFunc("", sessionCheckInHandler.Create).Methods(http.MethodPost)
		sessionCheckInsRouter.HandleFunc("/{id}", sessionCheckInHandler.Patch).Methods(http.MethodPatch)
		sessionCheckInsRouter.HandleFunc("/{id}", sessionCheckInHandler.Delete).Methods(http.MethodDelete)
		sessionCheckInsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		sessionCheckInsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("SessionCheckIns", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		sessionCheckInServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "SessionCheckIns",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {sessionCheckInServices.OnUpsert},
				api.UpdateEventType: {sessionCheckInServices.OnUpsert},
				api.DeleteEventType: {sessionCheckInServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(SessionCheckIn{}, "session_check_ins")
	presenters.RegisterPath(&SessionCheckIn{}, "session_check_ins")
	presenters.RegisterKind(SessionCheckIn{}, "SessionCheckIn")
	presenters.RegisterKind(&SessionCheckIn{}, "SessionCheckIn")

	db.RegisterMigration(migration())
}
