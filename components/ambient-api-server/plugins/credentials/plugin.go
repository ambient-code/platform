package credentials

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

type ServiceLocator func() CredentialService

func NewServiceLocator(env *environments.Env) ServiceLocator {
	return func() CredentialService {
		return NewCredentialService(
			db.NewAdvisoryLockFactory(env.Database.SessionFactory),
			NewCredentialDao(&env.Database.SessionFactory),
			events.Service(&env.Services),
		)
	}
}

func Service(s *environments.Services) CredentialService {
	if s == nil {
		return nil
	}
	if obj := s.GetService("Credentials"); obj != nil {
		locator := obj.(ServiceLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("Credentials", func(env interface{}) interface{} {
		return NewServiceLocator(env.(*environments.Env))
	})

	pkgserver.RegisterRoutes("credentials", func(apiV1Router *mux.Router, services pkgserver.ServicesInterface, authMiddleware environments.JWTMiddleware, authzMiddleware auth.AuthorizationMiddleware) {
		envServices := services.(*environments.Services)
		if dbAuthz := pkgrbac.Middleware(envServices); dbAuthz != nil {
			authzMiddleware = dbAuthz
		}
		credentialHandler := NewCredentialHandler(Service(envServices), generic.Service(envServices))

		credentialsRouter := apiV1Router.PathPrefix("/projects").Subrouter()
		credentialsRouter.HandleFunc("/{id}/credentials", credentialHandler.List).Methods(http.MethodGet)
		credentialsRouter.HandleFunc("/{id}/credentials", credentialHandler.Create).Methods(http.MethodPost)
		credentialsRouter.HandleFunc("/{id}/credentials/{cred_id}", credentialHandler.Get).Methods(http.MethodGet)
		credentialsRouter.HandleFunc("/{id}/credentials/{cred_id}", credentialHandler.Patch).Methods(http.MethodPatch)
		credentialsRouter.HandleFunc("/{id}/credentials/{cred_id}", credentialHandler.Delete).Methods(http.MethodDelete)
		credentialsRouter.HandleFunc("/{id}/credentials/{cred_id}/token", credentialHandler.GetToken).Methods(http.MethodGet)
		credentialsRouter.Use(authMiddleware.AuthenticateAccountJWT)
		credentialsRouter.Use(authzMiddleware.AuthorizeApi)
	})

	pkgserver.RegisterController("Credentials", func(manager *controllers.KindControllerManager, services pkgserver.ServicesInterface) {
		credentialServices := Service(services.(*environments.Services))

		manager.Add(&controllers.ControllerConfig{
			Source: "Credentials",
			Handlers: map[api.EventType][]controllers.ControllerHandlerFunc{
				api.CreateEventType: {credentialServices.OnUpsert},
				api.UpdateEventType: {credentialServices.OnUpsert},
				api.DeleteEventType: {credentialServices.OnDelete},
			},
		})
	})

	presenters.RegisterPath(Credential{}, "credentials")
	presenters.RegisterPath(&Credential{}, "credentials")
	presenters.RegisterKind(Credential{}, "Credential")
	presenters.RegisterKind(&Credential{}, "Credential")

	db.RegisterMigration(migration())
	db.RegisterMigration(rolesMigration())
	db.RegisterMigration(addProjectIDMigration())
	db.RegisterMigration(removeCredentialReaderRoleMigration())
}
