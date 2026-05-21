package integration

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/golang/glog"

	"github.com/ambient-code/platform/components/ambient-api-server/test"

	// Backend-compatible plugins only
	_ "github.com/ambient-code/platform/components/ambient-api-server/plugins/projectSettings"
	_ "github.com/ambient-code/platform/components/ambient-api-server/plugins/projects"
	_ "github.com/ambient-code/platform/components/ambient-api-server/plugins/sessions"
	_ "github.com/openshift-online/rh-trex-ai/plugins/events"
	_ "github.com/openshift-online/rh-trex-ai/plugins/generic"
)

var testHelper *test.Helper

func TestMain(m *testing.M) {
	flag.Parse()
	glog.Infof("Starting integration test using go version %s", runtime.Version())
	testHelper = test.NewHelper(&testing.T{})
	exitCode := m.Run()
	testHelper.Teardown()
	os.Exit(exitCode)
}

func TestServerStarts(t *testing.T) {
	if testHelper == nil {
		t.Fatal("test helper not initialized — TestMain did not run")
	}
	url := testHelper.RestURL("/api/ambient")
	if url == "" {
		t.Fatal("server URL is empty — server did not start")
	}
}
