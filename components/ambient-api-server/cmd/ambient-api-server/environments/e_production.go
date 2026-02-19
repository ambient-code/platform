package environments

import (
	"github.com/openshift-online/rh-trex-ai/pkg/config"
	"github.com/openshift-online/rh-trex-ai/pkg/db/db_session"
	pkgenv "github.com/openshift-online/rh-trex-ai/pkg/environments"
)

var _ pkgenv.EnvironmentImpl = &ProductionEnvImpl{}

type ProductionEnvImpl struct {
	Env *pkgenv.Env
}

func (e *ProductionEnvImpl) OverrideDatabase(c *pkgenv.Database) error {
	c.SessionFactory = db_session.NewProdFactory(e.Env.Config.Database)
	return nil
}

func (e *ProductionEnvImpl) OverrideConfig(c *config.ApplicationConfig) error {
	c.Server.CORSAllowedHeaders = []string{"X-Ambient-Project"}
	return nil
}

func (e *ProductionEnvImpl) OverrideServices(s *pkgenv.Services) error {
	return nil
}

func (e *ProductionEnvImpl) OverrideHandlers(h *pkgenv.Handlers) error {
	return nil
}

func (e *ProductionEnvImpl) OverrideClients(c *pkgenv.Clients) error {
	return nil
}

func (e *ProductionEnvImpl) Flags() map[string]string {
	return map[string]string{
		"v":             "1",
		"debug":         "false",
		"enable-mock":   "false",
		"enable-sentry": "true",
	}
}
