package integration

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/golang/glog"

	"github.com/ambient/platform/components/ambient-api-server/test"

	_ "github.com/ambient/platform/components/ambient-api-server/plugins/agents"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/permissions"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/projectKeys"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/projectSettings"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/projects"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/repositoryRefs"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/sessions"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/skills"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/tasks"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/users"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/workflowSkills"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/workflowTasks"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/workflows"
	_ "github.com/openshift-online/rh-trex-ai/plugins/events"
	_ "github.com/openshift-online/rh-trex-ai/plugins/generic"
)

func TestMain(m *testing.M) {
	flag.Parse()
	glog.Infof("Starting integration test using go version %s", runtime.Version())
	helper := test.NewHelper(&testing.T{})
	exitCode := m.Run()
	helper.Teardown()
	os.Exit(exitCode)
}
