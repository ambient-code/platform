package environments

import (
	"github.com/openshift-online/rh-trex-ai/pkg/config"
	"github.com/openshift-online/rh-trex-ai/pkg/db/db_session"
	pkgenv "github.com/openshift-online/rh-trex-ai/pkg/environments"
)

const OpenShiftDevEnv = "openshift-dev"

var _ pkgenv.EnvironmentImpl = &OpenShiftDevEnvImpl{}

type OpenShiftDevEnvImpl struct {
	Env *pkgenv.Env
}

func (e *OpenShiftDevEnvImpl) OverrideDatabase(c *pkgenv.Database) error {
	c.SessionFactory = db_session.NewProdFactory(e.Env.Config.Database)
	return nil
}

func (e *OpenShiftDevEnvImpl) OverrideConfig(c *config.ApplicationConfig) error {
	c.Server.CORSAllowedHeaders = []string{"X-Ambient-Project"}
	c.Auth.EnableJWT = false
	c.Auth.JwkCertURLs = []string{}
	c.Auth.JwkCertFile = ""
	return nil
}

func (e *OpenShiftDevEnvImpl) OverrideServices(s *pkgenv.Services) error {
	return nil
}

func (e *OpenShiftDevEnvImpl) OverrideHandlers(h *pkgenv.Handlers) error {
	return nil
}

func (e *OpenShiftDevEnvImpl) OverrideClients(c *pkgenv.Clients) error {
	return nil
}

func (e *OpenShiftDevEnvImpl) Flags() map[string]string {
	return map[string]string{
		"v":            "1",
		"debug":        "false",
		"enable-authz": "false",
		"enable-mock":  "true",
	}
}
