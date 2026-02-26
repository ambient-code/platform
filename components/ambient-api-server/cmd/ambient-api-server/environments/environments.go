package environments

import (
	"os"
	"path/filepath"
	"runtime"

	pkgenv "github.com/openshift-online/rh-trex-ai/pkg/environments"
	"github.com/openshift-online/rh-trex-ai/pkg/trex"
)

const AmbientEnvKey = "AMBIENT_ENV"

func init() {
	if v := os.Getenv(AmbientEnvKey); v != "" {
		_ = os.Setenv(pkgenv.EnvironmentStringKey, v)
	}

	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "../../..")

	trex.Init(trex.Config{
		ServiceName:    "ambient-api-server",
		BasePath:       "/api/ambient/v1",
		ErrorHref:      "/api/ambient/v1/errors/",
		MetadataID:     "ambient-api-server",
		ProjectRootDir: projectRoot,
	})

	env := pkgenv.NewEnvironment(nil)
	env.SetEnvironmentImpls(EnvironmentImpls(env))
}

func EnvironmentImpls(env *pkgenv.Env) map[string]pkgenv.EnvironmentImpl {
	return map[string]pkgenv.EnvironmentImpl{
		pkgenv.DevelopmentEnv:        &DevEnvImpl{Env: env},
		pkgenv.UnitTestingEnv:        &UnitTestingEnvImpl{Env: env},
		pkgenv.IntegrationTestingEnv: &IntegrationTestingEnvImpl{Env: env},
		pkgenv.ProductionEnv:         &ProductionEnvImpl{Env: env},
	}
}

func GetEnvironmentStrFromEnv() string {
	return pkgenv.GetEnvironmentStrFromEnv()
}

func Environment() *Env {
	return pkgenv.Environment()
}
