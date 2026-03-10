package rbac

import (
	"github.com/openshift-online/rh-trex-ai/pkg/auth"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
	"github.com/openshift-online/rh-trex-ai/pkg/registry"

	pkgrbac "github.com/ambient-code/platform/components/ambient-api-server/pkg/rbac"
)

type MiddlewareLocator func() auth.AuthorizationMiddleware

func Middleware(s *environments.Services) auth.AuthorizationMiddleware {
	if s == nil {
		return nil
	}
	if obj := s.GetService("RBACMiddleware"); obj != nil {
		locator := obj.(MiddlewareLocator)
		return locator()
	}
	return nil
}

func init() {
	registry.RegisterService("RBACMiddleware", func(env interface{}) interface{} {
		e := env.(*environments.Env)
		mw := pkgrbac.NewDBAuthorizationMiddleware(&e.Database.SessionFactory, e.Config.Auth.EnableAuthz)
		return MiddlewareLocator(func() auth.AuthorizationMiddleware {
			return mw
		})
	})
}
