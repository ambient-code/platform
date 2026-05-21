package environments

import (
	"os"

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

const defaultJwkCertURL = "https://sso.redhat.com/auth/realms/redhat-external/protocol/openid-connect/certs"

func (e *ProductionEnvImpl) OverrideConfig(c *config.ApplicationConfig) error {
	c.Server.CORSAllowedHeaders = []string{"X-Ambient-Project"}

	// Priority: CLI flag > env var > default.
	// The framework parses --jwk-cert-url before OverrideConfig runs,
	// so c.Auth.JwkCertURL already holds the flag value (or the flag's
	// built-in default if not explicitly set).
	switch {
	case c.Auth.JwkCertURL != "" && c.Auth.JwkCertURL != defaultJwkCertURL:
		// CLI flag was explicitly set to a non-default value; keep it.
	case os.Getenv("JWK_CERT_URL") != "":
		c.Auth.JwkCertURL = os.Getenv("JWK_CERT_URL")
	default:
		c.Auth.JwkCertURL = defaultJwkCertURL
	}

	c.Auth.JwkCertFile = ""
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
		"v":     "1",
		"debug": "false",
	}
}
