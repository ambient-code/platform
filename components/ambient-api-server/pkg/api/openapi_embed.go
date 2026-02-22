package api

import (
	"embed"
	"io/fs"
)

//go:embed openapi/api/openapi.yaml
var openapiFS embed.FS

func GetOpenAPISpec() ([]byte, error) {
	return fs.ReadFile(openapiFS, "openapi/api/openapi.yaml")
}
