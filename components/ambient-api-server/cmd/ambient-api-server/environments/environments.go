package environments

import (
	"path/filepath"
	"runtime"

	pkgenv "github.com/openshift-online/rh-trex-ai/pkg/environments"
	"github.com/openshift-online/rh-trex-ai/pkg/trex"
)

func init() {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "../../..")

	trex.Init(trex.Config{
		ServiceName:    "ambient-api-server",
		BasePath:       "/api/ambient-api-server/v1",
		ErrorHref:      "/api/ambient-api-server/v1/errors/",
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
