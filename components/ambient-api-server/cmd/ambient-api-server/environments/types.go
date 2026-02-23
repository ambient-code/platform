package environments

import (
	pkgenv "github.com/openshift-online/rh-trex-ai/pkg/environments"
)

const (
	UnitTestingEnv        = pkgenv.UnitTestingEnv
	IntegrationTestingEnv = pkgenv.IntegrationTestingEnv
	DevelopmentEnv        = pkgenv.DevelopmentEnv
	ProductionEnv         = pkgenv.ProductionEnv

	EnvironmentStringKey = pkgenv.EnvironmentStringKey
	EnvironmentDefault   = pkgenv.EnvironmentDefault
)

type Env = pkgenv.Env
type ApplicationConfig = pkgenv.ApplicationConfig
type Database = pkgenv.Database
type Handlers = pkgenv.Handlers
type Services = pkgenv.Services
type Clients = pkgenv.Clients
type ConfigDefaults = pkgenv.ConfigDefaults
type EnvironmentImpl = pkgenv.EnvironmentImpl
