package projectDocuments

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

type ServiceLocator func() ProjectDocumentService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() ProjectDocumentService {
		return NewProjectDocumentService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewProjectDocumentDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) ProjectDocumentService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("ProjectDocuments"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("ProjectDocuments", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("projectDocuments", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		projectDocumentHandler := NewProjectDocumentHandler(Service(envServices), generic.Service(envServices))

		projectDocumentsRouter := apiV1Router.PathPrefix("/project_documents").Subrouter()
		projectDocumentsRouter.HandleFunc("", projectDocumentHandler.List).Methods(http.MethodGet)
		projectDocumentsRouter.HandleFunc("/{id}", projectDocumentHandler.Get).Methods(http.MethodGet)
		projectDocumentsRouter.HandleFunc("", projectDocumentHandler.Create).Methods(http.MethodPost)
		projectDocumentsRouter.HandleFunc("/{id}", projectDocumentHandler.Patch).Methods(http.MethodPatch)
		projectDocumentsRouter.HandleFunc("/{id}", projectDocumentHandler.Delete).Methods(http.MethodDelete)
		projectDocumentsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		projectDocumentsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("ProjectDocuments", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		projectDocumentServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "ProjectDocuments",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {projectDocumentServices.OnUpsert},
				api.UpdateEventType: {projectDocumentServices.OnUpsert},
				api.DeleteEventType: {projectDocumentServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(ProjectDocument{}, "project_documents")
	presenters.RegisterPath(&ProjectDocument{}, "project_documents")
	presenters.RegisterKind(ProjectDocument{}, "ProjectDocument")
	presenters.RegisterKind(&ProjectDocument{}, "ProjectDocument")

	db.RegisterMigration(migration())
}
