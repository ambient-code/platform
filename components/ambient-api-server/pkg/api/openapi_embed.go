package api

func GetOpenAPISpec() ([]byte, error) {
	spec := `
openapi: 3.0.0
info:
  title: Ambient API Server
  version: 1.0.0
paths:
  /health:
    get:
      summary: Health check
      responses:
        '200':
          description: Service is healthy
`
	return []byte(spec), nil
}
