package main

import (
	"github.com/golang/glog"

	localapi "github.com/ambient/platform/components/ambient-api-server/pkg/api"
	pkgcmd "github.com/openshift-online/rh-trex-ai/pkg/cmd"

	_ "github.com/ambient/platform/components/ambient-api-server/cmd/ambient-api-server/environments"
	_ "github.com/ambient/platform/components/ambient-api-server/pkg/middleware"

	// Core plugins from upstream
	_ "github.com/openshift-online/rh-trex-ai/plugins/events"
	_ "github.com/openshift-online/rh-trex-ai/plugins/generic"
	
	// Backend-compatible plugins only
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/sessions"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/projects" 
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/projectSettings"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/users"
)

func main() {
	rootCmd := pkgcmd.NewRootCommand("ambient-api-server", "Ambient API Server")
	rootCmd.AddCommand(
		pkgcmd.NewMigrateCommand("ambient-api-server"),
		pkgcmd.NewServeCommand(localapi.GetOpenAPISpec),
	)

	if err := rootCmd.Execute(); err != nil {
		glog.Fatalf("error running command: %v", err)
	}
}
