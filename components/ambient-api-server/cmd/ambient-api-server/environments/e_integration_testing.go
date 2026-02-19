package environments

import (
	"os"

	"github.com/openshift-online/rh-trex-ai/pkg/config"
	"github.com/openshift-online/rh-trex-ai/pkg/db/db_session"
	pkgenv "github.com/openshift-online/rh-trex-ai/pkg/environments"
)

var _ pkgenv.EnvironmentImpl = &IntegrationTestingEnvImpl{}

type IntegrationTestingEnvImpl struct {
	Env *pkgenv.Env
}

func (e *IntegrationTestingEnvImpl) OverrideDatabase(c *pkgenv.Database) error {
	mode := os.Getenv("DB_FACTORY_MODE")
	if mode == "external" {
		c.SessionFactory = db_session.NewTestFactory(e.Env.Config.Database)
	} else {
		c.SessionFactory = db_session.NewTestcontainerFactory(e.Env.Config.Database)
	}
	return nil
}

func (e *IntegrationTestingEnvImpl) OverrideConfig(c *config.ApplicationConfig) error {
	if os.Getenv("DB_DEBUG") == "true" {
		c.Database.Debug = true
	}
	c.Server.CORSAllowedHeaders = []string{"X-Ambient-Project"}
	return nil
}

func (e *IntegrationTestingEnvImpl) OverrideServices(s *pkgenv.Services) error {
	return nil
}

func (e *IntegrationTestingEnvImpl) OverrideHandlers(h *pkgenv.Handlers) error {
	return nil
}

func (e *IntegrationTestingEnvImpl) OverrideClients(c *pkgenv.Clients) error {
	return nil
}

func (e *IntegrationTestingEnvImpl) Flags() map[string]string {
	return map[string]string{
		"v":                               "0",
		"logtostderr":                     "true",
		"ocm-base-url":                    "https://api.integration.openshift.com",
		"enable-https":                    "false",
		"enable-metrics-https":            "false",
		"enable-authz":                    "true",
		"ocm-debug":                       "false",
		"enable-ocm-mock":                 "true",
		"enable-sentry":                   "false",
		"api-server-bindaddress":          "localhost:0",
		"metrics-server-bindaddress":      "localhost:0",
		"health-check-server-bindaddress": "localhost:0",
	}
}
