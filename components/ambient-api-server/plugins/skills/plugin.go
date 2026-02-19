package skills

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

type ServiceLocator func() SkillService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() SkillService {
		return NewSkillService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewSkillDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) SkillService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Skills"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Skills", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("skills", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		skillHandler := NewSkillHandler(Service(envServices), generic.Service(envServices))

		skillsRouter := apiV1Router.PathPrefix("/skills").Subrouter()
		skillsRouter.HandleFunc("", skillHandler.List).Methods(http.MethodGet)
		skillsRouter.HandleFunc("/{id}", skillHandler.Get).Methods(http.MethodGet)
		skillsRouter.HandleFunc("", skillHandler.Create).Methods(http.MethodPost)
		skillsRouter.HandleFunc("/{id}", skillHandler.Patch).Methods(http.MethodPatch)
		skillsRouter.HandleFunc("/{id}", skillHandler.Delete).Methods(http.MethodDelete)
		skillsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		skillsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Skills", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		skillServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Skills",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {skillServices.OnUpsert},
				api.UpdateEventType: {skillServices.OnUpsert},
				api.DeleteEventType: {skillServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Skill{}, "skills")
	presenters.RegisterPath(&Skill{}, "skills")
	presenters.RegisterKind(Skill{}, "Skill")
	presenters.RegisterKind(&Skill{}, "Skill")

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
	db.RegisterMigration(projectIdMigration())
}
