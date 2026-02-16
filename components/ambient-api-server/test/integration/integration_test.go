package integration

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/golang/glog"

	"github.com/ambient/platform/components/ambient-api-server/test"

	// Backend-compatible plugins only
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/projectSettings"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/projects"
	_ "github.com/ambient/platform/components/ambient-api-server/plugins/sessions"
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
