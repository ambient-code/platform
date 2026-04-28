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
		sessionSvc := Service(envServices)
		sessionHandler := NewSessionHandler(sessionSvc, MessageSvc(envServices), generic.Service(envServices))
		msgHandler := NewMessageHandler(sessionSvc, MessageSvc(envServices))

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
		sessionsRouter.HandleFunc("/{id}/events", sessionHandler.StreamRunnerEvents).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/messages", msgHandler.GetMessages).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/messages", msgHandler.PushMessage).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/clone", sessionHandler.Clone).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/repos", sessionHandler.AddRepo).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/repos/{repoName}", sessionHandler.RemoveRepo).Methods(http.MethodDelete)
		sessionsRouter.HandleFunc("/{id}/workflow", sessionHandler.SetWorkflow).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/model", sessionHandler.SetModel).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/agui/events", sessionHandler.AGUIEvents).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/agui/run", sessionHandler.AGUIRun).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/agui/interrupt", sessionHandler.AGUIInterrupt).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/agui/feedback", sessionHandler.AGUIFeedback).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/agui/tasks", sessionHandler.AGUITasks).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/agui/tasks/{taskId}/stop", sessionHandler.AGUITaskStop).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/agui/tasks/{taskId}/output", sessionHandler.AGUITaskOutput).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/agui/capabilities", sessionHandler.AGUICapabilities).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/mcp/status", sessionHandler.MCPStatus).Methods(http.MethodGet)
		// Workspace file proxy
		sessionsRouter.HandleFunc("/{id}/workspace", sessionHandler.WorkspaceList).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/workspace/{path:.*}", sessionHandler.WorkspaceFile).Methods(http.MethodGet, http.MethodPut, http.MethodDelete)
		// Pre-upload file proxy
		sessionsRouter.HandleFunc("/{id}/files", sessionHandler.FilesList).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/files/{path:.*}", sessionHandler.FilesFile).Methods(http.MethodPut, http.MethodDelete)
		// Git proxy
		sessionsRouter.HandleFunc("/{id}/git/status", sessionHandler.GitStatus).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/git/configure-remote", sessionHandler.GitConfigureRemote).Methods(http.MethodPost)
		sessionsRouter.HandleFunc("/{id}/git/branches", sessionHandler.GitBranches).Methods(http.MethodGet)
		// Repos status proxy
		sessionsRouter.HandleFunc("/{id}/repos/status", sessionHandler.ReposStatus).Methods(http.MethodGet)
		// Pod events (K8s stub)
		sessionsRouter.HandleFunc("/{id}/pod-events", sessionHandler.PodEvents).Methods(http.MethodGet)
		// Operational sub-resources
		sessionsRouter.HandleFunc("/{id}/displayname", sessionHandler.PatchDisplayName).Methods(http.MethodPatch)
		sessionsRouter.HandleFunc("/{id}/workflow/metadata", sessionHandler.WorkflowMetadata).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/oauth/{provider}/url", sessionHandler.OAuthProviderURL).Methods(http.MethodGet)
		sessionsRouter.HandleFunc("/{id}/export", sessionHandler.ExportSession).Methods(http.MethodGet)
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
	db.RegisterMigration(agentIDMigration())
}
