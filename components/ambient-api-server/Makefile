BINARY_NAME=ambient-api-server
CONTAINER_ENGINE?=$(shell command -v podman 2>/dev/null || echo docker)

# Version information for ldflags
git_sha:=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
git_dirty:=$(shell git diff --quiet 2>/dev/null || echo "-modified")
build_version:=$(git_sha)$(git_dirty)
build_time:=$(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
ldflags=-X github.com/ambient/platform/components/ambient-api-server/pkg/api.Version=$(build_version) -X 'github.com/ambient/platform/components/ambient-api-server/pkg/api.BuildTime=$(build_time)'

.PHONY: binary
binary:
	@echo "Building version: $(build_version)"
	go build -ldflags="$(ldflags)" -o $(BINARY_NAME) ./cmd/ambient-api-server

.PHONY: install
install:
	go install -ldflags="$(ldflags)" ./cmd/ambient-api-server

.PHONY: run
run: binary
	./$(BINARY_NAME) migrate
	./$(BINARY_NAME) serve

.PHONY: run-no-auth
run-no-auth: binary
	./$(BINARY_NAME) migrate
	./$(BINARY_NAME) serve --enable-authz=false --enable-jwt=false

.PHONY: test
test: install
	AMBIENT_ENV=integration_testing go test -p 1 -v ./...

.PHONY: test-integration
test-integration: install
	AMBIENT_ENV=integration_testing go test -p 1 -v ./test/integration/...

.PHONY: proto
proto:
	cd proto && buf generate

.PHONY: proto-lint
proto-lint:
	cd proto && buf lint

.PHONY: proto-breaking
proto-breaking:
	cd proto && buf breaking --against '.git#subdir=proto'

.PHONY: proto-clean
proto-clean:
	rm -rf pkg/api/grpc/

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)

.PHONY: generate
generate:
	rm -rf pkg/api/openapi
	$(CONTAINER_ENGINE) build -t ambient-openapi -f Dockerfile.openapi .
	id=$$($(CONTAINER_ENGINE) create ambient-openapi) && \
	$(CONTAINER_ENGINE) cp $$id:/local/pkg/api/openapi ./pkg/api/openapi && \
	$(CONTAINER_ENGINE) rm $$id

.PHONY: db/setup
db/setup:
	$(CONTAINER_ENGINE) run -d --name ambient-api-server-postgres \
		-p 5432:5432 \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_DB=ambient_api_server \
		postgres:13

.PHONY: db/teardown
db/teardown:
	$(CONTAINER_ENGINE) stop ambient-api-server-postgres || true
	$(CONTAINER_ENGINE) rm ambient-api-server-postgres || true

.PHONY: db/login
db/login:
	$(CONTAINER_ENGINE) exec -it ambient-api-server-postgres psql -U postgres -d ambient_api_server

.PHONY: migrate
migrate: binary
	@if ! $(CONTAINER_ENGINE) ps --filter name=ambient-api-server-postgres --format "{{.Names}}" | grep -q ambient-api-server-postgres; then \
		echo "Database container not running, starting it..."; \
		$(MAKE) db/setup; \
		echo "Waiting for database to be ready..."; \
		sleep 5; \
	fi
	@if [ ! -f secrets/db.password ]; then \
		mkdir -p secrets; \
		echo "postgres" > secrets/db.password; \
		echo "Created default db.password file"; \
	fi
	./$(BINARY_NAME) migrate

.PHONY: build-image
build-image: binary
	@echo "Building container image..."
	$(CONTAINER_ENGINE) build -t vteam_api_server:latest .
