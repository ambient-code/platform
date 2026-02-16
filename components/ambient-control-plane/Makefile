BINARY_NAME=ambient-control-plane
CONTAINER_ENGINE?=$(shell command -v podman 2>/dev/null || echo docker)

git_sha:=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
git_dirty:=$(shell git diff --quiet 2>/dev/null || echo "-modified")
build_version:=$(git_sha)$(git_dirty)
build_time:=$(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
ldflags=-X main.version=$(build_version) -X 'main.buildTime=$(build_time)'

.PHONY: binary
binary:
	@echo "Building $(BINARY_NAME) version: $(build_version)"
	go build -ldflags="$(ldflags)" -o $(BINARY_NAME) ./cmd/ambient-control-plane

.PHONY: install
install:
	go install -ldflags="$(ldflags)" ./cmd/ambient-control-plane

.PHONY: run
run: binary
	./$(BINARY_NAME)

.PHONY: test
test:
	go test -v -race ./...

.PHONY: lint
lint:
	gofmt -l .
	go vet ./...

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: clean
clean:
	rm -f $(BINARY_NAME)

.PHONY: tidy
tidy:
	go mod tidy
