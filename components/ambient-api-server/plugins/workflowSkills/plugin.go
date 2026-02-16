package workflowSkills

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

type ServiceLocator func() WorkflowSkillService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() WorkflowSkillService {
		return NewWorkflowSkillService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewWorkflowSkillDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) WorkflowSkillService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("WorkflowSkills"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("WorkflowSkills", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("workflowSkills", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware auth.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		workflowSkillHandler := NewWorkflowSkillHandler(Service(envServices), generic.Service(envServices))

		workflowSkillsRouter := apiV1Router.PathPrefix("/workflow_skills").Subrouter()
		workflowSkillsRouter.HandleFunc("", workflowSkillHandler.List).Methods(http.MethodGet)
		workflowSkillsRouter.HandleFunc("/{id}", workflowSkillHandler.Get).Methods(http.MethodGet)
		workflowSkillsRouter.HandleFunc("", workflowSkillHandler.Create).Methods(http.MethodPost)
		workflowSkillsRouter.HandleFunc("/{id}", workflowSkillHandler.Patch).Methods(http.MethodPatch)
		workflowSkillsRouter.HandleFunc("/{id}", workflowSkillHandler.Delete).Methods(http.MethodDelete)
		workflowSkillsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		workflowSkillsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("WorkflowSkills", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		workflowSkillServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "WorkflowSkills",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {workflowSkillServices.OnUpsert},
				api.UpdateEventType: {workflowSkillServices.OnUpsert},
				api.DeleteEventType: {workflowSkillServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(WorkflowSkill{}, "workflow_skills")
	presenters.RegisterPath(&WorkflowSkill{}, "workflow_skills")
	presenters.RegisterKind(WorkflowSkill{}, "WorkflowSkill")
	presenters.RegisterKind(&WorkflowSkill{}, "WorkflowSkill")

	db.RegisterMigration(migration())
	db.RegisterMigration(constraintMigration())
}
