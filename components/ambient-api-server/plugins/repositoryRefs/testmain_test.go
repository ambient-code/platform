package repositoryRefs_test

import (
	"flag"
	"os"
	"runtime"
	"testing"

	"github.com/golang/glog"

	"github.com/ambient/platform/components/ambient-api-server/test"
)

func TestMain(m *testing.M) {
	flag.Parse()
	glog.Infof("Starting repositoryRefs integration test using go version %s", runtime.Version())
	helper := test.NewHelper(&testing.T{})
	exitCode := m.Run()
	helper.Teardown()
	os.Exit(exitCode)
}
