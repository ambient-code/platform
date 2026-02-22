package environments

import (
	"github.com/openshift-online/rh-trex-ai/pkg/config"
	"github.com/openshift-online/rh-trex-ai/pkg/db/db_session"
	pkgenv "github.com/openshift-online/rh-trex-ai/pkg/environments"
)

type DevEnvImpl struct {
	Env *pkgenv.Env
}

var _ pkgenv.EnvironmentImpl = &DevEnvImpl{}

func (e *DevEnvImpl) OverrideDatabase(c *pkgenv.Database) error {
	c.SessionFactory = db_session.NewProdFactory(e.Env.Config.Database)
	return nil
}

func (e *DevEnvImpl) OverrideConfig(c *config.ApplicationConfig) error {
	c.Server.EnableJWT = false
	c.Server.EnableHTTPS = false
	c.Server.CORSAllowedHeaders = []string{"X-Ambient-Project"}
	return nil
}

func (e *DevEnvImpl) OverrideServices(s *pkgenv.Services) error {
	return nil
}

func (e *DevEnvImpl) OverrideHandlers(h *pkgenv.Handlers) error {
	return nil
}

func (e *DevEnvImpl) OverrideClients(c *pkgenv.Clients) error {
	return nil
}

func (e *DevEnvImpl) Flags() map[string]string {
	return map[string]string{
		"v":                      "10",
		"enable-authz":           "false",
		"debug":                  "false",
		"enable-mock":            "true",
		"enable-https":           "false",
		"enable-metrics-https":   "false",
		"api-server-hostname":    "localhost",
		"api-server-bindaddress": "localhost:8000",
		"enable-sentry":          "false",
		"cors-allowed-origins":   "http://localhost:3000,http://localhost:8080",
	}
}
