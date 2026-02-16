package test

import (
	"testing"

	gm "github.com/onsi/gomega"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
)

func RegisterIntegration(t *testing.T) (*Helper, *openapi.APIClient) {
	gm.RegisterTestingT(t)
	helper := NewHelper(t)
	helper.DBFactory.ResetDB()
	client := helper.NewApiClient()

	return helper, client
}
